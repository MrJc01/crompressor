package search

import (
	"crypto/rand"
	"testing"
)

func BenchmarkHammingDistance(b *testing.B) {
	// 4KB chunks (common block size in chunker)
	chunkA := make([]byte, 4096)
	chunkB := make([]byte, 4096)
	rand.Read(chunkA)
	rand.Read(chunkB)

	b.Run("Standard", func(b *testing.B) {
		b.SetBytes(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = hammingDistance(chunkA, chunkB)
		}
	})

	b.Run("SIMD_Unrolled", func(b *testing.B) {
		b.SetBytes(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = hammingDistanceSIMD(chunkA, chunkB)
		}
	})
}
