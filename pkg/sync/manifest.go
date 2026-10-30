// Package sync provides the ChunkManifest protocol for CROM P2P synchronization.
//
// A ChunkManifest is a lightweight representation of a .crom file's contents:
// it lists every chunk's CodebookID and a hash of the delta residual, enabling
// two nodes to diff their local state and transfer only missing chunks.
//
// This implements the foundation for Content-Addressable Storage (CAS).
package sync

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/pkg/format"
	"github.com/cespare/xxhash/v2"
)

// ManifestVersion is the current version of the manifest protocol.
const ManifestVersion uint16 = 1

// ManifestEntryBinarySize is the fixed byte size of a serialized ManifestEntry.
// Layout: CodebookID(8) + DeltaHash(8) + ChunkSize(4) + CodebookIndex(1) = 21 bytes
const ManifestEntryBinarySize = 21

// ChunkManifest represents the complete content-addressable map of a .crom file.
type ChunkManifest struct {
	Version      uint16          `json:"version"`
	OriginalHash [32]byte        `json:"original_hash"`
	OriginalSize uint64          `json:"original_size"`
	ChunkCount   uint32          `json:"chunk_count"`
	Entries      []ManifestEntry `json:"entries"`
}

// ManifestEntry represents a single chunk's identity in the manifest.
type ManifestEntry struct {
	CodebookID    uint64 `json:"codebook_id"`    // The Codebook pattern used
	DeltaHash     uint64 `json:"delta_hash"`     // xxhash of the XOR residual
	ChunkSize     uint32 `json:"chunk_size"`     // Original uncompressed size of this chunk
	CodebookIndex uint8  `json:"codebook_index"` // Which brain or engine rebuilds this chunk
}

// DiffResult holds the result of comparing two manifests.
type DiffResult struct {
	Missing []ManifestEntry `json:"missing"` // Entries in remote but not in local
	Extra   []ManifestEntry `json:"extra"`   // Entries in local but not in remote
}

// ForEachBlock reads the delta pool from r and calls fn with each decompressed block
// and its [first, end) chunk range. Blocks for which want returns false are skipped.
func ForEachBlock(r io.Reader, header *format.Header, blockTable []uint32, entries []format.ChunkEntry, dict []byte, encryptionKey string, want func(first, end int) bool, fn func(first, end int, pool []byte) error) error {
	if header.Version < format.Version2 {
		return fmt.Errorf("sync: version %d is not supported", header.Version)
	}
	starts := format.BlockStarts(entries)
	if len(starts) != len(blockTable) {
		return fmt.Errorf("sync: chunk table describes %d blocks, block table has %d", len(starts), len(blockTable))
	}
	var derivedKey []byte
	if header.IsEncrypted {
		derivedKey = crypto.DeriveKey([]byte(encryptionKey), header.Salt[:])
	}
	for i, blockSize := range blockTable {
		first, end := format.BlockRange(starts, i, len(entries))
		if want != nil && !want(first, end) {
			if _, err := io.CopyN(io.Discard, r, int64(blockSize)); err != nil {
				return fmt.Errorf("sync: skip block %d: %w", i, err)
			}
			continue
		}
		blockData := make([]byte, blockSize)
		if _, err := io.ReadFull(r, blockData); err != nil {
			return fmt.Errorf("sync: read block %d: %w", i, err)
		}
		if header.IsEncrypted {
			dec, err := crypto.Decrypt(derivedKey, blockData)
			if err != nil {
				return fmt.Errorf("sync: decrypt block %d: %w", i, err)
			}
			blockData = dec
		}
		pool, err := delta.DecompressPoolDict(blockData, dict)
		if err != nil {
			return fmt.Errorf("sync: decompress block %d: %w", i, err)
		}
		if err := fn(first, end, pool); err != nil {
			return err
		}
	}
	return nil
}

