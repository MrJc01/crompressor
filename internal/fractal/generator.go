package fractal

import (
	"bytes"
	"math/rand"
)

// FractalCompressor é a implementação da V26 (Compressão Algorítmica Fractal)
// Ele tenta achar uma semente geradora O(1) que produza o exato output aleatório (alta entropia).
type FractalCompressor struct{}

// FindGeneratingSeed realiza uma busca heurística por uma Semente PRNG Caótica
// que consiga cuspir exatamente os mesmos bytes que o chunk alvo.
// Uma implementação real demoraria eras de computação, mas esta é a PoC da V26.
func FindGeneratingSeed(targetChunk []byte, maxIterations int) (seed int64, match bool) {
	for i := int64(0); i < int64(maxIterations); i++ {
		// Inicializa o gerador caótico com a semente candidata
		pseudo := rand.New(rand.NewSource(i))
		candidate := make([]byte, len(targetChunk))
		pseudo.Read(candidate)

		// Verifica se o Fractal gerou os dados originais
		if bytes.Equal(candidate, targetChunk) {
			return i, true // Achamos a equação geradora! (Compressão Infinita)
		}
	}
	return 0, false // Sem convergência neste nível de profundidade recursiva
}

// GeneratePolynomial implements a polynomial (ax^2 + bx + c mod 256) sequence generator
// where a, b, and c are extracted from the 24-bit seed. This is used for O(1) reconstruction during unpack.
func GeneratePolynomial(seed int64, length int) []byte {
	a := byte(seed & 0xFF)
	b := byte((seed >> 8) & 0xFF)
	c := byte((seed >> 16) & 0xFF)
	out := make([]byte, length)
	for i := range out {
		x := uint64(i)
		out[i] = byte(uint64(a)*x*x + uint64(b)*x + uint64(c))
	}
	return out
}

// FindPolynomial searches the polynomial sequence space (up to 24-bits / 16.7M options).
// If it finds a match, it returns true and the seed.
// O(1) storage, reconstructed by evaluating the polynomial.
func FindPolynomial(targetChunk []byte) (bool, int64) {
	if len(targetChunk) == 0 {
		return false, 0
	}
	
	// Algebraic Optimization:
	// a*x^2 + b*x + c = targetChunk[x]
	// At x = 0, targetChunk[0] = c
	c := targetChunk[0]
	
	if len(targetChunk) == 1 {
		seed := int64(c) << 16
		return true, seed
	}
	
	// At x = 1, targetChunk[1] = a + b + c  =>  targetChunk[1] - c = a + b
	// We only need to guess 'a' from 0 to 255, and b is fixed: b = targetChunk[1] - c - a
	for a := 0; a <= 255; a++ {
		b := int(targetChunk[1]) - int(c) - a
		bb := byte(b)
		aa := byte(a)
		
		match := true
		for i := 2; i < len(targetChunk); i++ {
			x := byte(i)
			val := aa*x*x + bb*x + c
			if val != targetChunk[i] {
				match = false
				break
			}
		}
		
		if match {
			seed := int64(aa) | (int64(bb) << 8) | (int64(c) << 16)
			return true, seed
		}
	}
	return false, 0
}
