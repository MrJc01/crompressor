package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

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

// ReceiveChunks reads streamed deltas, buffers them, and builds a robust V2 .crom file.
func ReceiveChunks(destPath string, manifest *cromsync.ChunkManifest, totalChunks int, s network.Stream) error {
	// 1. Read all streamed residuals into memory (for this phase, we assume all are received)
	// Note: In a true CAS system, we would ask multiple peers for different pieces.
	residuals := make(map[uint32][]byte)

	for i := 0; i < totalChunks; i++ {
		header := make([]byte, 8)
		if _, err := io.ReadFull(s, header); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("bitswap: read header: %w", err)
		}

		idx := binary.LittleEndian.Uint32(header[0:4])
		size := binary.LittleEndian.Uint32(header[4:8])

		residual := make([]byte, size)
		if size > 0 {
			if _, err := io.ReadFull(s, residual); err != nil {
				return fmt.Errorf("bitswap: read residual data: %w", err)
			}
		}

		residuals[idx] = residual
	}

	fmt.Printf("[Sync] Bitswap completo. %d/%d chunks recebidos. Repackaging...\n", len(residuals), totalChunks)

	// 2. Rebuild the .crom file from the manifest and the received residuals
	// We use the same format V2 structure: group every 8192 chunks into a block.

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	fileHeader := &format.Header{
		Version:      format.Version2,
		OriginalSize: manifest.OriginalSize,
		ChunkCount:   manifest.ChunkCount,
		IsEncrypted:  false, // We rebuild it unencrypted for simplicity in P2P receiver unless requested
	}
	copy(fileHeader.OriginalHash[:], manifest.OriginalHash[:])

	headerBytes := fileHeader.Serialize()
	if _, err := outFile.Write(headerBytes); err != nil {
		return err
	}

	numBlocks := fileHeader.NumBlocks()
	blockTable := make([]uint32, 0, numBlocks)

	// Dummy write for block table and chunk table
	blockTableSpace := make([]byte, numBlocks*4)
	if _, err := outFile.Write(blockTableSpace); err != nil {
		return err
	}

	chunkTableSpace := make([]byte, manifest.ChunkCount*format.EntrySize)
	if _, err := outFile.Write(chunkTableSpace); err != nil {
		return err
	}

	// Write blocks
	finalEntries := make([]format.ChunkEntry, manifest.ChunkCount)
	currentOffset := uint64(0)

	for b := uint32(0); b < numBlocks; b++ {
		var blockPlainDeltas []byte

		startIdx := b * format.ChunksPerBlock
		endIdx := startIdx + format.ChunksPerBlock
		if endIdx > manifest.ChunkCount {
			endIdx = manifest.ChunkCount
		}

		for idx := startIdx; idx < endIdx; idx++ {
			res, ok := residuals[idx]
			if !ok {
				// We don't have this chunk (sync failed)
				return fmt.Errorf("bitswap: missing chunk %d for reconstruction", idx)
			}

			finalEntries[idx] = format.ChunkEntry{
				CodebookID:   manifest.Entries[idx].CodebookID,
				DeltaOffset:  currentOffset,
				DeltaSize:    uint32(len(res)),
				OriginalSize: manifest.Entries[idx].ChunkSize,
			}

			blockPlainDeltas = append(blockPlainDeltas, res...)
			currentOffset += uint64(len(res))
		}

		// Compress block
		compBlock, err := delta.CompressPool(blockPlainDeltas)
		if err != nil {
			return fmt.Errorf("bitswap: repack compress block: %w", err)
		}

		blockTable = append(blockTable, uint32(len(compBlock)))

		if _, err := outFile.Write(compBlock); err != nil {
			return err
		}
	}

	// Rewrite Metadata
	outFile.Seek(0, 0)

	if _, err := outFile.Write(fileHeader.Serialize()); err != nil {
		return err
	}

	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	if _, err := outFile.Write(blockTableRaw); err != nil {
		return err
	}

	tableData := format.SerializeChunkTable(finalEntries)
	if _, err := outFile.Write(tableData); err != nil {
		return err
	}

	return nil
}

// Ensure Codebook is opened and loaded since bit-swapping usually requires verification,
// though during direct manifest trust we skip codebook hash check inside packets to save CPU.
func loadCb(path string) (*codebook.Reader, error) {
	return codebook.Open(path)
}
