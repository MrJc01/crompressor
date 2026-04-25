package cromlib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/entropy"
	"github.com/MrJc01/crompressor/internal/search"
	"github.com/MrJc01/crompressor/pkg/format"
)

// PackBytes compresses input data entirely in memory, without touching the filesystem.
// This is the primary entry point for WASM and embedded environments.
//
// Parameters:
//   - input: raw bytes to compress
//   - codebookData: raw .cromdb bytes (loaded via fetch in JS or go:embed)
//   - opts: compression options (same as Pack)
//
// Returns the packed .crom bytes, metrics, and any error.
func PackBytes(input []byte, codebookData []byte, opts PackOptions) ([]byte, *Metrics, error) {
	start := time.Now()

	if len(input) == 0 {
		return nil, nil, fmt.Errorf("packbytes: empty input")
	}

	// 1. Entropy analysis
	eScore := entropy.Shannon(input[:min(len(input), 65536)])

	// Adaptive chunk size
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		if eScore < 3.0 {
			chunkSize = 64
		} else if eScore < 6.0 {
			chunkSize = 128
		} else {
			chunkSize = 512
		}
	}

	// 2. Open codebook from bytes
	cb, err := codebook.OpenFromBytes(codebookData)
	if err != nil {
		return nil, nil, fmt.Errorf("packbytes: open codebook: %w", err)
	}
	defer cb.Close()

	searcher := search.NewLSHSearcher(cb)

	// 3. Chunk the input
	fc := chunker.NewFixedChunker(chunkSize)
	chunks := fc.Split(input)

	originalSize := uint64(len(input))
	originalHash := sha256.Sum256(input)

	// 4. Process each chunk
	var entries []format.ChunkEntry
	var deltaPool []byte
	currentOffset := uint64(0)

	var hitCount int
	var literalCount int
	var totalSimilarity float64

	for _, chunk := range chunks {
		chunkEntropy := entropy.Shannon(chunk.Data)

		// High entropy bypass in archive mode
		if opts.Mode != "edge" && chunkEntropy > 3.00 {
			entries = append(entries, format.ChunkEntry{
				CodebookID:   format.LiteralCodebookID,
				CodebookIndex: format.LiteralCodebookIndex,
				DeltaOffset:  currentOffset,
				DeltaSize:    uint32(len(chunk.Data)),
				OriginalSize: uint32(chunk.Size),
			})
			deltaPool = append(deltaPool, chunk.Data...)
			currentOffset += uint64(len(chunk.Data))
			literalCount++
			continue
		}

		match, err := searcher.FindBestMatch(chunk.Data)
		if err != nil {
			// No match — store as literal
			entries = append(entries, format.ChunkEntry{
				CodebookID:   format.LiteralCodebookID,
				CodebookIndex: format.LiteralCodebookIndex,
				DeltaOffset:  currentOffset,
				DeltaSize:    uint32(len(chunk.Data)),
				OriginalSize: uint32(chunk.Size),
			})
			deltaPool = append(deltaPool, chunk.Data...)
			currentOffset += uint64(len(chunk.Data))
			literalCount++
			continue
		}

		chunkBits := len(chunk.Data) * 8
		sim := match.Similarity(chunkBits)
		totalSimilarity += sim

		usablePattern := match.Pattern
		if len(usablePattern) > len(chunk.Data) {
			usablePattern = usablePattern[:len(chunk.Data)]
		}

		var residual []byte
		codeID := match.CodebookID

		if sim < 0.20 && opts.Mode != "edge" {
			residual = chunk.Data
			codeID = format.LiteralCodebookID
			literalCount++
		} else if opts.Mode == "edge" {
			residual = []byte{} // Lossy: discard delta
		} else {
			residual = delta.XOR(chunk.Data, usablePattern)
		}

		if sim >= 0.95 {
			hitCount++
		}

		entries = append(entries, format.ChunkEntry{
			CodebookID:   codeID,
			CodebookIndex: 0,
			DeltaOffset:  currentOffset,
			DeltaSize:    uint32(len(residual)),
			OriginalSize: uint32(chunk.Size),
		})
		deltaPool = append(deltaPool, residual...)
		currentOffset += uint64(len(residual))
	}

	// 5. Compress the delta pool
	compressedPool, err := delta.CompressPool(deltaPool)
	if err != nil {
		return nil, nil, fmt.Errorf("packbytes: compress pool: %w", err)
	}

	// 6. Build the .crom binary in memory
	header := &format.Header{
		Version:      format.Version2,
		OriginalSize: originalSize,
		ChunkCount:   uint32(len(entries)),
		ChunkSize:    uint32(chunkSize),
	}
	copy(header.OriginalHash[:], originalHash[:])

	cbHash := cb.BuildHash()
	copy(header.CodebookHash[:], cbHash[:8])

	var buf bytes.Buffer

	// Header
	buf.Write(header.Serialize())

	// Block table (single block for in-memory)
	blockSizeBytes := make([]byte, 4)
	blockSizeBytes[0] = byte(len(compressedPool))
	blockSizeBytes[1] = byte(len(compressedPool) >> 8)
	blockSizeBytes[2] = byte(len(compressedPool) >> 16)
	blockSizeBytes[3] = byte(len(compressedPool) >> 24)
	buf.Write(blockSizeBytes)

	// Chunk table
	tableData := format.SerializeChunkTable(entries, header.Version)
	buf.Write(tableData)

	// Compressed delta pool
	buf.Write(compressedPool)

	packedSize := uint64(buf.Len())
	duration := time.Since(start)

	var avgSim float64
	totalChunks := len(entries)
	if totalChunks > 0 {
		avgSim = totalSimilarity / float64(totalChunks)
	}

	m := &Metrics{
		OriginalSize:  originalSize,
		PackedSize:    packedSize,
		Duration:      duration,
		HitRate:       float64(hitCount) / float64(totalChunks) * 100,
		LiteralChunks: literalCount,
		TotalChunks:   totalChunks,
		AvgSimilarity: avgSim,
		Entropy:       eScore,
	}

	return buf.Bytes(), m, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
