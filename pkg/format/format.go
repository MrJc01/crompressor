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
	// Version3 introduces entropy passthrough.
	Version3 uint16 = 3
	// Version4 introduces adaptive ChunkSize and CodebookHash
	Version4 uint16 = 4
	// Version5 introduces MerkleTree Delta Sync
	Version5 uint16 = 5

	// HashSize is the size of SHA-256 hashes (32 bytes).
	HashSize = 32

	// HeaderSize is the fixed size of a .crom file v1 header (50 bytes).
	HeaderSize = MagicSize + 2 + HashSize + 8 + 4

	// HeaderSizeV2 is the fixed size of a .crom file v2 header (68 bytes).
	// Layout V2:
	//   Magic (4)
	//   Version (2)
	//   IsEncrypted (1)
	//   IsPassthrough (1)
	//   Salt (16)
	//   OriginalHash (32)
	//   OriginalSize (8)
	HeaderSizeV2 = 68

	// HeaderSizeV4 adds ChunkSize (4) and CodebookHash (8). Total 80 bytes.
	HeaderSizeV4 = 80

	// HeaderSizeV5 adds MerkleRoot (32). Total 112 bytes.
	HeaderSizeV5 = 112

	// EntrySize is the fixed size of a ChunkEntry in the Chunk Table (24 bytes).
	EntrySize = 8 + 8 + 4 + 4

	// ChunksPerBlock is the number of chunks grouped into a single Zstd frame in V2.
	// Must match cromlib.BlockSize / chunker.DefaultChunkSize = 16MB / 128B = 131072.
	ChunksPerBlock = 131072

	// LiteralCodebookID is a sentinel value (MAX_UINT64) used to mark chunks
	// where no good codebook match was found. These chunks are stored verbatim
	// in the delta pool without XOR against a pattern.
	LiteralCodebookID = ^uint64(0)
)

// Header contains the top-level metadata of a .crom file.
type Header struct {
	Version       uint16
	IsEncrypted   bool
	IsPassthrough bool
	Salt          [16]byte
	OriginalHash [HashSize]byte
	OriginalSize uint64
	ChunkCount   uint32
	ChunkSize    uint32
	CodebookHash [8]byte
	MerkleRoot   [32]byte
}

// NumBlocks returns the expected number of Zstd blocks for this file (V2+).
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

	if h.Version >= Version2 && h.Version <= Version5 {
		minSize := HeaderSizeV2
		if h.Version == Version4 {
			minSize = HeaderSizeV4
		} else if h.Version == Version5 {
			minSize = HeaderSizeV5
		}
		if len(data) < minSize {
			return nil, fmt.Errorf("format: header too small for v%d (%d < %d)", h.Version, len(data), minSize)
		}
		h.IsEncrypted = data[6] == 1
		if h.Version >= Version3 {
			h.IsPassthrough = data[7] == 1
		}
		copy(h.Salt[:], data[8:24])
		copy(h.OriginalHash[:], data[24:56])
		h.OriginalSize = binary.LittleEndian.Uint64(data[56:64])
		h.ChunkCount = binary.LittleEndian.Uint32(data[64:68])
		
		if h.Version >= Version4 {
			h.ChunkSize = binary.LittleEndian.Uint32(data[68:72])
			copy(h.CodebookHash[:], data[72:80])
		}
		if h.Version >= Version5 {
			copy(h.MerkleRoot[:], data[80:112])
		}
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

	// Default to V5 if not explicitly set and not V1
	if h.Version < Version2 || h.Version > Version5 {
		h.Version = Version5
	}
	
	size := HeaderSizeV2
	if h.Version == Version4 {
		size = HeaderSizeV4
	} else if h.Version == Version5 {
		size = HeaderSizeV5
	}
	
	buf := make([]byte, size)
	copy(buf[0:MagicSize], MagicString)
	binary.LittleEndian.PutUint16(buf[4:6], h.Version)
	if h.IsEncrypted {
		buf[6] = 1
	}
	if h.IsPassthrough {
		buf[7] = 1
	}
	copy(buf[8:24], h.Salt[:])
	copy(buf[24:56], h.OriginalHash[:])
	binary.LittleEndian.PutUint64(buf[56:64], h.OriginalSize)
	binary.LittleEndian.PutUint32(buf[64:68], h.ChunkCount)
	
	if h.Version >= Version4 {
		binary.LittleEndian.PutUint32(buf[68:72], h.ChunkSize)
		copy(buf[72:80], h.CodebookHash[:])
	}
	
	if h.Version >= Version5 {
		copy(buf[80:112], h.MerkleRoot[:])
	}
	
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
