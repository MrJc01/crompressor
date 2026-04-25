// Package entropy provides public access to CROM entropy analysis operations.
//
// This package re-exports functions from internal/entropy for use by
// satellite repositories (crompressor-wasm, etc).
package entropy

import (
	"github.com/MrJc01/crompressor/internal/entropy"
)

// Shannon computes the Shannon entropy (bits per byte) of the given data.
// Returns a value between 0.0 (perfectly uniform) and 8.0 (perfectly random).
func Shannon(data []byte) float64 {
	return entropy.Shannon(data)
}

// IsLowEntropy returns true if the entropy score indicates trivially compressible data.
func IsLowEntropy(score float64) bool {
	return entropy.IsLowEntropy(score)
}
