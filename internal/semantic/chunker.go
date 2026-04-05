package semantic

import (
	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/cespare/xxhash/v2"
)

// ContextualChunker implements the chunker.Chunker interface but splits data
// based on semantic structures (AST elements, lines, or JSON tokens).
type ContextualChunker struct {
	fileType string
	maxSize  int
	minSize  int
}

// NewContextualChunker creates a new Semantic chunker.
func NewContextualChunker(fileType string, maxSize int) *ContextualChunker {
	return &ContextualChunker{
		fileType: fileType,
		maxSize:  maxSize,
		minSize:  32, // Minimum chunk size to avoid generating too many 1-byte chunks
	}
}

// Split divides data by applying content-aware heuristics.
func (s *ContextualChunker) Split(data []byte) []chunker.Chunk {
	if len(data) == 0 {
		return nil
	}

	var chunks []chunker.Chunk
	var offset uint64 = 0

	// Strategy: find delimiters based on file type.
	// For JSON, we use '{', '}' or ',' to define natural node boundaries.
	// For LINES / JSONL, we use '\n'.
	var delim byte = '\n'
	if s.fileType == "JSON" {
		delim = '}'
	} else if s.fileType == "JSONL" {
		delim = '\n'
	} else if s.fileType == "UNKNOWN" || s.fileType == "ELF" || s.fileType == "ZIP" || s.fileType == "PNG" {
		// Fallback to strict sizing for unstructured binary
		return chunker.NewFixedChunker(s.maxSize).Split(data)
	}

	left := 0
	length := len(data)

	for left < length {
		right := left + s.minSize
		if right >= length {
			right = length
		} else {
			// Scan forward from minSize to maxSize to find the delimiter (Rabin-Karp inspired Boundary)
			found := false
			maxRight := left + s.maxSize
			if maxRight > length {
				maxRight = length
			}
			
			for i := right; i < maxRight; i++ {
				if data[i] == delim {
					// Include the delimiter in the chunk
					right = i + 1
					found = true
					break
				}
				// Secondary JSON delimiter
				if s.fileType == "JSON" && data[i] == ',' {
					right = i + 1
					found = true
					break
				}
			}
			
			if !found {
				// If no delimiter was found in the contextual window, force a cut.
				right = maxRight
			}
		}

		chunkData := data[left:right]
		chunks = append(chunks, chunker.Chunk{
			Data:   chunkData,
			Offset: offset,
			Size:   uint32(len(chunkData)),
			Hash:   xxhash.Sum64(chunkData),
		})
		
		offset += uint64(len(chunkData))
		left = right
	}
	return chunks
}
