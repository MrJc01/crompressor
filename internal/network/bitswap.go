//go:build !wasm

package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

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

	var derivedKey []byte
	if header.IsEncrypted {
		derivedKey = crypto.DeriveKey([]byte(encryptionKey), header.Salt[:])
	}

	var uncompressedPool []byte
	if header.Version >= format.Version2 {
		for i, blockSize := range blockTable {
			blockData := make([]byte, blockSize)
			if _, err := io.ReadFull(rStream, blockData); err != nil {
				return fmt.Errorf("bitswap: read block %d: %w", i, err)
			}

			if header.IsEncrypted {
				dec, err := crypto.Decrypt(derivedKey, blockData)
				if err != nil {
					return fmt.Errorf("bitswap: decrypt block %d: %w", i, err)
				}
				blockData = dec
			}

			decompressed, err := delta.DecompressPool(blockData)
			if err != nil {
				return fmt.Errorf("bitswap: decompress block %d: %w", i, err)
			}
			uncompressedPool = append(uncompressedPool, decompressed...)
		}
	} else {
		compDeltaPool, _ := io.ReadAll(rStream)
		uncompressedPool, err = delta.DecompressPool(compDeltaPool)
		if err != nil {
			return err
		}
	}

	// Stream chunks
	for _, idx := range indices {
		if idx >= uint32(len(entries)) {
			continue // Invalid index
		}

		entry := entries[idx]
		endOffset := entry.DeltaOffset + uint64(entry.DeltaSize)
		if endOffset > uint64(len(uncompressedPool)) {
			return fmt.Errorf("bitswap: bounds error on chunk %d", idx)
		}

		residual := uncompressedPool[entry.DeltaOffset:endOffset]

		// Format: [Chunk Index (4)] [Residual Size (4)] [Residual Data]
		header := make([]byte, 8)
		binary.LittleEndian.PutUint32(header[0:4], idx)
		binary.LittleEndian.PutUint32(header[4:8], uint32(len(residual)))

		if _, err := s.Write(header); err != nil {
			return err
		}
		if len(residual) > 0 {
			if _, err := s.Write(residual); err != nil {
				return err
			}
		}
	}

	return nil
}

