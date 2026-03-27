// Package chunker provides file chunking strategies for the CROM compression system.
// Chunks are the fundamental unit of processing: each chunk is compared against the
// Codebook to find the closest matching pattern.
package chunker

// DefaultChunkSize is the default size for fixed chunking (128 bytes).
const DefaultChunkSize = 128

// Chunk represents a single fragment of the original file.
type Chunk struct {
	// Data contains the raw bytes of this chunk.
	Data []byte

	// Offset is the byte position of this chunk in the original file.
	Offset uint64

	// Size is the number of bytes in this chunk (may be < chunk size for the last chunk).
	Size uint32

	// Hash is a fast non-cryptographic hash (xxhash) for quick comparison.
	Hash uint64
}

// Chunker defines the interface for splitting data into chunks.
type Chunker interface {
	// Split divides data into a slice of Chunks.
	Split(data []byte) []Chunk
}

// Reassemble concatenates chunks back into the original byte stream.
// The chunks must be in order by offset.
func Reassemble(chunks []Chunk) []byte {
	if len(chunks) == 0 {
		return nil
	}

	// Calculate total size from chunks
	totalSize := uint64(0)
	for _, c := range chunks {
		totalSize += uint64(c.Size)
	}

	result := make([]byte, 0, totalSize)
	for _, c := range chunks {
		result = append(result, c.Data...)
	}

	return result
}
