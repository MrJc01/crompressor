package format

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Writer writes a properly formatted .crom file sequentially.
type Writer struct {
	w io.Writer
}

// NewWriter creates a new Writer around an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write writes the header, block map (if v2), chunk table, and delta pool to the underlying writer.
// Important: the deltaPool should already be Zstandard compressed.
func (cw *Writer) Write(header *Header, blockTable []uint32, entries []ChunkEntry, compDeltaPool []byte) error {
	if header.ChunkCount != uint32(len(entries)) {
		return fmt.Errorf("format: mismatch between Header.ChunkCount (%d) and entries length (%d)",
			header.ChunkCount, len(entries))
	}

	// 1. Write Header
	if _, err := cw.w.Write(header.Serialize()); err != nil {
		return fmt.Errorf("format: write header: %w", err)
	}

	// 2. Write Block Table (V2 only)
	if header.Version == Version2 && len(blockTable) > 0 {
		buf := make([]byte, len(blockTable)*4)
		for i, size := range blockTable {
			binary.LittleEndian.PutUint32(buf[i*4:], size)
		}
		if _, err := cw.w.Write(buf); err != nil {
			return fmt.Errorf("format: write block table: %w", err)
		}
	}

	// 3. Write Chunk Table
	tableData := SerializeChunkTable(entries, header.Version)
	if _, err := cw.w.Write(tableData); err != nil {
		return fmt.Errorf("format: write chunk table: %w", err)
	}

	// 4. Write compressed Delta Pool (remaining bytes)
	if _, err := cw.w.Write(compDeltaPool); err != nil {
		return fmt.Errorf("format: write delta pool: %w", err)
	}

	return nil
}
