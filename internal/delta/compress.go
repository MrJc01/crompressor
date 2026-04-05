package delta

import (
	"bytes"
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// CompressPool uses Zstandard (zstd) to compress a contiguous block of deltas.
// The given byte pool is expected to be highly compressible because it represents
// the "errors" (differences) from the closest patterns, which should contain many zeros.
func CompressPool(pool []byte) ([]byte, error) {
	var buf bytes.Buffer

	// We use the BestCompression level to minimize the Delta Pool size,
	// trading off write speed for maximum compression ratio since packing
	// is typically a write-once operation.
	enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf("delta: init zstd encoder: %w", err)
	}

	if _, err := enc.Write(pool); err != nil {
		enc.Close()
		return nil, fmt.Errorf("delta: compress pool: %w", err)
	}

	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("delta: close zstd encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// DecompressPool decompresses the Zstandard compressed Delta Pool back
// into its original uncompressed form.
func DecompressPool(compressed []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("delta: init zstd decoder: %w", err)
	}
	defer dec.Close()

	// Read all decompressed data. zstd reader automatically stops at EOF.
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(dec); err != nil {
		return nil, fmt.Errorf("delta: decompress pool: %w", err)
	}

	return buf.Bytes(), nil
}
