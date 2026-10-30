//go:build !wasm

package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/cespare/xxhash/v2"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/pkg/format"
	cromsync "github.com/MrJc01/crompressor/pkg/sync"
)

// StreamChunks extracts the uncompressed XOR delta for each requested index
// from the local .crom file and sends it over the libp2p stream.
func StreamChunks(localPath, codebookPath, encryptionKey string, indices []uint32, s network.Stream) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := format.NewReader(f)
	header, blockTable, entries, rStream, err := reader.ReadStream(encryptionKey)
	if err != nil {
		return err
	}

	wireCb, err := loadCb(codebookPath)
	if err != nil {
		return fmt.Errorf("bitswap: open codebook: %w", err)
	}
	defer wireCb.Close()
	dict := wireCb.Dictionary()
	zw, err := delta.NewBrainWriter(s, dict)
	if err != nil {
		return err
	}
	defer zw.Close()

	var poolDict []byte
	if header.UsesBrainDict {
		poolDict = dict
	}
	ordered := append([]uint32(nil), indices...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	next := 0

	err = cromsync.ForEachBlock(rStream, header, blockTable, entries, poolDict, encryptionKey,
		func(first, end int) bool {
			return next < len(ordered) && int(ordered[next]) < end
		},
		func(first, end int, pool []byte) error {
			base := entries[first].DeltaOffset
			for ; next < len(ordered) && int(ordered[next]) < end; next++ {
				idx := ordered[next]
				if int(idx) < first {
					continue // Invalid or repeated index
				}
				entry := entries[idx]
				from := entry.DeltaOffset - base
				to := from + uint64(entry.DeltaSize)
				if to > uint64(len(pool)) {
					return fmt.Errorf("bitswap: bounds error on chunk %d", idx)
				}
				residual := pool[from:to]

				// Format: [Chunk Index (4)] [Residual Size (4)] [Residual Data]
				chunkHeader := make([]byte, 8)
				binary.LittleEndian.PutUint32(chunkHeader[0:4], idx)
				binary.LittleEndian.PutUint32(chunkHeader[4:8], uint32(len(residual)))
				if _, err := zw.Write(chunkHeader); err != nil {
					return err
				}
				if _, err := zw.Write(residual); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return zw.Close()
}

// maxPatchOverhead is the largest micro-patch the compiler writes for a chunk.
const maxPatchOverhead = 256

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// ReceiveChunks reads streamed deltas and rebuilds the .crom file, taking the chunks
// it already has from the local version. It returns the bytes read from the network.
func ReceiveChunks(tempPath string, outPath string, manifest *cromsync.ChunkManifest, missingIndices []uint32, s network.Stream, codebookPath string, encryptionKey string) (int64, error) {
	var received int64

	// Residuals wait on disk, so memory does not grow with the file.
	spill, err := os.CreateTemp("", "crom_sync_residuals_*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(spill.Name())
	defer spill.Close()
	type spilled struct {
		off  int64
		size uint32
	}
	var spillEnd int64
	keep := func(data []byte) (spilled, error) {
		ref := spilled{spillEnd, uint32(len(data))}
		_, err := spill.Write(data)
		spillEnd += int64(len(data))
		return ref, err
	}
	residuals := make(map[uint32]spilled)

	cb, err := loadCb(codebookPath)
	if err != nil {
		return 0, fmt.Errorf("bitswap: open codebook: %w", err)
	}
	defer cb.Close()
	brainDict := cb.Dictionary()

	// 1. Read the missing chunks from network
	counted := &countingReader{r: s}
	zr, err := delta.NewBrainReader(counted, brainDict)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	for i := 0; i < len(missingIndices); i++ {
		header := make([]byte, 8)
		if _, err := io.ReadFull(zr, header); err != nil {
			if err == io.EOF {
				break
			}
			return received, fmt.Errorf("bitswap: read header: %w", err)
		}

		idx := binary.LittleEndian.Uint32(header[0:4])
		size := binary.LittleEndian.Uint32(header[4:8])

		// The peer is not trusted to size allocations: a residual never exceeds its chunk.
		if idx >= manifest.ChunkCount || size > manifest.Entries[idx].ChunkSize+maxPatchOverhead {
			return received, fmt.Errorf("bitswap: invalid chunk %d with %d bytes", idx, size)
		}

		residual := make([]byte, size)
		if size > 0 {
			if _, err := io.ReadFull(zr, residual); err != nil {
				return received, fmt.Errorf("bitswap: read residual data: %w", err)
			}
		}

		ref, err := keep(residual)
		if err != nil {
			return received, err
		}
		residuals[idx] = ref
	}
	received = counted.n

	fmt.Printf("[Sync] Bitswap completo. %d chunks recebidos. Repackaging...\n", len(residuals))

	// 2. Keep from the local version the residuals the new one still uses, wherever they are
	type chunkKey struct{ CodebookID, DeltaHash uint64 }
	needed := make(map[chunkKey]bool, len(manifest.Entries))
	for i, e := range manifest.Entries {
		if _, ok := residuals[uint32(i)]; !ok {
			needed[chunkKey{e.CodebookID, e.DeltaHash}] = true
		}
	}
	local := make(map[chunkKey]spilled)
	if f, err := os.Open(outPath); err == nil {
		defer f.Close()
		reader := format.NewReader(f)
		lHeader, lBlockTable, lEnts, rStream, err := reader.ReadStream(encryptionKey)
		if err == nil && lHeader.ChunkCount > 0 {
			var localDict []byte
			if lHeader.UsesBrainDict {
				localDict = brainDict
			}
			err = cromsync.ForEachBlock(rStream, lHeader, lBlockTable, lEnts, localDict, encryptionKey, nil, func(first, end int, pool []byte) error {
				base := lEnts[first].DeltaOffset
				for _, le := range lEnts[first:end] {
					from := le.DeltaOffset - base
					to := from + uint64(le.DeltaSize)
					if to > uint64(len(pool)) {
						continue
					}
					res := pool[from:to]
					key := chunkKey{le.CodebookID, xxhash.Sum64(res)}
					if _, have := local[key]; needed[key] && !have {
						ref, err := keep(res)
						if err != nil {
							return err
						}
						local[key] = ref
					}
				}
				return nil
			})
			if err != nil {
				return received, fmt.Errorf("bitswap: read local version: %w", err)
			}
		}
	}

	// 3. Rebuild the .crom file from the manifest and the received residuals
	outFile, err := os.Create(tempPath)
	if err != nil {
		return received, err
	}
	defer outFile.Close()

	fileHeader := &format.Header{
		Version:       format.Version7,
		OriginalSize:  manifest.OriginalSize,
		ChunkCount:    manifest.ChunkCount,
		UsesBrainDict: true,
	}
	copy(fileHeader.OriginalHash[:], manifest.OriginalHash[:])

	// With a key on this node, the rebuilt file stays encrypted at rest.
	var newKey []byte
	if encryptionKey != "" {
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return received, fmt.Errorf("bitswap: generate salt: %w", err)
		}
		copy(fileHeader.Salt[:], salt)
		fileHeader.IsEncrypted = true
		newKey = crypto.DeriveKey([]byte(encryptionKey), salt)
	}

	finalEntries := make([]format.ChunkEntry, manifest.ChunkCount)
	for i, e := range manifest.Entries {
		finalEntries[i] = format.ChunkEntry{
			CodebookID:    e.CodebookID,
			CodebookIndex: e.CodebookIndex,
			OriginalSize:  e.ChunkSize,
		}
	}
	blockStarts := format.BlockStarts(finalEntries)

	// The compressed table size is only known after the blocks, so they go to a temp file.
	tmpPool, err := os.CreateTemp("", "crom_sync_pool_*")
	if err != nil {
		return received, err
	}
	defer os.Remove(tmpPool.Name())
	defer tmpPool.Close()

	blockTable := make([]uint32, 0, len(blockStarts))
	currentOffset := uint64(0)

	for b := range blockStarts {
		var blockPlainDeltas []byte
		startIdx, endIdx := format.BlockRange(blockStarts, b, len(finalEntries))

		for idx := startIdx; idx < endIdx; idx++ {
			ref, ok := residuals[uint32(idx)]
			if !ok {
				e := manifest.Entries[idx]
				if ref, ok = local[chunkKey{e.CodebookID, e.DeltaHash}]; !ok {
					return received, fmt.Errorf("bitswap: missing chunk %d for reconstruction", idx)
				}
			}
			res := make([]byte, ref.size)
			if ref.size > 0 {
				if _, err := spill.ReadAt(res, ref.off); err != nil {
					return received, fmt.Errorf("bitswap: read residual: %w", err)
				}
			}

			finalEntries[idx].DeltaOffset = currentOffset
			finalEntries[idx].DeltaSize = uint32(len(res))
			blockPlainDeltas = append(blockPlainDeltas, res...)
			currentOffset += uint64(len(res))
		}

		compBlock, err := delta.CompressPoolDict(blockPlainDeltas, brainDict)
		if err != nil {
			return received, fmt.Errorf("bitswap: repack compress block: %w", err)
		}
		if fileHeader.IsEncrypted {
			if compBlock, err = crypto.Encrypt(newKey, compBlock); err != nil {
				return received, fmt.Errorf("bitswap: encrypt block: %w", err)
			}
		}

		blockTable = append(blockTable, uint32(len(compBlock)))
		if _, err := tmpPool.Write(compBlock); err != nil {
			return received, err
		}
	}

	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	fileHeader.CompressedTable = true
	tableData, err := format.EncodeCompressedTable(finalEntries, format.Version7, newKey)
	if err != nil {
		return received, err
	}
	for _, part := range [][]byte{fileHeader.Serialize(), blockTableRaw, tableData} {
		if _, err := outFile.Write(part); err != nil {
			return received, err
		}
	}
	if _, err := tmpPool.Seek(0, 0); err != nil {
		return received, err
	}
	if _, err := io.Copy(outFile, tmpPool); err != nil {
		return received, err
	}

	return received, nil
}

// Ensure Codebook is opened and loaded since bit-swapping usually requires verification,
// though during direct manifest trust we skip codebook hash check inside packets to save CPU.
func loadCb(path string) (*codebook.Reader, error) {
	return codebook.Open(path)
}
