package codebook

import (
	"fmt"
	"os"
)

// ReadPatterns loads all codeword patterns from a .cromdb file and returns
// them as a slice of byte slices. This is used by the trainer for incremental
// updates (--update) and transfer learning (--base).
func ReadPatterns(path string) ([][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("codebook: read file: %w", err)
	}

	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("codebook: parse header: %w", err)
	}

	cwSize := uint64(header.CodewordSize)
	count := header.CodewordCount
	offset := header.DataOffset

	expectedEnd := offset + cwSize*count
	if uint64(len(data)) < expectedEnd {
		return nil, fmt.Errorf(
			"codebook: file truncated: size=%d, expected at least %d for %d codewords",
			len(data), expectedEnd, count,
		)
	}

	patterns := make([][]byte, 0, count)
	for i := uint64(0); i < count; i++ {
		start := offset + i*cwSize
		end := start + cwSize
		p := make([]byte, cwSize)
		copy(p, data[start:end])
		patterns = append(patterns, p)
	}

	return patterns, nil
}
