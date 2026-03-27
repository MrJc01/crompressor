package chunker

import (
	"github.com/cespare/xxhash/v2"
)

// FixedChunker splits data into fixed-size blocks.
// The last chunk may be smaller than ChunkSize if the data length is not evenly divisible.
type FixedChunker struct {
	// ChunkSize is the size of each chunk in bytes.
	ChunkSize int
}

// NewFixedChunker creates a FixedChunker with the given block size.
// If chunkSize <= 0, DefaultChunkSize (128) is used.
func NewFixedChunker(chunkSize int) *FixedChunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &FixedChunker{ChunkSize: chunkSize}
}

// Split divides data into fixed-size chunks.
// Each chunk includes its offset in the original data, its size, and an xxhash digest.
func (fc *FixedChunker) Split(data []byte) []Chunk {
	if len(data) == 0 {
		return nil
	}

	numChunks := (len(data) + fc.ChunkSize - 1) / fc.ChunkSize
	chunks := make([]Chunk, 0, numChunks)

	for offset := 0; offset < len(data); offset += fc.ChunkSize {
		end := offset + fc.ChunkSize
		if end > len(data) {
			end = len(data)
		}

		block := data[offset:end]

		chunks = append(chunks, Chunk{
			Data:   block,
			Offset: uint64(offset),
			Size:   uint32(end - offset),
			Hash:   xxhash.Sum64(block),
		})
	}

	return chunks
}
