//go:build !js
// +build !js

package codebook

import (
	"fmt"
	"os"
	"syscall"
)

// Open opens a .cromdb file and maps it into memory.
// The file is opened read-only and mapped with MAP_SHARED | PROT_READ.
func Open(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("codebook: open file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("codebook: stat file: %w", err)
	}

	size := info.Size()
	if size < HeaderSize {
		f.Close()
		return nil, fmt.Errorf("codebook: file too small: %d bytes (minimum %d)", size, HeaderSize)
	}

	// mmap: map the entire file into virtual address space.
	// Pages are loaded on demand by the OS kernel (page faults → disk reads).
	// This means a 50GB codebook only uses ~200MB of RAM (hot pages).
	data, err := syscall.Mmap(
		int(f.Fd()),
		0,
		int(size),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("codebook: mmap failed: %w", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		syscall.Munmap(data)
		f.Close()
		return nil, fmt.Errorf("codebook: parse header: %w", err)
	}

	// Validate that the file is large enough for all declared codewords
	expectedSize := header.DataOffset + uint64(header.CodewordSize)*header.CodewordCount
	if uint64(size) < expectedSize {
		syscall.Munmap(data)
		f.Close()
		return nil, fmt.Errorf(
			"codebook: file truncated: size=%d, expected at least %d for %d codewords",
			size, expectedSize, header.CodewordCount,
		)
	}

	return &Reader{
		file:   f,
		data:   data,
		header: header,
	}, nil
}

// Close unmaps the memory region and closes the underlying file.
func (r *Reader) Close() error {
	if r.data != nil {
		if r.file != nil {
			if err := syscall.Munmap(r.data); err != nil {
				r.file.Close()
				return fmt.Errorf("codebook: munmap failed: %w", err)
			}
		}
		r.data = nil
	}
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// OpenFromBytes creates a Reader from raw bytes in memory (no mmap).
// This is used by the WASM target where filesystem access is unavailable.
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
		file:   nil, // No underlying file
		data:   data,
		header: header,
	}, nil
}
