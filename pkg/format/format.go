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
	// Version6 introduces Convergent Encryption and Hierarchical Codebooks (L1, L2, L3)
	Version6 uint16 = 6
	// Version7 introduces Multi-Brain Routing via CodebookIndex
	Version7 uint16 = 7
	// Version8 introduces Metamorphic In-Band Brains (MicroDicts embedded in the header)
	Version8 uint16 = 8
	// Version9 introduces Living CROMFS and O(1) Append Mutations (LSM WAL)
	Version9 uint16 = 9

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

	// HeaderSizeV6 adds IsConvergentEncrypted (1) and CodebookHashes (24). Total 112+1+24 = 137 bytes.
	HeaderSizeV6 = 137

	// HeaderSizeV7 is identical to V6 for now, as routing happens in Chunk Table
	HeaderSizeV7 = 137

	// HeaderSizeV8 adds MicroDictSize (4). Total 141 bytes. Plus dynamic payload.
	HeaderSizeV8 = 141

	// EntrySizeV6 is the fixed size of a ChunkEntry in the Chunk Table (24 bytes).
	EntrySizeV6 uint32 = 24

	// EntrySizeV7 introduces a 1 byte CodebookIndex for Multi-Brain Routing (25 bytes).
	EntrySizeV7 uint32 = 25

	// LiteralCodebookIndex is a sentinel value for literal uncompressed chunks
	LiteralCodebookIndex uint8 = 255

	// MetamorphicCodebookIndex marks chunks routed via the in-band Micro-Brain (V8)
	MetamorphicCodebookIndex uint8 = 254

	// FractalCodebookIndex marks chunks routed via the V26 Fractal Generative Engine
	FractalCodebookIndex uint8 = 253

	// MaxMicroDictSize is the hard security cap for in-band micro-dictionaries.
	// Prevents OOM attacks from forged V8 headers. (32 MiB)
	MaxMicroDictSize uint32 = 32 * 1024 * 1024

	// ChunksPerBlock is the number of chunks grouped into a single Zstd frame in V2.
	// Must match cromlib.BlockSize / chunker.DefaultChunkSize = 16MB / 128B = 131072.
	ChunksPerBlock = 131072

	// FlagIsPatch is applied to CodebookID (bit 60) to indicate the residual is a Micro-Patch (MyersDiff) instead of XOR.
	FlagIsPatch uint64 = 1 << 60

	// LiteralCodebookID is a sentinel value (MAX_UINT64) used to mark chunks
	// where no good codebook match was found. These chunks are stored verbatim
	// in the delta pool without XOR against a pattern.
	LiteralCodebookID = ^uint64(0)
)

// Header contains the top-level metadata of a .crom file.
type Header struct {
	Version               uint16
	IsEncrypted           bool
	IsPassthrough         bool
	IsConvergentEncrypted bool
	Salt                  [16]byte
	OriginalHash          [HashSize]byte
	OriginalSize          uint64
	ChunkCount            uint32
	ChunkSize             uint32
	CodebookHash          [8]byte    // Used for V4 and V5
	CodebookHashes        [3][8]byte // Used for V6 (L1, L2, L3 brains)
	MerkleRoot            [32]byte
	MicroDictSize         uint32     // Used for V8 (Metamorphic In-band Brain)
	MicroDictionaryData   []byte     // Used for V8 (Dynamic array holding BPE patterns)
	Mutations             []V9MutationHeader // Used for V9 to hold metadata of appended deltas
}

// V9MutationHeader represents a WAL entry at the end of a Version9 .crom file.
type V9MutationHeader struct {
	Magic         [4]byte // "CMUT" (Crom MUTation)
	Timestamp     int64   // Unix timestamp of the mutation
	DiffPatchSize uint32  // Size of the appended diff payload
}

// V9MutationHeaderSize is the size of the mutation header (4 + 8 + 4 = 16 bytes).
const V9MutationHeaderSize = 16

// Bytes returns the binary representation of the V9MutationHeader.
func (v *V9MutationHeader) Bytes() []byte {
	buf := make([]byte, V9MutationHeaderSize)
	copy(buf[0:4], v.Magic[:])
	binary.LittleEndian.PutUint64(buf[4:12], uint64(v.Timestamp))
	binary.LittleEndian.PutUint32(buf[12:16], v.DiffPatchSize)
	return buf
}

// ParseV9MutationHeader parses bytes into a V9MutationHeader.
func ParseV9MutationHeader(data []byte) (*V9MutationHeader, error) {
	if len(data) < V9MutationHeaderSize {
		return nil, fmt.Errorf("format: mutation header too small (%d < %d)", len(data), V9MutationHeaderSize)
	}
	h := &V9MutationHeader{}
	copy(h.Magic[:], data[0:4])
	if string(h.Magic[:]) != "CMUT" {
		return nil, fmt.Errorf("format: invalid mutation magic: %q", string(h.Magic[:]))
	}
	h.Timestamp = int64(binary.LittleEndian.Uint64(data[4:12]))
	h.DiffPatchSize = binary.LittleEndian.Uint32(data[12:16])
	return h, nil
}

// NumBlocks returns the expected number of Zstd blocks for this file (V2+).
func (h *Header) NumBlocks() uint32 {
	return (h.ChunkCount + ChunksPerBlock - 1) / ChunksPerBlock
}

