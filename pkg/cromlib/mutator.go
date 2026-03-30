package cromlib

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MrJc01/crompressor/pkg/format"
)

// AppendMutation appends a V9 mutation log to the end of an opened .crom file.
// This allows the CROM filesystem to act as an LSM-Tree, where edits are appended O(1)
// instead of requiring a full decompression, recompression, and rewrite of the file.
func AppendMutation(file *os.File, patchBytes []byte) error {
	if file == nil {
		return fmt.Errorf("cromlib: cannot append mutation to nil file")
	}

	// 1. Seek to the end of the file.
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("cromlib: failed to seek to end of file: %w", err)
	}

	// 2. Prepare the V9 mutation header.
	header := &format.V9MutationHeader{
		Timestamp:     time.Now().Unix(),
		DiffPatchSize: uint32(len(patchBytes)),
	}
	copy(header.Magic[:], "CMUT")

	// 3. Write the header.
	if _, err := file.Write(header.Bytes()); err != nil {
		return fmt.Errorf("cromlib: failed to write mutation header: %w", err)
	}

	// 4. Write the mutation payload.
	if len(patchBytes) > 0 {
		if _, err := file.Write(patchBytes); err != nil {
			return fmt.Errorf("cromlib: failed to write mutation payload: %w", err)
		}
	}

	return nil
}