// ReceiveChunks reads streamed deltas and rebuilds the .crom file, taking the chunks
// it already has from the local version. It returns the bytes read from the network.
func ReceiveChunks(tempPath string, outPath string, manifest *cromsync.ChunkManifest, missingIndices []uint32, s network.Stream, codebookPath string, encryptionKey string) (int64, error) {
	residuals := make(map[uint32][]byte)
	var received int64

	// 1. Read the missing chunks from network
	for i := 0; i < len(missingIndices); i++ {
		header := make([]byte, 8)
		if _, err := io.ReadFull(s, header); err != nil {
			if err == io.EOF {
				break
			}
			return received, fmt.Errorf("bitswap: read header: %w", err)
		}

		idx := binary.LittleEndian.Uint32(header[0:4])
		size := binary.LittleEndian.Uint32(header[4:8])

		residual := make([]byte, size)
		if size > 0 {
			if _, err := io.ReadFull(s, residual); err != nil {
				return received, fmt.Errorf("bitswap: read residual data: %w", err)
			}
		}

		residuals[idx] = residual
		received += int64(len(header)) + int64(size)
	}

	fmt.Printf("[Sync] Bitswap completo. %d chunks recebidos. Repackaging...\n", len(residuals))

	// 2. Index the residuals of the local version, which may be at other positions
	type chunkKey struct{ CodebookID, DeltaHash uint64 }
	local := make(map[chunkKey][]byte)
	if f, err := os.Open(outPath); err == nil {
		reader := format.NewReader(f)
		lHeader, lBlockTable, lEnts, rStream, err := reader.ReadStream(encryptionKey)
		if err == nil {
			var derivedKey []byte
			if lHeader.IsEncrypted {
				derivedKey = crypto.DeriveKey([]byte(encryptionKey), lHeader.Salt[:])
			}
			var pool []byte
			for i, blockSize := range lBlockTable {
				blockData := make([]byte, blockSize)
				if _, err := io.ReadFull(rStream, blockData); err != nil {
					f.Close()
					return received, fmt.Errorf("bitswap: read local block %d: %w", i, err)
				}
				if lHeader.IsEncrypted {
					if blockData, err = crypto.Decrypt(derivedKey, blockData); err != nil {
						f.Close()
						return received, fmt.Errorf("bitswap: decrypt local block %d: %w", i, err)
					}
				}
				decompressed, err := delta.DecompressPool(blockData)
				if err != nil {
					f.Close()
					return received, fmt.Errorf("bitswap: decompress local block %d: %w", i, err)
				}
				pool = append(pool, decompressed...)
			}
			for _, le := range lEnts {
				end := le.DeltaOffset + uint64(le.DeltaSize)
				if end <= uint64(len(pool)) {
					res := pool[le.DeltaOffset:end]
					local[chunkKey{le.CodebookID, xxhash.Sum64(res)}] = res
				}
			}
		}
		f.Close()
	}

	// 3. Rebuild the .crom file from the manifest and the received residuals
	outFile, err := os.Create(tempPath)
	if err != nil {
		return received, err
	}
	defer outFile.Close()

	fileHeader := &format.Header{
		Version:      format.Version7,
		OriginalSize: manifest.OriginalSize,
		ChunkCount:   manifest.ChunkCount,
	}
	copy(fileHeader.OriginalHash[:], manifest.OriginalHash[:])

	finalEntries := make([]format.ChunkEntry, manifest.ChunkCount)
	for i, e := range manifest.Entries {
		finalEntries[i] = format.ChunkEntry{
			CodebookID:    e.CodebookID,
			CodebookIndex: e.CodebookIndex,
			OriginalSize:  e.ChunkSize,
		}
	}
	blockStarts := format.BlockStarts(finalEntries)

	metadataSize := int64(len(fileHeader.Serialize())) + int64(len(blockStarts))*4 + int64(manifest.ChunkCount)*int64(format.GetEntrySize(format.Version7))
	if _, err := outFile.Seek(metadataSize, 0); err != nil {
		return received, err
	}

	blockTable := make([]uint32, 0, len(blockStarts))
	currentOffset := uint64(0)

	for b := range blockStarts {
		var blockPlainDeltas []byte
		startIdx, endIdx := format.BlockRange(blockStarts, b, len(finalEntries))

		for idx := startIdx; idx < endIdx; idx++ {
			res, ok := residuals[uint32(idx)]
			if !ok {
				e := manifest.Entries[idx]
				if res, ok = local[chunkKey{e.CodebookID, e.DeltaHash}]; !ok {
					return received, fmt.Errorf("bitswap: missing chunk %d for reconstruction", idx)
				}
			}

			finalEntries[idx].DeltaOffset = currentOffset
			finalEntries[idx].DeltaSize = uint32(len(res))
			blockPlainDeltas = append(blockPlainDeltas, res...)
			currentOffset += uint64(len(res))
		}

		compBlock, err := delta.CompressPool(blockPlainDeltas)
		if err != nil {
			return received, fmt.Errorf("bitswap: repack compress block: %w", err)
		}

		blockTable = append(blockTable, uint32(len(compBlock)))
		if _, err := outFile.Write(compBlock); err != nil {
			return received, err
		}
	}

	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	if _, err := outFile.Seek(0, 0); err != nil {
		return received, err
	}
	for _, part := range [][]byte{fileHeader.Serialize(), blockTableRaw, format.SerializeChunkTable(finalEntries, format.Version7)} {
		if _, err := outFile.Write(part); err != nil {
			return received, err
		}
	}

	return received, nil
}

// Ensure Codebook is opened and loaded since bit-swapping usually requires verification,
// though during direct manifest trust we skip codebook hash check inside packets to save CPU.
func loadCb(path string) (*codebook.Reader, error) {
	return codebook.Open(path)
}
