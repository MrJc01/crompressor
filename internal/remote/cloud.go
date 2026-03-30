package remote

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// CloudReader implements an io.ReaderAt and io.Reader interface over HTTP.
// This allows Remote FUSE mounting and Neural Grep via HTTP Range Requests (S3, Minio, CDNs)
// without downloading the entire .crom payload.
type CloudReader struct {
	url    string
	client *http.Client
	offset int64
	size   int64
}

// NewCloudReader initializes a secure HTTP client to lazily load ranges of a .crom file.
func NewCloudReader(url string) (*CloudReader, error) {
	// Send a HEAD request to verify file existence and get Content-Length
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("remote: invalid url: %w", err)
	}

	client := &http.Client{}
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

	return &CloudReader{
		url:    url,
		client: client,
		offset: 0,
		size:   size,
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

	end := off + bytesToRead - 1

	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("invalid response status: %d", resp.StatusCode)
	}

	// Read exactly bytesToRead
	return io.ReadFull(resp.Body, p[:bytesToRead])
}
