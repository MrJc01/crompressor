// Package delta provides public access to CROM delta compression operations.
//
// This package re-exports functions from internal/delta for use by
// satellite repositories (crompressor-sync, etc).
package delta

import (
	"github.com/MrJc01/crompressor/internal/delta"
)

// CompressPool compresses a delta pool using zstd.
func CompressPool(pool []byte) ([]byte, error) {
	return delta.CompressPool(pool)
}

// DecompressPool decompresses a zstd-compressed delta pool.
func DecompressPool(compressed []byte) ([]byte, error) {
	return delta.DecompressPool(compressed)
}

// XOR computes the byte-wise XOR between original and pattern.
func XOR(original []byte, pattern []byte) []byte {
	return delta.XOR(original, pattern)
}

// Apply applies a XOR delta to a pattern to reconstruct the original.
func Apply(pattern []byte, d []byte) []byte {
	return delta.Apply(pattern, d)
}

// Diff produces a compact binary patch from original to pattern.
func Diff(original, pattern []byte) []byte {
	return delta.Diff(original, pattern)
}

// ApplyPatch applies a binary patch to a pattern to reconstruct the original.
func ApplyPatch(pattern, script []byte) ([]byte, error) {
	return delta.ApplyPatch(pattern, script)
}
