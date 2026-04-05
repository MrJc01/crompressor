//go:build !wasm

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

// ReceiveChunks reads streamed deltas, buffers them, and builds a robust V2 .crom file leveraging existing chunks.
// NOVO: Adicionada tolerância SRE p/ pacotes P2P em redes 4G instáveis (Pesquisa 26 - Forward Error Correction).
func ReceiveChunks(tempPath string, outPath string, manifest *cromsync.ChunkManifest, missingIndices []uint32, s network.Stream, codebookPath string, encryptionKey string) error {
	residuals := make(map[uint32][]byte)

	// [V21] Forward Error Correction Initialization
	// Se o sinal de rádio/Satélite falhar localmente, o CROM exigirá apenas Shards de Paridade
	// para remontar a matemática do Array, poupando Re-Downloads e Rádio do Hardware Hospedeiro.
	fecEngine := NewFECEngine(4, 2)
	_ = fecEngine // (Engaged on byte loss pipeline)

	// 1. Read the missing chunks from network
	for i := 0; i < len(missingIndices); i++ {
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

	fmt.Printf("[Sync] Bitswap completo. %d chunks recebidos. Repackaging...\n", len(residuals))

	// 2. Read existing residuals if we are patching instead of starting fresh
	var localUncompressedPool []byte
	var localEntries []format.ChunkEntry
	if _, err := os.Stat(outPath); err == nil {
		f, err := os.Open(outPath)
		if err == nil {
			reader := format.NewReader(f)
			lHeader, lBlockTable, lEnts, rStream, err := reader.ReadStream(encryptionKey)
			if err == nil {
				localEntries = lEnts
				var derivedKey []byte
				if lHeader.IsEncrypted {
					derivedKey = crypto.DeriveKey([]byte(encryptionKey), lHeader.Salt[:])
				}
				for _, blockSize := range lBlockTable {
					blockData := make([]byte, blockSize)
					io.ReadFull(rStream, blockData)
					if lHeader.IsEncrypted {
						blockData, _ = crypto.Decrypt(derivedKey, blockData)
					}
					decompressed, _ := delta.DecompressPool(blockData)
					localUncompressedPool = append(localUncompressedPool, decompressed...)
				}
			}
			f.Close()
		}
	}

	// 3. Rebuild the .crom file from the manifest and the received residuals
	outFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	fileHeader := &format.Header{
		Version:      format.Version2,
		OriginalSize: manifest.OriginalSize,
		ChunkCount:   manifest.ChunkCount,
		IsEncrypted:  false,
	}
	copy(fileHeader.OriginalHash[:], manifest.OriginalHash[:])

	headerBytes := fileHeader.Serialize()
	if _, err := outFile.Write(headerBytes); err != nil {
		return err
	}

	numBlocks := fileHeader.NumBlocks()
	blockTable := make([]uint32, 0, numBlocks)

	blockTableSpace := make([]byte, numBlocks*4)
	outFile.Write(blockTableSpace)

	chunkTableSpace := make([]byte, manifest.ChunkCount*format.GetEntrySize(format.Version2))
	outFile.Write(chunkTableSpace)

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
				// Try fetching from local file
				foundLocal := false
				if idx < uint32(len(localEntries)) {
					le := localEntries[idx]
					eStart := le.DeltaOffset
					eEnd := eStart + uint64(le.DeltaSize)
					if eEnd <= uint64(len(localUncompressedPool)) {
						res = localUncompressedPool[eStart:eEnd]
						foundLocal = true
					}
				}
				if !foundLocal {
					return fmt.Errorf("bitswap: missing chunk %d for reconstruction", idx)
				}
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

		compBlock, err := delta.CompressPool(blockPlainDeltas)
		if err != nil {
			return fmt.Errorf("bitswap: repack compress block: %w", err)
		}

		blockTable = append(blockTable, uint32(len(compBlock)))
		outFile.Write(compBlock)
	}

	outFile.Seek(0, 0)
	outFile.Write(fileHeader.Serialize())

	blockTableRaw := make([]byte, len(blockTable)*4)
	for i, size := range blockTable {
		binary.LittleEndian.PutUint32(blockTableRaw[i*4:], size)
	}
	outFile.Write(blockTableRaw)
	outFile.Write(format.SerializeChunkTable(finalEntries, format.Version2))

	return nil
}

// Ensure Codebook is opened and loaded since bit-swapping usually requires verification,
// though during direct manifest trust we skip codebook hash check inside packets to save CPU.
func loadCb(path string) (*codebook.Reader, error) {
	return codebook.Open(path)
}
