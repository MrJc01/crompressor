// Package delta provides the lossless refinement logic for the CROM system.
// It computes exact residuals (deltas) between a data chunk and its closest
// matching pattern, and perfectly reconstructs the original data.
package delta

// XOR computes the byte-wise XOR difference between original and pattern.
// Both slices must have the same length.
//
// original ^ pattern = delta
//
// 0 ^ 0 = 0
// 1 ^ 1 = 0
// 1 ^ 0 = 1
// XOR computes the byte-wise XOR difference between original and pattern.
// If original is longer than pattern, the remaining bytes of original are
// kept exactly as they are (XOR with 0).
func XOR(original []byte, pattern []byte) []byte {
	nOrig := len(original)
	nPat := len(pattern)
	
	delta := make([]byte, nOrig)
	
	for i := 0; i < nOrig; i++ {
		if i < nPat {
			delta[i] = original[i] ^ pattern[i]
		} else {
			delta[i] = original[i]
		}
	}

	return delta
}

// Apply applies a delta to a pattern to reconstruct the original data.
// Since delta represents the exact footprint of the original, the
// returned slice has length len(delta).
func Apply(pattern []byte, delta []byte) []byte {
	nDelta := len(delta)
	nPat := len(pattern)

	original := make([]byte, nDelta)
	for i := 0; i < nDelta; i++ {
		if i < nPat {
			original[i] = pattern[i] ^ delta[i]
		} else {
			original[i] = delta[i]
		}
	}

	return original
}
