// Package codebook provides read access to CROM Codebook files (.cromdb).
// The Codebook is a static binary database of codewords (byte patterns) that
// serves as the Universal Pattern Dictionary for the CROM compression system.
package codebook

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// HeaderSize is the fixed size of the .cromdb header in bytes.
	HeaderSize = 512

	// MagicString is the magic identifier at the start of every .cromdb file.
	MagicString = "CROMDB"

	// MagicSize is the number of bytes used by the magic identifier.
	MagicSize = 6

	// Version1 is the current format version.
	Version1 uint16 = 1

	// BuildHashSize is the size of the SHA-256 build hash.
	BuildHashSize = 32
)

// Header represents the parsed header of a .cromdb file.
//
// Binary layout (512 bytes total):
//
//	Offset  Size   Field
//	0       6      Magic ("CROMDB")
//	6       2      Version (uint16 LE)
//	8       2      CodewordSize (uint16 LE)
//	10      8      CodewordCount (uint64 LE)
//	18      8      DataOffset (uint64 LE) — where codeword data begins
//	26      32     BuildHash (SHA-256 of codeword data)
//	58      454    Reserved (zero-padded)
type Header struct {
	Magic         [MagicSize]byte
	Version       uint16
	CodewordSize  uint16
	CodewordCount uint64
	DataOffset    uint64
	BuildHash     [BuildHashSize]byte
}

// ParseHeader reads and validates a Header from a byte slice (must be >= HeaderSize).
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("codebook: data too short for header: %d < %d", len(data), HeaderSize)
	}

	h := &Header{}

	// Magic
	copy(h.Magic[:], data[0:MagicSize])
	if string(h.Magic[:]) != MagicString {
		return nil, fmt.Errorf("codebook: invalid magic: got %q, want %q", string(h.Magic[:]), MagicString)
	}

	// Version
	h.Version = binary.LittleEndian.Uint16(data[6:8])
	if h.Version != Version1 {
		return nil, fmt.Errorf("codebook: unsupported version: %d", h.Version)
	}

	// Codeword Size
	h.CodewordSize = binary.LittleEndian.Uint16(data[8:10])
	if h.CodewordSize == 0 {
		return nil, errors.New("codebook: codeword size cannot be zero")
	}

	// Codeword Count
	h.CodewordCount = binary.LittleEndian.Uint64(data[10:18])

	// Data Offset
	h.DataOffset = binary.LittleEndian.Uint64(data[18:26])
	if h.DataOffset < HeaderSize {
		return nil, fmt.Errorf("codebook: data offset %d is within header region", h.DataOffset)
	}

	// Build Hash
	copy(h.BuildHash[:], data[26:58])

	return h, nil
}

// Serialize writes the header to a byte slice of exactly HeaderSize bytes.
func (h *Header) Serialize() []byte {
	buf := make([]byte, HeaderSize)

	copy(buf[0:MagicSize], h.Magic[:])
	binary.LittleEndian.PutUint16(buf[6:8], h.Version)
	binary.LittleEndian.PutUint16(buf[8:10], h.CodewordSize)
	binary.LittleEndian.PutUint64(buf[10:18], h.CodewordCount)
	binary.LittleEndian.PutUint64(buf[18:26], h.DataOffset)
	copy(buf[26:58], h.BuildHash[:])
	// Remaining bytes 58..511 are zero (reserved).

	return buf
}
