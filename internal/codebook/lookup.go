package codebook

import (
	"fmt"
)

// Lookup returns the raw bytes of the codeword at the given ID.
// This is an O(1) direct access operation: offset = DataOffset + (id × CodewordSize).
// The returned slice is a view into the mmap'd region — do NOT modify it.
func (r *Reader) Lookup(id uint64) ([]byte, error) {
	if id >= r.header.CodewordCount {
		return nil, fmt.Errorf(
			"codebook: lookup out of bounds: id=%d, count=%d",
			id, r.header.CodewordCount,
		)
	}

	cwSize := uint64(r.header.CodewordSize)
	offset := r.header.DataOffset + (id * cwSize)
	end := offset + cwSize

	// Safety check (should not happen if Open validated the file size)
	if end > uint64(len(r.data)) {
		return nil, fmt.Errorf(
			"codebook: lookup would read past end of file: offset=%d, end=%d, file_size=%d",
			offset, end, len(r.data),
		)
	}

	return r.data[offset:end], nil
}

// LookupUnsafe returns the codeword at the given ID without bounds checking.
// Only use this in hot paths where the ID has already been validated.
func (r *Reader) LookupUnsafe(id uint64) []byte {
	cwSize := uint64(r.header.CodewordSize)
	offset := r.header.DataOffset + (id * cwSize)
	return r.data[offset : offset+cwSize]
}