// ChunkEntry represents a single chunk mapping in the Chunk Table.
// It maps a Codebook codeword to a corresponding XOR residual in the Delta Pool.
type ChunkEntry struct {
	CodebookID    uint64 // The ID of the closest pattern in the Codebook.
	DeltaOffset   uint64 // Offset within the DECOMPRESSED delta block.
	DeltaSize     uint32 // Size of the delta in the decompressed pool.
	OriginalSize  uint32 // Original uncompressed size of this chunk.
	CodebookIndex uint8  // NEW in V7: Which Codebook was used (0=L1, 1=L2, 2=L3, 255=Literal)
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

	if h.Version >= Version2 && h.Version <= Version9 {
		minSize := HeaderSizeV2
		if h.Version == Version4 {
			minSize = HeaderSizeV4
		} else if h.Version == Version5 {
			minSize = HeaderSizeV5
		} else if h.Version == Version6 || h.Version == Version7 {
			minSize = HeaderSizeV6
		} else if h.Version == Version8 || h.Version == Version9 {
			minSize = HeaderSizeV8
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
			copy(h.CodebookHash[:], data[72:80]) // Legacy location
		}
		if h.Version >= Version5 {
			copy(h.MerkleRoot[:], data[80:112])
		}
		if h.Version >= Version6 {
			h.IsConvergentEncrypted = data[112] == 1
			copy(h.CodebookHashes[0][:], data[113:121])
			copy(h.CodebookHashes[1][:], data[121:129])
			copy(h.CodebookHashes[2][:], data[129:137])
			// Mirror the first hash to the legacy field for compatibility wrappers
			copy(h.CodebookHash[:], data[113:121])
		}
		if h.Version >= Version8 {
			h.MicroDictSize = binary.LittleEndian.Uint32(data[137:141])
			if h.MicroDictSize > MaxMicroDictSize {
				return nil, fmt.Errorf("format: v8 micro-dict size %d exceeds safety cap %d (OOM defense)", h.MicroDictSize, MaxMicroDictSize)
			}
			if h.MicroDictSize > 0 {
				if len(data) < int(HeaderSizeV8)+int(h.MicroDictSize) {
					return nil, fmt.Errorf("format: header too small for v8 micro-dict payload (%d < %d)", len(data), int(HeaderSizeV8)+int(h.MicroDictSize))
				}
				h.MicroDictionaryData = make([]byte, h.MicroDictSize)
				copy(h.MicroDictionaryData, data[141:141+h.MicroDictSize])
			}
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

	// Default to V8 if not explicitly set and not V1
	if h.Version < Version2 || h.Version > Version8 {
		h.Version = Version8
	}
	
	size := HeaderSizeV2
	if h.Version == Version4 {
		size = HeaderSizeV4
	} else if h.Version == Version5 {
		size = HeaderSizeV5
	} else if h.Version == Version6 || h.Version == Version7 {
		size = HeaderSizeV6
	} else if h.Version >= Version8 {
		size = HeaderSizeV8 + int(h.MicroDictSize)
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
		if h.Version >= Version6 {
			// In V6/V7, CodebookHash legacy slot still gets [0] for older extractors viewing the first 80 bytes
			copy(buf[72:80], h.CodebookHashes[0][:])
		} else {
			copy(buf[72:80], h.CodebookHash[:])
		}
	}
	
	if h.Version >= Version5 {
		copy(buf[80:112], h.MerkleRoot[:])
	}

	if h.Version >= Version6 {
		if h.IsConvergentEncrypted {
			buf[112] = 1
		}
		copy(buf[113:121], h.CodebookHashes[0][:])
		copy(buf[121:129], h.CodebookHashes[1][:])
		copy(buf[129:137], h.CodebookHashes[2][:])
	}
	
	if h.Version >= Version8 {
		binary.LittleEndian.PutUint32(buf[137:141], h.MicroDictSize)
		if h.MicroDictSize > 0 {
			copy(buf[141:141+h.MicroDictSize], h.MicroDictionaryData)
		}
	}
	
	return buf
}

// GetEntrySize returns the expected ChunkEntry size given the version.
func GetEntrySize(version uint16) uint32 {
	if version >= Version7 {
		return EntrySizeV7
	}
	return EntrySizeV6
}

// ParseChunkTable decodes the contiguous slice of ChunkEntries.
func ParseChunkTable(data []byte, count uint32, version uint16) ([]ChunkEntry, error) {
	entrySize := int(GetEntrySize(version))
	expectedLen := int(count) * entrySize
	if len(data) < expectedLen {
		return nil, errors.New("format: chunk table data too short")
	}

	entries := make([]ChunkEntry, count)
	for i := uint32(0); i < count; i++ {
		offset := int(i) * entrySize
		entry := ChunkEntry{
			CodebookID:   binary.LittleEndian.Uint64(data[offset : offset+8]),
			DeltaOffset:  binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
			DeltaSize:    binary.LittleEndian.Uint32(data[offset+16 : offset+20]),
			OriginalSize: binary.LittleEndian.Uint32(data[offset+20 : offset+24]),
		}
		if version >= Version7 {
			entry.CodebookIndex = data[offset+24]
		}
		entries[i] = entry
	}
	return entries, nil
}

// Serialize encodes a slice of ChunkEntries into a contiguous byte slice.
func SerializeChunkTable(entries []ChunkEntry, version uint16) []byte {
	entrySize := int(GetEntrySize(version))
	buf := make([]byte, len(entries)*entrySize)
	for i, e := range entries {
		offset := i * entrySize
		binary.LittleEndian.PutUint64(buf[offset:offset+8], e.CodebookID)
		binary.LittleEndian.PutUint64(buf[offset+8:offset+16], e.DeltaOffset)
		binary.LittleEndian.PutUint32(buf[offset+16:offset+20], e.DeltaSize)
		binary.LittleEndian.PutUint32(buf[offset+20:offset+24], e.OriginalSize)
		if version >= Version7 {
			buf[offset+24] = e.CodebookIndex
		}
	}
	return buf
}
