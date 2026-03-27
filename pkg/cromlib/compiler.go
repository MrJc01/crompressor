// Package cromlib orchestrates the complete encode/decode pipeline for CROM.
package cromlib

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/crom-project/crom/internal/chunker"
	"github.com/crom-project/crom/internal/codebook"
	"github.com/crom-project/crom/internal/crypto"
	"github.com/crom-project/crom/internal/delta"
	"github.com/crom-project/crom/internal/search"
	"github.com/crom-project/crom/pkg/format"
)

// PackOptions defines the compiler settings.
type PackOptions struct {
	Concurrency   int
	EncryptionKey string // Passphrase for AES-256-GCM. If empty, no encryption.
	// Callback for progress bar integration, called with bytes processed
	OnProgress func(bytesProcessed int)
}

// Metrics holds the output telemetry of the compilation process.
type Metrics struct {
	OriginalSize uint64
	PackedSize   uint64
	Duration     time.Duration
	HitRate      float64 // Percentage of chunks perfectly matching or with < 50% bit delta
}

// DefaultPackOptions returns sensible defaults.
func DefaultPackOptions() PackOptions {
	return PackOptions{
		Concurrency: 4,
		OnProgress:  func(n int) {},
	}
}

// Memory block size to process per batch (16 MB)
const BlockSize = 16 * 1024 * 1024

// Pack reads an input file iteratively, searches the spatial index in parallel,
// and streams compressed/encrypted deltas to disk in block frames without blowing up RAM.
func Pack(inputPath, outputPath, codebookPath string, opts PackOptions) (*Metrics, error) {
	start := time.Now()

	opts.OnProgress(0)

	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return nil, fmt.Errorf("pack: failed to open codebook: %w", err)
	}
	defer cb.Close()

	searcher := search.NewLSHSearcher(cb)

	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("pack: open input file: %w", err)
	}
	defer inFile.Close()

	info, err := inFile.Stat()
	if err != nil {
		return nil, err
	}
	originalSize := uint64(info.Size())
	numExpectedChunks := uint32((originalSize + chunker.DefaultChunkSize - 1) / chunker.DefaultChunkSize)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	// 1. Setup Header
	header := &format.Header{
		Version:      format.Version2,
		OriginalSize: originalSize,
		ChunkCount:   numExpectedChunks,
	}

	var derivedKey []byte
	if opts.EncryptionKey != "" {
		header.IsEncrypted = true
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return nil, fmt.Errorf("pack: generate salt: %w", err)
		}
		copy(header.Salt[:], salt)
		derivedKey = crypto.DeriveKey([]byte(opts.EncryptionKey), salt)
	}

	headerBytes := header.Serialize()

	// 2. Write Dummy Space for Header, BlockTable, and ChunkTable
	if _, err := outFile.Write(headerBytes); err != nil {
		return nil, err
	}

	numBlocks := header.NumBlocks()
	blockTableSpace := make([]byte, numBlocks*4)
	if _, err := outFile.Write(blockTableSpace); err != nil {
		return nil, err
	}

	chunkTableSize := numExpectedChunks * format.EntrySize
	if header.IsEncrypted {
		chunkTableSize += 28 // AES-GCM overhead
	}
	chunkTableSpace := make([]byte, chunkTableSize)
	if _, err := outFile.Write(chunkTableSpace); err != nil {
		return nil, err
	}

	// 3. Process Stream
	hasher := sha256.New()
	fc := chunker.NewFixedChunker(chunker.DefaultChunkSize)

	var finalEntries []format.ChunkEntry
	var blockTable []uint32

	currentOffset := uint64(0)
	buf := make([]byte, BlockSize)

	var hitCount int

	for {
		n, errRead := inFile.Read(buf)
		if n > 0 {
			blockData := buf[:n]
			hasher.Write(blockData)
			opts.OnProgress(n)

			chunks := fc.Split(blockData)
			numChunks := len(chunks)

			type processedChunk struct {
				entry format.ChunkEntry
				res   []byte
				err   error
			}
			results := make([]processedChunk, numChunks)
			jobs := make(chan int, numChunks)
			var wg sync.WaitGroup

			for w := 0; w < opts.Concurrency; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						chunk := chunks[i]
						match, err := searcher.FindBestMatch(chunk.Data)
						if err != nil {
							results[i] = processedChunk{err: err}
							continue
						}

						residual := delta.XOR(chunk.Data, match.Pattern)

						results[i] = processedChunk{
							res: residual,
							entry: format.ChunkEntry{
								CodebookID:   match.CodebookID,
								DeltaSize:    uint32(len(residual)),
								OriginalSize: uint32(chunk.Size),
							},
						}
					}
				}()
			}

			for i := 0; i < numChunks; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()

			// Gather the residuals for this Block
			var blockPlainDeltas []byte

			for i := 0; i < numChunks; i++ {
				res := results[i]
				if res.err != nil {
					return nil, res.err
				}

				res.entry.DeltaOffset = currentOffset
				finalEntries = append(finalEntries, res.entry)

				blockPlainDeltas = append(blockPlainDeltas, res.res...)
				currentOffset += uint64(len(res.res))

				if res.entry.DeltaSize > 0 {
					zeroes := 0
					for _, b := range res.res {
						if b == 0 {
							zeroes++
						}
					}
					if zeroes > (int(res.entry.DeltaSize) * 95 / 100) {
						hitCount++
					}
				}
			}

			// Compress this independent block
			compBlock, err := delta.CompressPool(blockPlainDeltas)
			if err != nil {
				return nil, fmt.Errorf("pack: compress block: %w", err)
			}

			// Encrypt if required
			if header.IsEncrypted {
				compBlock, err = crypto.Encrypt(derivedKey, compBlock)
				if err != nil {
					return nil, fmt.Errorf("pack: encrypt block: %w", err)
				}
			}

			blockTable = append(blockTable, uint32(len(compBlock)))

			if _, err := outFile.Write(compBlock); err != nil {
				return nil, err
			}
		}

		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			return nil, errRead
		}
	}

	// 4. Finalize
	copy(header.OriginalHash[:], hasher.Sum(nil))

	outFile.Seek(0, 0)

	tableData := format.SerializeChunkTable(finalEntries)
	if header.IsEncrypted {
		encTable, err := crypto.Encrypt(derivedKey, tableData)
		if err != nil {
			return nil, fmt.Errorf("pack: encrypt chunk table: %w", err)
		}
		tableData = encTable
	}

	// We can't use format.Writer.Write directly because we already streamed the DeltaPool.
	// We just write the Metadata manually.
	if _, err := outFile.Write(header.Serialize()); err != nil {
		return nil, err
	}

	// Block Table
	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	if _, err := outFile.Write(blockTableRaw); err != nil {
		return nil, err
	}

	// Chunk Table
	if _, err := outFile.Write(tableData); err != nil {
		return nil, err
	}

	packedInfo, _ := outFile.Stat()

	return &Metrics{
		OriginalSize: originalSize,
		PackedSize:   uint64(packedInfo.Size()),
		Duration:     time.Since(start),
		HitRate:      (float64(hitCount) / float64(numExpectedChunks)) * 100,
	}, nil
}
