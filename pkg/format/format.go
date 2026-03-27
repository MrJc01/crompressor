// Package format provides the binary serialization logic for .crom files.
package format

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// MagicString identifies a CROM compressed file.
	MagicString = "CROM"
	// MagicSize is the length of the magic string (4 bytes).
	MagicSize = 4

	// Version1 is the first version of the CROM format.
	Version1 uint16 = 1
	// Version2 introduces block-based Deltas and AES-GCM encryption.
	Version2 uint16 = 2

	// HashSize is the size of SHA-256 hashes (32 bytes).
	HashSize = 32

	// HeaderSize is the fixed size of a .crom file v1 header (50 bytes).
	HeaderSize = MagicSize + 2 + HashSize + 8 + 4

	// HeaderSizeV2 is the fixed size of a .crom file v2 header (68 bytes).
	// Layout V2:
	//   Magic (4)
	//   Version (2)
	//   IsEncrypted (1)
	//   Reserved (1)
	//   Salt (16)
	//   OriginalHash (32)
	//   OriginalSize (8)
	//   ChunkCount (4)
	HeaderSizeV2 = 68

	// EntrySize is the fixed size of a ChunkEntry in the Chunk Table (24 bytes).
	EntrySize = 8 + 8 + 4 + 4

	// ChunksPerBlock is the number of chunks grouped into a single Zstd frame in V2 (8192 chunks = 1MB).
	ChunksPerBlock = 8192
)

// Header contains the top-level metadata of a .crom file.
type Header struct {
	Version      uint16
	IsEncrypted  bool
	Salt         [16]byte
	OriginalHash [HashSize]byte
	OriginalSize uint64
	ChunkCount   uint32
}

// NumBlocks returns the expected number of Zstd blocks for this file (V2 only).
func (h *Header) NumBlocks() uint32 {
	return (h.ChunkCount + ChunksPerBlock - 1) / ChunksPerBlock
}

// ChunkEntry represents a single chunk mapping in the Chunk Table.
// It maps a Codebook codeword to a corresponding XOR residual in the Delta Pool.
type ChunkEntry struct {
	CodebookID   uint64 // The ID of the closest pattern in the Codebook.
	DeltaOffset  uint64 // Offset within the DECOMPRESSED delta block.
	DeltaSize    uint32 // Size of the delta in the decompressed pool.
	OriginalSize uint32 // Original uncompressed size of this chunk.
}

// ParseHeader parses either a V1 or V2 header based on the bytes provided.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("format: header too small (%d < %d)", len(data), HeaderSize)
	}

	magic := string(data[0:MagicSize])
	if magic != MagicString {
		return nil, fmt.Errorf("format: invalid magic: %q", magic)
	}

	h := &Header{}
	h.Version = binary.LittleEndian.Uint16(data[4:6])

	if h.Version == Version1 {
		copy(h.OriginalHash[:], data[6:38])
		h.OriginalSize = binary.LittleEndian.Uint64(data[38:46])
		h.ChunkCount = binary.LittleEndian.Uint32(data[46:50])
		return h, nil
	}

	if h.Version == Version2 {
		if len(data) < HeaderSizeV2 {
			return nil, fmt.Errorf("format: v2 header too small (%d < %d)", len(data), HeaderSizeV2)
		}
		h.IsEncrypted = data[6] == 1
		copy(h.Salt[:], data[8:24])
		copy(h.OriginalHash[:], data[24:56])
		h.OriginalSize = binary.LittleEndian.Uint64(data[56:64])
		h.ChunkCount = binary.LittleEndian.Uint32(data[64:68])
		return h, nil
	}

	return nil, fmt.Errorf("format: unsupported version %d", h.Version)
}

// Serialize encodes the header. Generates V2 format bytes by default unless h.Version == 1.
func (h *Header) Serialize() []byte {
	if h.Version == Version1 {
		buf := make([]byte, HeaderSize)
		copy(buf[0:MagicSize], MagicString)
		binary.LittleEndian.PutUint16(buf[4:6], h.Version)
		copy(buf[6:38], h.OriginalHash[:])
		binary.LittleEndian.PutUint64(buf[38:46], h.OriginalSize)
		binary.LittleEndian.PutUint32(buf[46:50], h.ChunkCount)
		return buf
	}

	// Default to V2
	h.Version = Version2
	buf := make([]byte, HeaderSizeV2)
	copy(buf[0:MagicSize], MagicString)
	binary.LittleEndian.PutUint16(buf[4:6], h.Version)
	if h.IsEncrypted {
		buf[6] = 1
	}
	copy(buf[8:24], h.Salt[:])
	copy(buf[24:56], h.OriginalHash[:])
	binary.LittleEndian.PutUint64(buf[56:64], h.OriginalSize)
	binary.LittleEndian.PutUint32(buf[64:68], h.ChunkCount)
	return buf
}

// ParseChunkTable decodes the contiguous slice of ChunkEntries.
func ParseChunkTable(data []byte, count uint32) ([]ChunkEntry, error) {
	expectedLen := int(count) * EntrySize
	if len(data) < expectedLen {
		return nil, errors.New("format: chunk table data too short")
	}

	entries := make([]ChunkEntry, count)
	for i := uint32(0); i < count; i++ {
		offset := i * EntrySize
		entries[i] = ChunkEntry{
			CodebookID:   binary.LittleEndian.Uint64(data[offset : offset+8]),
			DeltaOffset:  binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
			DeltaSize:    binary.LittleEndian.Uint32(data[offset+16 : offset+20]),
			OriginalSize: binary.LittleEndian.Uint32(data[offset+20 : offset+24]),
		}
	}
	return entries, nil
}

// Serialize encodes a slice of ChunkEntries into a contiguous byte slice.
func SerializeChunkTable(entries []ChunkEntry) []byte {
	buf := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		offset := i * EntrySize
		binary.LittleEndian.PutUint64(buf[offset:offset+8], e.CodebookID)
		binary.LittleEndian.PutUint64(buf[offset+8:offset+16], e.DeltaOffset)
		binary.LittleEndian.PutUint32(buf[offset+16:offset+20], e.DeltaSize)
		binary.LittleEndian.PutUint32(buf[offset+20:offset+24], e.OriginalSize)
	}
	return buf
}
