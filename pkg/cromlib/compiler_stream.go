// Package cromlib — Stream-based compiler for pipe/stdin compression.
//
// PackStream compresses data from an io.Reader (no Seek) by buffering blocks
// to temporary files and composing the final .crom output after EOF.
package cromlib

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/entropy"
	"github.com/MrJc01/crompressor/internal/merkle"
	"github.com/MrJc01/crompressor/internal/search"
	"github.com/MrJc01/crompressor/pkg/format"
)

// PackStream compresses data from an io.Reader (e.g. os.Stdin pipe) and writes
// a valid .crom file to the io.Writer output. Since we cannot Seek on a pipe,
// we buffer compressed blocks to /tmp and assemble the final file after EOF.
//
// This enables:
//
//	tail -f /var/log/syslog | crompressor pack --stream -c dict.cromdb -o out.crom
func PackStream(in io.Reader, outPath string, codebookPath string, opts PackOptions) (*Metrics, error) {
	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return nil, fmt.Errorf("stream: open codebook: %w", err)
	}
	defer cb.Close()

	searcher := search.NewLSHSearcher(cb)

	var fc chunker.Chunker
	chunkSize := opts.ChunkSize
	if chunkSize == 0 {
		chunkSize = 128
	}
	if opts.UseACAC {
		fc = chunker.NewSemanticChunker(opts.ACACDelimiter, chunkSize*8)
	} else if opts.UseCDC {
		fc = chunker.NewFastCDCChunker(chunkSize)
	} else {
		fc = chunker.NewFixedChunker(chunkSize)
	}

	hasher := sha256.New()

	var allEntries []format.ChunkEntry
	var blockTable []uint32
	var blockHashes [][]byte

	// Temporary file for the compressed delta pool
	tmpPool, err := os.CreateTemp("", "crom_stream_pool_*")
	if err != nil {
		return nil, fmt.Errorf("stream: create temp pool: %w", err)
	}
	defer os.Remove(tmpPool.Name())
	defer tmpPool.Close()

	var currentOffset uint64
	var hitCount, literalCount, totalChunks int
	var totalSimilarity float64
	var totalBytesRead uint64

	buf := make([]byte, BlockSize)

	for {
		n, errRead := io.ReadFull(in, buf)
		if errRead == io.ErrUnexpectedEOF {
			errRead = nil
		}
		if n > 0 {
			blockData := buf[:n]
			hasher.Write(blockData)
			totalBytesRead += uint64(n)

			if opts.OnProgress != nil {
				opts.OnProgress(n)
			}

			chunks := fc.Split(blockData)
			numChunks := len(chunks)

			type processedChunk struct {
				entry format.ChunkEntry
				res   []byte
				sim   float64
				err   error
			}
			results := make([]processedChunk, numChunks)
			jobs := make(chan int, numChunks)
			var wg sync.WaitGroup

			concurrency := opts.Concurrency
			if concurrency == 0 {
				concurrency = 4
			}

			for w := 0; w < concurrency; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						c := chunks[i]
						
						// EXPERT ROUTING: Entropy-based Passthrough
						chunkEntropy := entropy.Shannon(c.Data)
						if chunkEntropy > 7.8 {
							results[i] = processedChunk{
								entry: format.ChunkEntry{
									CodebookID:   format.LiteralCodebookID,
									DeltaSize:    uint32(len(c.Data)),
									OriginalSize: uint32(len(c.Data)),
								},
								res: c.Data,
								sim: 0.0,
							}
							continue
						}

						match, err := searcher.FindBestMatch(c.Data)
						if err != nil {
							results[i] = processedChunk{err: err}
							continue
						}

						residual := delta.XOR(c.Data, match.Pattern)
						sim := 1.0 - float64(match.Distance)/float64(len(c.Data)*8)

						results[i] = processedChunk{
							entry: format.ChunkEntry{
								CodebookID:   match.CodebookID,
								DeltaSize:    uint32(len(residual)),
								OriginalSize: uint32(len(c.Data)),
							},
							res: residual,
							sim: sim,
						}
					}
				}()
			}

			for i := range chunks {
				jobs <- i
			}
			close(jobs)
			wg.Wait()

			// Assemble block
			var blockDeltas []byte
			var blockEntries []format.ChunkEntry

			for _, r := range results {
				if r.err != nil {
					return nil, r.err
				}
				r.entry.DeltaOffset = currentOffset
				currentOffset += uint64(len(r.res))
				blockDeltas = append(blockDeltas, r.res...)
				blockEntries = append(blockEntries, r.entry)

				totalChunks++
				totalSimilarity += r.sim
				if r.sim >= 0.5 {
					hitCount++
				} else {
					literalCount++
				}
			}

			allEntries = append(allEntries, blockEntries...)

			// Compress block and write to temp
			compBlock, err := delta.CompressPool(blockDeltas)
			if err != nil {
				return nil, fmt.Errorf("stream: compress block: %w", err)
			}

			blockTable = append(blockTable, uint32(len(compBlock)))

			blockHash := sha256.Sum256(compBlock)
			blockHashes = append(blockHashes, blockHash[:])

			if _, err := tmpPool.Write(compBlock); err != nil {
				return nil, fmt.Errorf("stream: write temp pool: %w", err)
			}
		}

		if errRead == io.EOF || errRead != nil {
			break
		}
	}

	// --- Assemble final .crom file ---
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	originalHash := hasher.Sum(nil)
	var hashArr [32]byte
	copy(hashArr[:], originalHash)

	merkleTree := merkle.BuildFromHashes(blockHashes)
	merkleRoot := merkleTree.Root()

	header := &format.Header{
		Version:      format.Version5,
		OriginalSize: totalBytesRead,
		ChunkCount:   uint32(len(allEntries)),
		OriginalHash: hashArr,
		MerkleRoot:   merkleRoot,
	}

	headerBytes := header.Serialize()
	outFile.Write(headerBytes)

	// Write Block Table
	numBlocks := uint32(len(blockTable))
	blockTableRaw := make([]byte, numBlocks*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	outFile.Write(blockTableRaw)

	// Write Chunk Table
	chunkTableData := format.SerializeChunkTable(allEntries, format.Version5)
	outFile.Write(chunkTableData)

	// Copy the compressed delta pool from temp
	tmpPool.Seek(0, 0)
	if _, err := io.Copy(outFile, tmpPool); err != nil {
		return nil, fmt.Errorf("stream: copy pool: %w", err)
	}

	stat, _ := outFile.Stat()
	packedSize := uint64(0)
	if stat != nil {
		packedSize = uint64(stat.Size())
	}

	var hitRate, avgSim float64
	if totalChunks > 0 {
		hitRate = float64(hitCount) / float64(totalChunks) * 100
		avgSim = totalSimilarity / float64(totalChunks)
	}

	return &Metrics{
		OriginalSize:  totalBytesRead,
		PackedSize:    packedSize,
		HitRate:       hitRate,
		LiteralChunks: literalCount,
		TotalChunks:   totalChunks,
		AvgSimilarity: avgSim,
	}, nil
}