// GenerateManifest reads a .crom file and produces a ChunkManifest.
// It reads all blocks, decompresses the delta pool, and hashes each chunk's residual.
func GenerateManifest(cromFile string, codebookPath string, encryptionKey string) (*ChunkManifest, error) {
	f, err := os.Open(cromFile)
	if err != nil {
		return nil, fmt.Errorf("manifest: open .crom: %w", err)
	}
	defer f.Close()

	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return nil, fmt.Errorf("manifest: open codebook: %w", err)
	}
	defer cb.Close()

	reader := format.NewReader(f)
	header, blockTable, entries, rStream, err := reader.ReadStream(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("manifest: parse error in %s: %w", cromFile, err)
	}

	// Hash each chunk's residual one block at a time, so memory does not grow with the file
	manifestEntries := make([]ManifestEntry, len(entries))
	var dict []byte
	if header.UsesBrainDict {
		dict = cb.Dictionary()
	}
	err = ForEachBlock(rStream, header, blockTable, entries, dict, encryptionKey, nil, func(first, end int, pool []byte) error {
		base := entries[first].DeltaOffset
		for i := first; i < end; i++ {
			entry := entries[i]
			from := entry.DeltaOffset - base
			to := from + uint64(entry.DeltaSize)
			if to > uint64(len(pool)) {
				return fmt.Errorf("manifest: delta bounds error for chunk %d", i)
			}
			manifestEntries[i] = ManifestEntry{
				CodebookID:    entry.CodebookID,
				DeltaHash:     xxhash.Sum64(pool[from:to]),
				ChunkSize:     entry.OriginalSize,
				CodebookIndex: entry.CodebookIndex,
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	manifest := &ChunkManifest{
		Version:      ManifestVersion,
		OriginalHash: header.OriginalHash,
		OriginalSize: header.OriginalSize,
		ChunkCount:   header.ChunkCount,
		Entries:      manifestEntries,
	}

	return manifest, nil
}

// --- Serialization ---

// ToJSON serializes the manifest to indented JSON.
func (m *ChunkManifest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// FromJSON deserializes a ChunkManifest from JSON.
func FromJSON(data []byte) (*ChunkManifest, error) {
	var m ChunkManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: unmarshal JSON: %w", err)
	}
	return &m, nil
}

// ToBinary serializes the manifest to a compact binary format.
//
// Layout:
//
//	Version       (2 bytes)
//	OriginalHash  (32 bytes)
//	OriginalSize  (8 bytes)
//	ChunkCount    (4 bytes)
//	Entries       (ChunkCount * 21 bytes each)
//
// Total header: 46 bytes + (ChunkCount * 21) bytes
func (m *ChunkManifest) ToBinary() []byte {
	headerSize := 2 + 32 + 8 + 4
	buf := make([]byte, headerSize+len(m.Entries)*ManifestEntryBinarySize)

	binary.LittleEndian.PutUint16(buf[0:2], m.Version)
	copy(buf[2:34], m.OriginalHash[:])
	binary.LittleEndian.PutUint64(buf[34:42], m.OriginalSize)
	binary.LittleEndian.PutUint32(buf[42:46], m.ChunkCount)

	// Columns, with IDs as differences, so that compression makes the manifest tiny.
	n := len(m.Entries)
	ids, hashes := buf[headerSize:], buf[headerSize+n*8:]
	sizes, indexes := buf[headerSize+n*16:], buf[headerSize+n*20:]
	var prevID uint64
	for i, e := range m.Entries {
		binary.LittleEndian.PutUint64(ids[i*8:], e.CodebookID-prevID)
		prevID = e.CodebookID
		binary.LittleEndian.PutUint64(hashes[i*8:], e.DeltaHash)
		binary.LittleEndian.PutUint32(sizes[i*4:], e.ChunkSize)
		indexes[i] = e.CodebookIndex
	}

	return buf
}

// FromBinary deserializes a ChunkManifest from binary format.
func FromBinary(data []byte) (*ChunkManifest, error) {
	headerSize := 2 + 32 + 8 + 4
	if len(data) < headerSize {
		return nil, fmt.Errorf("manifest: binary data too short (%d < %d)", len(data), headerSize)
	}

	m := &ChunkManifest{}
	m.Version = binary.LittleEndian.Uint16(data[0:2])
	copy(m.OriginalHash[:], data[2:34])
	m.OriginalSize = binary.LittleEndian.Uint64(data[34:42])
	m.ChunkCount = binary.LittleEndian.Uint32(data[42:46])

	expectedLen := headerSize + int(m.ChunkCount)*ManifestEntryBinarySize
	if len(data) < expectedLen {
		return nil, fmt.Errorf("manifest: binary data truncated (%d < %d)", len(data), expectedLen)
	}

	m.Entries = make([]ManifestEntry, m.ChunkCount)
	n := int(m.ChunkCount)
	ids, hashes := data[headerSize:], data[headerSize+n*8:]
	sizes, indexes := data[headerSize+n*16:], data[headerSize+n*20:]
	var prevID uint64
	for i := 0; i < n; i++ {
		prevID += binary.LittleEndian.Uint64(ids[i*8:])
		m.Entries[i] = ManifestEntry{
			CodebookID:    prevID,
			DeltaHash:     binary.LittleEndian.Uint64(hashes[i*8:]),
			ChunkSize:     binary.LittleEndian.Uint32(sizes[i*4:]),
			CodebookIndex: indexes[i],
		}
	}

	return m, nil
}

// --- Diff ---

// Diff compares a local manifest against a remote manifest and returns
// which entries are missing locally and which are extra (not in remote).
//
// This allows a P2P node to request only the chunks it needs.
func Diff(local, remote *ChunkManifest) *DiffResult {
	type chunkKey struct {
		CodebookID uint64
		DeltaHash  uint64
	}

	localSet := make(map[chunkKey]struct{}, len(local.Entries))
	for _, e := range local.Entries {
		localSet[chunkKey{e.CodebookID, e.DeltaHash}] = struct{}{}
	}

	remoteSet := make(map[chunkKey]struct{}, len(remote.Entries))
	for _, e := range remote.Entries {
		remoteSet[chunkKey{e.CodebookID, e.DeltaHash}] = struct{}{}
	}

	result := &DiffResult{}

	// Missing: in remote but not in local
	for _, e := range remote.Entries {
		if _, ok := localSet[chunkKey{e.CodebookID, e.DeltaHash}]; !ok {
			result.Missing = append(result.Missing, e)
		}
	}

	// Extra: in local but not in remote
	for _, e := range local.Entries {
		if _, ok := remoteSet[chunkKey{e.CodebookID, e.DeltaHash}]; !ok {
			result.Extra = append(result.Extra, e)
		}
	}

	return result
}
