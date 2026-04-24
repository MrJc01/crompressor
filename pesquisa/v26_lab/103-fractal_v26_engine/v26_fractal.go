package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// =============================================================================
// CROM V26 — Motor Fractal Gerativo Multiestratégia (V2 — Com TTL)
// =============================================================================

const chunkSize = 8 // 8 bytes para viabilidade de brute-force

type Strategy struct {
	Name string
	Fn   func(seed int64, length int) []byte
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║   CROM V26 - MOTOR FRACTAL MULTIESTRATÉGIA       ║")
	fmt.Println("╠═══════════════════════════════════════════════════╣")
	fmt.Println("║  Estratégias: PRNG | XOR Fractal | Poly | Fib    ║")
	fmt.Println("║  TTL: 3s por estratégia (Chaitin Fallback)       ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	targetChunk := []byte{10, 20, 30, 40, 50, 60, 70, 80}

	fmt.Printf("\n[ALVO] Chunk de %d bytes: %v\n", len(targetChunk), targetChunk)

	strategies := []Strategy{
		{"PRNG (Go rand)", strategyPRNG},
		{"XOR Fractal Iterado", strategyXOR},
		{"Polinomial (ax²+bx+c mod 256)", strategyPoly},
		{"Fibonacci Modulado", strategyFib},
	}

	ttl := 3 * time.Second
	fmt.Printf("[CONFIG] TTL por estratégia: %v\n\n", ttl)

	type Result struct {
		Strategy string
		Found    bool
		Seed     int64
		Duration time.Duration
		Tested   int64
	}

	results := make([]Result, len(strategies))

	for i, s := range strategies {
		fmt.Printf("▶ Testando estratégia: %s ...\n", s.Name)
		ctx, cancel := context.WithTimeout(context.Background(), ttl)
		start := time.Now()
		found := false
		var winnerSeed int64
		var tested int64

		for seed := int64(0); ; seed++ {
			select {
			case <-ctx.Done():
				tested = seed
				goto done
			default:
			}
			candidate := s.Fn(seed, len(targetChunk))
			if bytes.Equal(candidate, targetChunk) {
				found = true
				winnerSeed = seed
				tested = seed + 1
				goto done
			}
			// Check context every 100K seeds to avoid overhead
			if seed%100000 == 0 {
				select {
				case <-ctx.Done():
					tested = seed
					goto done
				default:
				}
			}
		}
	done:
		cancel()
		dur := time.Since(start)

		results[i] = Result{
			Strategy: s.Name,
			Found:    found,
			Seed:     winnerSeed,
			Duration: dur,
			Tested:   tested,
		}

		if found {
			fmt.Printf("  ✔ MATCH! Seed=%d em %v (%d tentativas)\n", winnerSeed, dur, tested)
		} else {
			fmt.Printf("  ✗ TTL expirado em %v (%d tentativas)\n", dur, tested)
		}
	}

	// Relatório
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                 RELATÓRIO COMPARATIVO V26                 ║")
	fmt.Println("╠═════════════════════════╦═══════╦════════════╦════════════╣")
	fmt.Println("║ Estratégia              ║ Match ║ Seed       ║ Tempo      ║")
	fmt.Println("╠═════════════════════════╬═══════╬════════════╬════════════╣")
	for _, r := range results {
		match := "✗"
		seedStr := "N/A"
		if r.Found {
			match = "✔"
			seedStr = fmt.Sprintf("%d", r.Seed)
		}
		fmt.Printf("║ %-23s ║ %-5s ║ %-10s ║ %-10s ║\n",
			r.Strategy, match, seedStr, r.Duration.Truncate(time.Millisecond))
	}
	fmt.Println("╚═════════════════════════╩═══════╩════════════╩════════════╝")

	entropyBits := shannonEntropy(targetChunk)
	fmt.Printf("\n  Entropia do Chunk: %.2f bits/byte (máx: 8.00)\n", entropyBits)
}

// === ESTRATÉGIAS ===

func strategyPRNG(seed int64, length int) []byte {
	r := rand.New(rand.NewSource(seed))
	out := make([]byte, length)
	r.Read(out)
	return out
}

func strategyXOR(seed int64, length int) []byte {
	out := make([]byte, length)
	s := byte(seed & 0xFF)
	a := byte((seed >> 8) & 0xFF)
	if a == 0 { a = 1 }
	for i := range out {
		s = s ^ (a + byte(i))
		out[i] = s
	}
	return out
}

func strategyPoly(seed int64, length int) []byte {
	a := byte(seed & 0xFF)
	b := byte((seed >> 8) & 0xFF)
	c := byte((seed >> 16) & 0xFF)
	out := make([]byte, length)
	for i := range out {
		x := byte(i)
		out[i] = a*x*x + b*x + c
	}
	return out
}

func strategyFib(seed int64, length int) []byte {
	a := byte(seed & 0xFF)
	b := byte((seed >> 8) & 0xFF)
	out := make([]byte, length)
	if length > 0 { out[0] = a }
	if length > 1 { out[1] = b }
	for i := 2; i < length; i++ {
		out[i] = out[i-1] + out[i-2]
	}
	return out
}

func shannonEntropy(data []byte) float64 {
	freq := make(map[byte]float64)
	for _, b := range data { freq[b]++ }
	n := float64(len(data))
	entropy := 0.0
	for _, count := range freq {
		p := count / n
		if p > 0 { entropy -= p * math.Log2(p) }
	}
	return entropy
}
