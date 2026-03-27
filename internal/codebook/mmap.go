package codebook

import (
	"fmt"
	"os"
	"syscall"
)

// Reader provides read-only access to a .cromdb file via memory mapping.
// The entire file is mapped into virtual address space but only accessed pages
// are loaded into RAM by the OS kernel (demand paging).
type Reader struct {
	file   *os.File
	data   []byte // mmap'd region
	header *Header
}

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

// Header returns the parsed header of the codebook.
func (r *Reader) Header() *Header {
	return r.header
}

// CodewordCount returns the number of codewords in the codebook.
func (r *Reader) CodewordCount() uint64 {
	return r.header.CodewordCount
}

// CodewordSize returns the size of each codeword in bytes.
func (r *Reader) CodewordSize() uint16 {
	return r.header.CodewordSize
}

// BuildHash returns the SHA-256 hash of the codeword data section.
func (r *Reader) BuildHash() [BuildHashSize]byte {
	return r.header.BuildHash
}

// Close unmaps the memory region and closes the underlying file.
func (r *Reader) Close() error {
	if r.data != nil {
		if err := syscall.Munmap(r.data); err != nil {
			r.file.Close()
			return fmt.Errorf("codebook: munmap failed: %w", err)
		}
		r.data = nil
	}
	return r.file.Close()
}
