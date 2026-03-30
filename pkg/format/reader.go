package format

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/MrJc01/crompressor/internal/crypto"
)

// Reader reads and parses a .crom file sequentially.
type Reader struct {
	r io.Reader
}

// NewReader creates a new Reader around an io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// ReadMetadata parses the .crom file sequentially up to the Chunk Table.
// It does NOT load the Delta Pool into memory, leaving the underlying reader at the start of the pool.
func (cr *Reader) ReadMetadata(encryptionKey string) (*Header, []uint32, []ChunkEntry, error) {
	// 1. Read Magic + Version (6 bytes)
	metaBuf := make([]byte, 6)
	if _, err := io.ReadFull(cr.r, metaBuf); err != nil {
		return nil, nil, nil, fmt.Errorf("format: read magic/version: %w", err)
	}

	version := binary.LittleEndian.Uint16(metaBuf[4:6])

	var headerBuf []byte
	if version == Version1 {
		headerBuf = make([]byte, HeaderSize)
		copy(headerBuf, metaBuf)
		if _, err := io.ReadFull(cr.r, headerBuf[6:]); err != nil {
			return nil, nil, nil, fmt.Errorf("format: read v1 header: %w", err)
		}
	} else if version >= Version2 && version <= Version9 {
		size := HeaderSizeV2
		if version == Version4 {
			size = HeaderSizeV4
		} else if version == Version5 {
			size = HeaderSizeV5
		} else if version == Version6 || version == Version7 {
			size = HeaderSizeV6
		} else if version == Version8 || version == Version9 {
			size = HeaderSizeV8
		}
		
		headerBuf = make([]byte, size)
		copy(headerBuf, metaBuf)
		if _, err := io.ReadFull(cr.r, headerBuf[6:]); err != nil {
			return nil, nil, nil, fmt.Errorf("format: read v%d header (base): %w", version, err)
		}

		// Se for V8, o header estendido contém o tamanho do array dinâmico MicroDictSize
		if version >= Version8 {
			microDictSize := binary.LittleEndian.Uint32(headerBuf[137:141])
			if microDictSize > MaxMicroDictSize {
				return nil, nil, nil, fmt.Errorf("format: v8 microdict size %d exceeds safety cap (OOM defense)", microDictSize)
			}
			if microDictSize > 0 {
				payloadBuf := make([]byte, microDictSize)
				if _, err := io.ReadFull(cr.r, payloadBuf); err != nil {
					return nil, nil, nil, fmt.Errorf("format: read v8 microdict payload: %w", err)
				}
				headerBuf = append(headerBuf, payloadBuf...)
			}
		}

	} else {
		return nil, nil, nil, fmt.Errorf("format: unsupported version %d", version)
	}

	header, err := ParseHeader(headerBuf)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("format: parse header: %w", err)
	}

	var derivedKey []byte
	if header.IsEncrypted {
		if encryptionKey == "" {
			return nil, nil, nil, fmt.Errorf("format: file is encrypted but no key was provided")
		}
		derivedKey = crypto.DeriveKey([]byte(encryptionKey), header.Salt[:])
	}

	// 2. Read Block Table (V2/V3 only)
	var blockTable []uint32
	if header.Version >= Version2 {
		numBlocks := header.NumBlocks()
		blockTableSize := int(numBlocks) * 4
		blockBuf := make([]byte, blockTableSize)
		if _, err := io.ReadFull(cr.r, blockBuf); err != nil {
			return nil, nil, nil, fmt.Errorf("format: read block table: %w", err)
		}
		blockTable = make([]uint32, numBlocks)
		for i := uint32(0); i < numBlocks; i++ {
			blockTable[i] = binary.LittleEndian.Uint32(blockBuf[i*4 : i*4+4])
		}
	}

	// 3. Read Chunk Table
	// If encrypted, the length of the ciphertext is greater due to the 12-byte nonce and 16-byte tag (28 bytes overhead).
	tableSize := int(header.ChunkCount) * int(GetEntrySize(header.Version))
	if header.IsEncrypted {
		tableSize += 28
	}

	if tableSize < 0 {
		return nil, nil, nil, fmt.Errorf("format: chunk map size overflow: %d entries", header.ChunkCount)
	}

	tableBuf := make([]byte, tableSize)
	if _, err := io.ReadFull(cr.r, tableBuf); err != nil {
		return nil, nil, nil, fmt.Errorf("format: read chunk table: %w", err)
	}

	if header.IsEncrypted {
		tableBuf, err = crypto.Decrypt(derivedKey, tableBuf)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("format: decrypt chunk table: %w", err)
		}
	}

	entries, err := ParseChunkTable(tableBuf, header.ChunkCount, header.Version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("format: parse chunk table: %w", err)
	}

	return header, blockTable, entries, nil
}

// ReadStream returns the underlying reader positioned exactly at the start of the compressed delta pool.
func (cr *Reader) ReadStream(encryptionKey string) (*Header, []uint32, []ChunkEntry, io.Reader, error) {
	header, blockTable, entries, err := cr.ReadMetadata(encryptionKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return header, blockTable, entries, cr.r, nil
}
