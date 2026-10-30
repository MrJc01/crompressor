package delta

import (
	"bytes"
	"fmt"
	"io"

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

const (
	brainDictID     = 0x43524F4D // "CROM"
	brainDictWindow = 32 * 1024 * 1024
)

func brainEncoderOptions(dict []byte) []zstd.EOption {
	opts := []zstd.EOption{
		zstd.WithEncoderLevel(zstd.SpeedBestCompression),
		zstd.WithEncoderConcurrency(1),
	}
	if len(dict) > 0 {
		opts = append(opts, zstd.WithWindowSize(brainDictWindow), zstd.WithEncoderDictRaw(brainDictID, dict))
	}
	return opts
}

func brainDecoderOptions(dict []byte) []zstd.DOption {
	opts := []zstd.DOption{zstd.WithDecoderConcurrency(1)}
	if len(dict) > 0 {
		// The C encoder writes frames without a dictionary ID.
		opts = append(opts, zstd.WithDecoderDictRaw(brainDictID, dict), zstd.WithDecoderDictRaw(0, dict))
	}
	return opts
}

// CompressPoolDict compresses a pool using the shared brain as a zstd dictionary, so
// data that is only similar to the brain also benefits from it. A nil dict compresses plainly.
func CompressPoolDict(pool, dict []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf, brainEncoderOptions(dict)...)
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

// DecompressPoolDict reverses CompressPoolDict.
func DecompressPoolDict(compressed, dict []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(compressed), brainDecoderOptions(dict)...)
	if err != nil {
		return nil, fmt.Errorf("delta: init zstd decoder: %w", err)
	}
	defer dec.Close()
	out, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("delta: decompress pool: %w", err)
	}
	return out, nil
}

// BrainWriter compresses a stream with the shared brain as dictionary.
type BrainWriter struct{ *zstd.Encoder }

// NewBrainWriter returns a compressing writer over w.
func NewBrainWriter(w io.Writer, dict []byte) (*BrainWriter, error) {
	enc, err := zstd.NewWriter(w, brainEncoderOptions(dict)...)
	if err != nil {
		return nil, fmt.Errorf("delta: init stream encoder: %w", err)
	}
	return &BrainWriter{enc}, nil
}

// NewBrainReader returns a decompressing reader over r.
func NewBrainReader(r io.Reader, dict []byte) (*zstd.Decoder, error) {
	dec, err := zstd.NewReader(r, brainDecoderOptions(dict)...)
	if err != nil {
		return nil, fmt.Errorf("delta: init stream decoder: %w", err)
	}
	return dec, nil
}
