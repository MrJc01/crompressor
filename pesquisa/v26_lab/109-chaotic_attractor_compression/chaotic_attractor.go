package main

import (
	"fmt"
	"math"
	"time"
)

// Pesquisa 109: Mapear chunks para órbitas de atratores caóticos (Lorenz, Rössler).
// Hipótese: Sequências determinísticas caóticas podem codificar dados com poucos parâmetros iniciais.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  109 - CHAOTIC ATTRACTOR COMPRESSION              ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	target := []byte{145, 23, 99, 211, 45, 11, 201, 89, 33, 177, 66, 240, 12, 88, 190, 55}
	fmt.Printf("[ALVO] %v (%d bytes)\n\n", target, len(target))

	start := time.Now()

	// Testar múltiplos parâmetros do atrator de Lorenz
	bestSigma, bestRho, bestBeta := 0.0, 0.0, 0.0
	bestDist := len(target) + 1
	bestOffset := 0
	tested := 0

	for sigma := 8.0; sigma <= 12.0; sigma += 0.5 {
		for rho := 20.0; rho <= 30.0; rho += 0.5 {
			for beta := 1.0; beta <= 4.0; beta += 0.5 {
				orbit := lorenzOrbit(sigma, rho, beta, 500)
				tested++
				for off := 0; off <= len(orbit)-len(target); off++ {
					dist := hammingDist(target, orbit[off:off+len(target)])
					if dist < bestDist {
						bestDist = dist
						bestSigma = sigma
						bestRho = rho
						bestBeta = beta
						bestOffset = off
						if dist == 0 {
							goto done
						}
					}
				}
			}
		}
	}
done:
	dur := time.Since(start)

	fmt.Println("═══ Resultado ═══")
	fmt.Printf("  Atrator: Lorenz(σ=%.1f, ρ=%.1f, β=%.1f)\n", bestSigma, bestRho, bestBeta)
	fmt.Printf("  Offset: %d\n", bestOffset)
	fmt.Printf("  Distância Hamming: %d/%d\n", bestDist, len(target))
	fmt.Printf("  Precisão: %.1f%%\n", (1.0-float64(bestDist)/float64(len(target)))*100)
	fmt.Printf("  Configurações testadas: %d\n", tested)
	fmt.Printf("  Tempo: %v\n", dur)

	orbit := lorenzOrbit(bestSigma, bestRho, bestBeta, 500)
	if bestOffset+len(target) <= len(orbit) {
		fmt.Printf("  Reconstruído: %v\n", orbit[bestOffset:bestOffset+len(target)])
	}
	fmt.Printf("  Original:     %v\n", target)

	if bestDist == 0 {
		fmt.Println("\n✔ MATCH PERFEITO via Atrator de Lorenz!")
		fmt.Printf("  Codebook: {σ,ρ,β,offset} = 32 bytes vs %d bytes\n", len(target))
	} else {
		fmt.Println("\n⚠ Match parcial — atratores caóticos cobrem subconjunto do espaço de bytes")
	}
}

func lorenzOrbit(sigma, rho, beta float64, steps int) []byte {
	x, y, z := 1.0, 1.0, 1.0
	dt := 0.01
	orbit := make([]byte, 0, steps)

	for i := 0; i < steps; i++ {
		dx := sigma * (y - x)
		dy := x*(rho-z) - y
		dz := x*y - beta*z
		x += dx * dt
		y += dy * dt
		z += dz * dt

		// Mapear x para byte
		val := math.Mod(math.Abs(x)*100, 256)
		orbit = append(orbit, byte(val))
	}
	return orbit
}

func hammingDist(a, b []byte) int {
	dist := 0
	for i := range a {
		if a[i] != b[i] {
			dist++
		}
	}
	return dist
}
