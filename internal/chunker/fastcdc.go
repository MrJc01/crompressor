package chunker

import (
	"github.com/cespare/xxhash/v2"
)

// gearTable is a precomputed table of 256 random 64-bit integers for Gear Hash.
var gearTable [256]uint64

func init() {
	// Initialize gear table with pseudo-random numbers
	h := xxhash.New()
	for i := 0; i < 256; i++ {
		h.Write([]byte{byte(i)})
		gearTable[i] = h.Sum64()
		h.Reset()
	}
}

type FastCDCChunker struct {
	targetSize int
	minSize    int
	maxSize    int
	mask       uint64
}

func NewFastCDCChunker(targetSize int) *FastCDCChunker {
	if targetSize <= 0 {
		targetSize = DefaultChunkSize
	}
	
	// FastCDC uses a mask to find boundaries where hash & mask == 0
	mask := uint64(targetSize) - 1

	return &FastCDCChunker{
		targetSize: targetSize,
		minSize:    targetSize / 4,
		maxSize:    targetSize * 4,
		mask:       mask,
	}
}

func (c *FastCDCChunker) Split(data []byte) []Chunk {
	if len(data) == 0 {
		return nil
	}

	n := len(data)
	if n <= c.minSize {
		return []Chunk{makeChunk(data, 0, n)}
	}

	var chunks []Chunk
	start := 0
	
	var hash uint64

	for i := 0; i < n; i++ {
		hash = (hash << 1) + gearTable[data[i]]
		
		chunkLen := i - start + 1
		
		if chunkLen >= c.minSize {
			if chunkLen >= c.maxSize || (hash&c.mask) == 0 {
				chunks = append(chunks, makeChunk(data, start, i+1))
				start = i + 1
				hash = 0
			}
		}
	}

	if start < n {
		chunks = append(chunks, makeChunk(data, start, n))
	}

	return chunks
}
