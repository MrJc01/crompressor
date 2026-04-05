package delta

import (
	"bytes"
	"testing"
)

func TestXOR_VariableSizes(t *testing.T) {
	pattern := []byte("1234") // length 4

	tests := []struct {
		name     string
		original []byte
	}{
		{
			name:     "Equal Length",
			original: []byte("ABCD"),
		},
		{
			name:     "Original Larger Than Pattern",
			original: []byte("ABCDEFGH"), // length 8
		},
		{
			name:     "Original Smaller Than Pattern",
			original: []byte("AB"), // length 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate Delta
			d := XOR(tt.original, pattern)

			// Delta MUST have the same size as Original, because Delta encapsulates everything
			// needed to reconstruct the Original (including the trailing bytes if it's larger).
			if len(d) != len(tt.original) {
				t.Fatalf("Delta length %d != Original length %d", len(d), len(tt.original))
			}

			// Apply Delta
			restored := Apply(pattern, d)

			// Restored MUST exactly match Original
			if !bytes.Equal(restored, tt.original) {
				t.Fatalf("Restored mismatch! Expected %v, Got %v", tt.original, restored)
			}
		})
	}
}
