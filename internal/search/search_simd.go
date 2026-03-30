package search

import (
	"encoding/binary"
	"math"
	"math/bits"
)

// hammingDistanceSIMD computes the Hamming distance processing 32 bytes (256 bits) at a time.
// This loop unrolling strategy takes advantage of modern CPU Instruction-Level Parallelism (ILP)
// and allows the Go compiler to vectorize the _mm256 operations natively without unsafe assembly.
func hammingDistanceSIMD(a, b []byte) int {
	dist := 0
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	blocks32 := minLen / 32
	for i := 0; i < blocks32; i++ {
		offset := i * 32
		
		// 32-byte chunks (256 bits per cycle)
		v1 := binary.LittleEndian.Uint64(a[offset : offset+8])
		v2 := binary.LittleEndian.Uint64(b[offset : offset+8])
		
		v3 := binary.LittleEndian.Uint64(a[offset+8 : offset+16])
		v4 := binary.LittleEndian.Uint64(b[offset+8 : offset+16])
		
		v5 := binary.LittleEndian.Uint64(a[offset+16 : offset+24])
		v6 := binary.LittleEndian.Uint64(b[offset+16 : offset+24])
		
		v7 := binary.LittleEndian.Uint64(a[offset+24 : offset+32])
		v8 := binary.LittleEndian.Uint64(b[offset+24 : offset+32])

		// Hardware POPCNT executed in parallel execution ports over 4 words
		dist += bits.OnesCount64(v1^v2) +
			bits.OnesCount64(v3^v4) +
			bits.OnesCount64(v5^v6) +
			bits.OnesCount64(v7^v8)
	}

	// Process remaining 8-byte blocks
	remStart := blocks32 * 32
	blocks8 := (minLen - remStart) / 8
	for i := 0; i < blocks8; i++ {
		offset := remStart + (i * 8)
		v1 := binary.LittleEndian.Uint64(a[offset : offset+8])
		v2 := binary.LittleEndian.Uint64(b[offset : offset+8])
		dist += bits.OnesCount64(v1 ^ v2)
	}

	// Process remaining bytes
	for i := remStart + (blocks8 * 8); i < minLen; i++ {
		dist += bits.OnesCount8(a[i] ^ b[i])
	}

	if len(a) != len(b) {
		dist += int(math.Abs(float64(len(a)-len(b)))) * 8
	}

	return dist
}
