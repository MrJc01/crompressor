package chunker

import (
	"github.com/cespare/xxhash/v2"
)

const (
	// CDCWindowSize corresponds to the rolling hash window.
	CDCWindowSize = 8
)

// Rabin-Karp inspired rolling hash parameters
const (
	prime64 = 1099511628211
)

var primePower uint64

func init() {
	primePower = 1
	for i := 0; i < CDCWindowSize; i++ {
		primePower *= prime64
	}
}

// CDCChunker implements Content-Defined Chunking using a simple rolling hash.
// It finds boundaries where `(hash % targetSize) == 0`.
type CDCChunker struct{
	targetSize int
	minSize    int
	maxSize    int
}

func NewCDCChunker(targetSize int) *CDCChunker {
	return &CDCChunker{
		targetSize: targetSize,
		minSize:    targetSize / 4,
		maxSize:    targetSize * 2,
	}
}

// Split divides the data into chunks based on data content to resist byte-shifting.
func (c *CDCChunker) Split(data []byte) []Chunk {
	if len(data) == 0 {
		return nil
	}

	var chunks []Chunk
	n := len(data)

	// If data is smaller than min size, just return it as a single chunk
	if n <= c.minSize {
		return []Chunk{makeChunk(data, 0, n)}
	}

	start := 0
	offset := 0

	var rollHash uint64

	for offset < n {
		// Calculate precise chunk length so far
		chunkLen := offset - start

		// Force boundary if we reach MaxSize
		if chunkLen >= c.maxSize {
			chunks = append(chunks, makeChunk(data, start, offset))
			start = offset
			rollHash = 0
			continue
		}

		// Update rolling hash
		if offset >= start+CDCWindowSize {
			oldByte := uint64(data[offset-CDCWindowSize])
			rollHash = rollHash*prime64 + uint64(data[offset]) - oldByte*primePower
		} else {
			rollHash = rollHash*prime64 + uint64(data[offset])
		}

		// Check boundary condition: only if we passed MinSize
		if chunkLen >= c.minSize && offset >= start+CDCWindowSize {
			if rollHash%uint64(c.targetSize) == 0 {
				chunks = append(chunks, makeChunk(data, start, offset+1))
				start = offset + 1
				rollHash = 0
			}
		}

		offset++
	}

	// Deal with remaining data
	if start < n {
		chunks = append(chunks, makeChunk(data, start, n))
	}

	return chunks
}

func makeChunk(data []byte, start, end int) Chunk {
	slice := data[start:end]
	return Chunk{
		Data:   slice,
		Offset: uint64(start), // Offset within the current slice context
		Size:   uint32(len(slice)),
		Hash:   xxhash.Sum64(slice),
	}
}
