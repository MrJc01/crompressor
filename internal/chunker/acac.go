package chunker

import (
	"bytes"

	"github.com/cespare/xxhash/v2"
)

// SemanticChunker splits data based on byte delimiters (like newlines)
// while adhering to a maximum chunk size fallback to avoid OOM or 
// CPU starvation on extremely long unbroken lines.
type SemanticChunker struct {
	delimiter byte
	maxSize   int
}

// NewSemanticChunker returns an ACAC instance keyed for JSON Lines or Logs.
func NewSemanticChunker(delimiter byte, maxSize int) *SemanticChunker {
	if maxSize <= 0 {
		maxSize = 1024 // 1KB max unbroken line limit
	}
	return &SemanticChunker{
		delimiter: delimiter,
		maxSize:   maxSize,
	}
}

// Split divides the incoming buffer into strictly semantic blocks.
func (c *SemanticChunker) Split(data []byte) []Chunk {
	if len(data) == 0 {
		return nil
	}

	var chunks []Chunk
	start := 0
	n := len(data)

	// Optimization: Allocate an initial capacity assuming avg 128 byte lines
	chunks = make([]Chunk, 0, n/128+1)

	for start < n {
		end := start + c.maxSize
		if end > n {
			end = n
		}

		// Find the true semantic boundary (the nearest newline)
		delimIdx := bytes.IndexByte(data[start:end], c.delimiter)
		
		var chunkLen int
		if delimIdx != -1 {
			// Include the newline in the chunk
			chunkLen = delimIdx + 1
		} else {
			// If no newline is found within maxSize, we forcefully cut at MaxSize
			// (Hard fallback like FixedChunker for safety)
			chunkLen = end - start
		}

		// Hasher is non-cryptographic, used only for in-memory deduplication tracking
		cData := data[start : start+chunkLen]
		hash := xxhash.Sum64(cData)

		chunks = append(chunks, Chunk{
			Data:   cData,
			Offset: uint64(start),
			Size:   uint32(chunkLen),
			Hash:   hash,
		})

		start += chunkLen
	}

	return chunks
}
