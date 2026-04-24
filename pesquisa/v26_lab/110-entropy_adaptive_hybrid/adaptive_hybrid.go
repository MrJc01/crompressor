package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Pesquisa 110: Motor Híbrido Adaptativo V2
// Otimizado: PRNG search 50K, sem expansão de dados

type CompressionMethod int

const (
	MethodLiteral CompressionMethod = iota
	MethodPRNG
	MethodPoly
	MethodFib
)

func (m CompressionMethod) String() string {
	switch m {
	case MethodLiteral: return "LITERAL"
	case MethodPRNG:    return "PRNG"
	case MethodPoly:    return "POLINOMIAL"
	case MethodFib:     return "FIBONACCI"
	}
	return "UNKNOWN"
}

type ChunkResult struct {
	Method   CompressionMethod
	Seed     int64
	Size     int
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  110 - ENTROPY ADAPTIVE HYBRID ENGINE (V2)       ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	chunks := [][]byte{
		{10, 20, 30, 40, 50, 60, 70, 80},
		{0, 1, 1, 2, 3, 5, 8, 13},
		{145, 23, 99, 211, 45, 11, 201, 89},
		{255, 255, 255, 255, 255, 255, 255, 255},
		{1, 4, 9, 16, 25, 36, 49, 64},
	}

	fmt.Printf("[ARQUIVO] %d chunks de 8 bytes cada\n\n", len(chunks))

	start := time.Now()
	totalOrig := 0
	totalComp := 0

	type Row struct {
		ID      int
		Entropy float64
		Method  CompressionMethod
		Orig    int
		Comp    int
	}
	rows := make([]Row, len(chunks))

	for i, chunk := range chunks {
		ent := shannonEntropy(chunk)
		result := hybridCompress(chunk, ent)
		totalOrig += len(chunk)
		totalComp += result.Size

		rows[i] = Row{i, ent, result.Method, len(chunk), result.Size}
		fmt.Printf("  Chunk %d: entropy=%.2f → %s (%d→%d bytes)\n",
			i, ent, result.Method, len(chunk), result.Size)
	}

	dur := time.Since(start)

	fmt.Println("\n╔═════╦══════════╦══════════════╦═════════╦══════════╗")
	fmt.Println("║  #  ║ Entropia ║ Método       ║ Orig    ║ Comp     ║")
	fmt.Println("╠═════╬══════════╬══════════════╬═════════╬══════════╣")
	for _, r := range rows {
		fmt.Printf("║  %d  ║  %.2f    ║ %-12s ║ %d B     ║ %d B      ║\n",
			r.ID, r.Entropy, r.Method, r.Orig, r.Comp)
	}
	fmt.Println("╚═════╩══════════╩══════════════╩═════════╩══════════╝")
	fmt.Printf("\n  Original:   %d bytes\n", totalOrig)
	fmt.Printf("  Comprimido: %d bytes\n", totalComp)
	if totalOrig > 0 {
		fmt.Printf("  Razão:      %.1f%%\n", float64(totalComp)/float64(totalOrig)*100)
	}
	fmt.Printf("  Tempo:      %v\n", dur)
}

func hybridCompress(chunk []byte, entropy float64) ChunkResult {
	origSize := len(chunk)
	seedSize := 8 + 1 // int64 seed + method tag

	// Constante/repetição → RLE
	if entropy < 0.5 {
		return ChunkResult{MethodLiteral, 0, 2}
	}

	// Fibonacci (rápido, O(65K))
	for seed := int64(0); seed < 65536; seed++ {
		if bytes.Equal(fibGen(seed, origSize), chunk) {
			if seedSize < origSize {
				return ChunkResult{MethodFib, seed, seedSize}
			}
		}
	}

	// Polinomial (O(16M) mas check rápido)
	for seed := int64(0); seed < 65536; seed++ {
		if bytes.Equal(polyGen(seed, origSize), chunk) {
			if seedSize < origSize {
				return ChunkResult{MethodPoly, seed, seedSize}
			}
		}
	}

	// PRNG (reduzido a 50K para velocidade)
	for seed := int64(0); seed < 10000; seed++ {
		r := rand.New(rand.NewSource(seed))
		c := make([]byte, origSize)
		r.Read(c)
		if bytes.Equal(c, chunk) {
			if seedSize < origSize {
				return ChunkResult{MethodPRNG, seed, seedSize}
			}
		}
	}

	// Fallback: literal (NUNCA expandir)
	return ChunkResult{MethodLiteral, 0, origSize}
}

func fibGen(seed int64, n int) []byte {
	a, b := byte(seed&0xFF), byte((seed>>8)&0xFF)
	out := make([]byte, n)
	if n > 0 { out[0] = a }
	if n > 1 { out[1] = b }
	for i := 2; i < n; i++ { out[i] = out[i-1] + out[i-2] }
	return out
}

func polyGen(seed int64, n int) []byte {
	a, b, c := byte(seed&0xFF), byte((seed>>8)&0xFF), byte((seed>>16)&0xFF)
	out := make([]byte, n)
	for i := range out { x := byte(i); out[i] = a*x*x + b*x + c }
	return out
}

func shannonEntropy(data []byte) float64 {
	freq := make(map[byte]float64)
	for _, b := range data { freq[b]++ }
	n := float64(len(data))
	e := 0.0
	for _, c := range freq { p := c / n; if p > 0 { e -= p * math.Log2(p) } }
	return e
}
