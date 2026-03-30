package codebook

import "os"

// Reader provides read-only access to a .cromdb file.
// On native targets, the data comes from mmap. On WASM, it's read into memory.
type Reader struct {
	file   *os.File
	data   []byte // mmap'd region (depreciating) or in-memory config for WASM
	header *Header

	// V20: Paging B-Tree / LRU Cache mechanism for 50GB Codebooks
	pageSize int
	lruCache map[uint64][]byte
	pageReqs uint64
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
