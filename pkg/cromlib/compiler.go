// Package cromlib orchestrates the complete encode/decode pipeline for CROM.
package cromlib

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/search"
	"github.com/MrJc01/crompressor/pkg/format"
)

// PackOptions defines the compiler settings.
type PackOptions struct {
	Concurrency   int
	EncryptionKey string // Passphrase for AES-256-GCM. If empty, no encryption.
	ChunkSize     int    // Size of the chunks (default 128)
	UseCDC        bool   // If true, uses Content-Defined Chunking instead of FixedChunker
	// Callback for progress bar integration, called with bytes processed
	OnProgress func(bytesProcessed int)
}

// Metrics holds the output telemetry of the compilation process.
type Metrics struct {
	OriginalSize uint64
	PackedSize   uint64
	Duration     time.Duration
	HitRate      float64 // Percentage of chunks perfectly matching or with < 50% bit delta
}

// DefaultPackOptions returns sensible defaults.
func DefaultPackOptions() PackOptions {
	return PackOptions{
		Concurrency: 4,
		ChunkSize:   chunker.DefaultChunkSize,
		UseCDC:      false,
		OnProgress:  func(n int) {},
	}
}

// Memory block size to process per batch (16 MB)
const BlockSize = 16 * 1024 * 1024

// Pack reads an input file iteratively, searches the spatial index in parallel,
// and streams compressed/encrypted deltas to disk in block frames without blowing up RAM.
func Pack(inputPath, outputPath, codebookPath string, opts PackOptions) (*Metrics, error) {
	start := time.Now()

	opts.OnProgress(0)

	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return nil, fmt.Errorf("pack: failed to open codebook: %w", err)
	}
	defer cb.Close()

	searcher := search.NewLSHSearcher(cb)

	inFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("pack: open input file: %w", err)
	}
	defer inFile.Close()

	info, err := inFile.Stat()
	if err != nil {
		return nil, err
	}
	originalSize := uint64(info.Size())
	// Pre-calculate an estimate for dummy space allocation.
	// The REAL chunk count will be set after the processing loop.
	numEstimatedChunks := uint32((originalSize + uint64(opts.ChunkSize) - 1) / uint64(opts.ChunkSize))

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	// 1. Setup Header
	header := &format.Header{
		Version:      format.Version2,
		OriginalSize: originalSize,
		ChunkCount:   numEstimatedChunks,
	}

	var derivedKey []byte
	if opts.EncryptionKey != "" {
		header.IsEncrypted = true
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return nil, fmt.Errorf("pack: generate salt: %w", err)
		}
		copy(header.Salt[:], salt)
		derivedKey = crypto.DeriveKey([]byte(opts.EncryptionKey), salt)
	}

	headerBytes := header.Serialize()

	// 2. Write Dummy Space for Header, BlockTable, and ChunkTable
	if _, err := outFile.Write(headerBytes); err != nil {
		return nil, err
	}

	numBlocks := header.NumBlocks()
	blockTableSpace := make([]byte, numBlocks*4)
	if _, err := outFile.Write(blockTableSpace); err != nil {
		return nil, err
	}

	chunkTableSize := numEstimatedChunks * format.EntrySize
	if header.IsEncrypted {
		chunkTableSize += 28 // AES-GCM overhead
	}
	chunkTableSpace := make([]byte, chunkTableSize)
	if _, err := outFile.Write(chunkTableSpace); err != nil {
		return nil, err
	}

	// Remember the offset where the Delta Pool starts, so we can
	// truncate and rewrite if the estimated sizes were wrong.
	deltaPoolStartOffset := int64(len(headerBytes)) + int64(len(blockTableSpace)) + int64(len(chunkTableSpace))

	// 3. Process Stream
	hasher := sha256.New()
	
	var fc chunker.Chunker
	if opts.UseCDC {
		fc = chunker.NewCDCChunker(opts.ChunkSize)
	} else {
		fc = chunker.NewFixedChunker(opts.ChunkSize)
	}

	var finalEntries []format.ChunkEntry
	var blockTable []uint32

	currentOffset := uint64(0)
	buf := make([]byte, BlockSize)

	var hitCount int

	for {
		// Use io.ReadFull to guarantee complete 16MB block reads.
		// Regular Read() can return partial reads, causing block boundary
		// misalignment that corrupts the delta pool on decompression.
		n, errRead := io.ReadFull(inFile, buf)
		if errRead == io.ErrUnexpectedEOF {
			errRead = nil // Partial read is OK for the last block
		}
		if n > 0 {
			blockData := buf[:n]
			hasher.Write(blockData)
			opts.OnProgress(n)

			chunks := fc.Split(blockData)
			numChunks := len(chunks)

			type processedChunk struct {
				entry format.ChunkEntry
				res   []byte
				err   error
			}
			results := make([]processedChunk, numChunks)
			jobs := make(chan int, numChunks)
			var wg sync.WaitGroup

			for w := 0; w < opts.Concurrency; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						chunk := chunks[i]
						match, err := searcher.FindBestMatch(chunk.Data)
						if err != nil {
							results[i] = processedChunk{err: err}
							continue
						}

						residual := delta.XOR(chunk.Data, match.Pattern)

						results[i] = processedChunk{
							res: residual,
							entry: format.ChunkEntry{
								CodebookID:   match.CodebookID,
								DeltaSize:    uint32(len(residual)),
								OriginalSize: uint32(chunk.Size),
							},
						}
					}
				}()
			}

			for i := 0; i < numChunks; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()

			// Gather the residuals for this Block
			var blockPlainDeltas []byte

			for i := 0; i < numChunks; i++ {
				res := results[i]
				if res.err != nil {
					return nil, res.err
				}

				res.entry.DeltaOffset = currentOffset
				finalEntries = append(finalEntries, res.entry)

				blockPlainDeltas = append(blockPlainDeltas, res.res...)
				currentOffset += uint64(len(res.res))

				if res.entry.DeltaSize > 0 {
					zeroes := 0
					for _, b := range res.res {
						if b == 0 {
							zeroes++
						}
					}
					if zeroes > (int(res.entry.DeltaSize) * 95 / 100) {
						hitCount++
					}
				}
			}

			// Compress this independent block
			compBlock, err := delta.CompressPool(blockPlainDeltas)
			if err != nil {
				return nil, fmt.Errorf("pack: compress block: %w", err)
			}

			// Encrypt if required
			if header.IsEncrypted {
				compBlock, err = crypto.Encrypt(derivedKey, compBlock)
				if err != nil {
					return nil, fmt.Errorf("pack: encrypt block: %w", err)
				}
			}

			blockTable = append(blockTable, uint32(len(compBlock)))

			if _, err := outFile.Write(compBlock); err != nil {
				return nil, err
			}
		}

		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			return nil, errRead
		}
	}

	// 4. Finalize — Update header with REAL chunk count (may differ from estimate)
	actualChunkCount := uint32(len(finalEntries))
	header.ChunkCount = actualChunkCount
	copy(header.OriginalHash[:], hasher.Sum(nil))

	tableData := format.SerializeChunkTable(finalEntries)
	if header.IsEncrypted {
		encTable, err := crypto.Encrypt(derivedKey, tableData)
		if err != nil {
			return nil, fmt.Errorf("pack: encrypt chunk table: %w", err)
		}
		tableData = encTable
	}

	// Build actual block table bytes
	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}

	// Calculate the actual metadata size
	actualHeaderSize := int64(len(header.Serialize()))
	actualBlockTableSize := int64(len(blockTableRaw))
	actualChunkTableSize := int64(len(tableData))
	actualMetadataSize := actualHeaderSize + actualBlockTableSize + actualChunkTableSize

	// If actual metadata size differs from what we reserved, we need to
	// rewrite the entire file with correct offsets.
	if actualMetadataSize != deltaPoolStartOffset {
		// Read back all the delta pool data we already wrote
		deltaPoolSize, _ := outFile.Seek(0, 2) // seek to end
		deltaPoolSize -= deltaPoolStartOffset
		deltaPoolData := make([]byte, deltaPoolSize)
		outFile.Seek(deltaPoolStartOffset, 0)
		io.ReadFull(outFile, deltaPoolData)

		// Truncate and rewrite from the beginning
		outFile.Truncate(actualMetadataSize + int64(len(deltaPoolData)))
		outFile.Seek(0, 0)

		if _, err := outFile.Write(header.Serialize()); err != nil {
			return nil, err
		}
		if _, err := outFile.Write(blockTableRaw); err != nil {
			return nil, err
		}
		if _, err := outFile.Write(tableData); err != nil {
			return nil, err
		}
		if _, err := outFile.Write(deltaPoolData); err != nil {
			return nil, err
		}
	} else {
		// Metadata size matches estimate — just seek back and overwrite in place
		outFile.Seek(0, 0)
		if _, err := outFile.Write(header.Serialize()); err != nil {
			return nil, err
		}
		if _, err := outFile.Write(blockTableRaw); err != nil {
			return nil, err
		}
		if _, err := outFile.Write(tableData); err != nil {
			return nil, err
		}
	}

	packedInfo, _ := outFile.Stat()

	return &Metrics{
		OriginalSize: originalSize,
		PackedSize:   uint64(packedInfo.Size()),
		Duration:     time.Since(start),
		HitRate:      (float64(hitCount) / float64(actualChunkCount)) * 100,
	}, nil
}
