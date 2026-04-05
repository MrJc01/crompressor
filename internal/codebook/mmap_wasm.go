//go:build js && wasm
// +build js,wasm

package codebook

import (
	"fmt"
	"os"
)

// Open opens a .cromdb file by reading it entirely into memory.
// In WASM environments, mmap is not available so we fall back to ReadFile.
func Open(path string) (*Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("codebook: read file: %w", err)
	}
	return OpenFromBytes(data)
}

// OpenFromBytes creates a Reader from raw bytes in memory (no mmap).
func OpenFromBytes(data []byte) (*Reader, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("codebook: data too small: %d bytes (minimum %d)", len(data), HeaderSize)
	}

	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("codebook: parse header: %w", err)
	}

	expectedSize := header.DataOffset + uint64(header.CodewordSize)*header.CodewordCount
	if uint64(len(data)) < expectedSize {
		return nil, fmt.Errorf(
			"codebook: data truncated: size=%d, expected at least %d for %d codewords",
			len(data), expectedSize, header.CodewordCount,
		)
	}

	return &Reader{
		file:   nil,
		data:   data,
		header: header,
	}, nil
}

// Close releases the reader resources. In WASM mode there's no mmap to unmap.
func (r *Reader) Close() error {
	r.data = nil
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}
