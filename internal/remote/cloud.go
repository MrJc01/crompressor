package remote

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Constantes de Tuning do Egress Optimizer: Cache LRU limitando o uso extremo do S3/HTTP.
const (
	PageSize  = 256 * 1024 // 256KB por HTTP Range Chunk
	MaxPages  = 256        // Total = 64MiB na RAM (ideal para Edge computing e media mount)
	PrefetchDepth = 2      // Quantas páginas para a frente devemos puxar anonimamente
)

// CloudReader implements an io.ReaderAt and io.Reader interface over HTTP.
// This allows Remote FUSE mounting and Neural Grep via HTTP Range Requests (S3, Minio, CDNs)
// without downloading the entire .crom payload.
type CloudReader struct {
	url      string
	client   *http.Client
	offset   int64
	size     int64
	cache    *lru.Cache[int64, []byte]
	inFlight sync.Map // Rastreia quais páginas já estão sofrendo download concorrente
}

// NewCloudReader initializes a secure HTTP client to lazily load ranges of a .crom file.
func NewCloudReader(url string) (*CloudReader, error) {
	// Send a HEAD request to verify file existence and get Content-Length
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("remote: invalid url: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: head request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: file error, status code %d", resp.StatusCode)
	}

	size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if size <= 0 {
		return nil, fmt.Errorf("remote: invalid file size (must be greater than 0)")
	}

	cache, err := lru.New[int64, []byte](MaxPages)
	if err != nil {
		return nil, fmt.Errorf("remote: failed to initialize LRU cache: %w", err)
	}

	// Um client enxuto com timeout defensivo contra Zombificações do Prefetch
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &CloudReader{
		url:    url,
		client: httpClient,
		offset: 0,
		size:   size,
		cache:  cache,
	}, nil
}

// Size returns the full remote file size.
func (c *CloudReader) Size() int64 {
	return c.size
}

// Read implements io.Reader sequentially.
func (c *CloudReader) Read(p []byte) (n int, err error) {
	n, err = c.ReadAt(p, c.offset)
	c.offset += int64(n)
	return n, err
}

// ReadAt implements io.ReaderAt for random access via HTTP Range requests.
// Agora acoplado ao poderoso Cache LRU + Async Prefetcher (Egress Optimizer).
func (c *CloudReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= c.size {
		return 0, io.EOF
	}

	bytesToRead := int64(len(p))
	if off+bytesToRead > c.size {
		bytesToRead = c.size - off
	}

	if bytesToRead <= 0 {
		return 0, nil
	}

	startPage := off / PageSize
	endPage := (off + bytesToRead - 1) / PageSize

	bytesRead := 0

	for pageNum := startPage; pageNum <= endPage; pageNum++ {
		pageData, errFetch := c.loadPage(pageNum)
		if errFetch != nil && errFetch != io.EOF {
			return bytesRead, fmt.Errorf("remote: failed to fetch page %d: %w", pageNum, errFetch)
		}

		if pageData == nil || len(pageData) == 0 {
			break // EOF hit in this page
		}

		pageAbsStart := pageNum * PageSize
		copyStart := off + int64(bytesRead) - pageAbsStart
		if copyStart < 0 {
			copyStart = 0
		}

		copyLen := int64(len(pageData)) - copyStart
		if int64(bytesRead)+copyLen > bytesToRead {
			copyLen = bytesToRead - int64(bytesRead)
		}

		copy(p[bytesRead:], pageData[copyStart:copyStart+copyLen])
		bytesRead += int(copyLen)

		if int64(bytesRead) == bytesToRead || int64(len(pageData)) < PageSize { // Fim dos tempos
			break
		}
	}

	// ASYNC PREFETCHER: Identifica linearidade pura se estamos varrendo arquivos
	// Disparamos go-routines ocultas puxando Page+1 e Page+2 pro cache da Memória.
	for i := int64(1); i <= PrefetchDepth; i++ {
		ahead := endPage + i
		if ahead*PageSize < c.size {
			go c.prefetchPhantom(ahead)
		}
	}

	if bytesRead == 0 && err == nil {
		return 0, io.EOF
	}

	return bytesRead, nil
}

// prefetchPhantom engata uma thread silenciosa de pre-load HTTP, evitando redundâncias locais.
func (c *CloudReader) prefetchPhantom(pageNum int64) {
	c.loadPage(pageNum)
}

// loadPage lida simultaneamente com a verificação de LRU, lock contra duplicação
// e o Download brutal da faixa do arquivo no bucket remoto.
func (c *CloudReader) loadPage(pageNum int64) ([]byte, error) {
	// 1. Verificação Instantânea Mágica L1
	if data, hit := c.cache.Get(pageNum); hit {
		return data, nil
	}

	// 2. Flight Control Lock "Sync.Map" (Impede 5 threads bajulando a mesma página 12 num pre-fetch)
	flightKey := pageNum
	inFlightCh := make(chan struct{})
	actualCh, loaded := c.inFlight.LoadOrStore(flightKey, inFlightCh)
	if loaded {
		// Outra goroutine já está baixando isso AGORA. Vamos aguardar pacientemente.
		<-actualCh.(chan struct{})
		if data, hit := c.cache.Get(pageNum); hit {
			return data, nil
		}
		// Se ainda deu miss após a espera, algo deu mto errado. Try fallback below.
	} else {
		// Somos os pioneiros deste Chunk! Trancamos a porta p/ limpar após fetch
		defer func() {
			c.inFlight.Delete(flightKey)
			close(inFlightCh)
		}()
	}

	// 3. Egress Downstream Fetch Real
	startByte := pageNum * PageSize
	if startByte >= c.size {
		return nil, io.EOF
	}

	endByte := startByte + PageSize - 1
	if endByte >= c.size {
		endByte = c.size - 1
	}

	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote status %d for page %d", resp.StatusCode, pageNum)
	}

	pageData, err := io.ReadAll(resp.Body)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("remote EOF/Truncation page %d: %w", pageNum, err)
	}

	// 4. Salvar na Memória LRU
	c.cache.Add(pageNum, pageData)
	return pageData, nil
}
