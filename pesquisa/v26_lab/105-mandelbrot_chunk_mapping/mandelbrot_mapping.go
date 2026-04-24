package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"time"
)

// Pesquisa 105: Mapeamento de chunks nas iterações do Conjunto de Mandelbrot.
// Hipótese: substrings de órbitas Mandelbrot podem coincidir com chunks de dados.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  105 - MANDELBROT CHUNK MAPPING ENGINE            ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	target := []byte{145, 23, 99, 211, 45, 11, 201, 89}
	fmt.Printf("[ALVO] %v (%d bytes)\n\n", target, len(target))

	start := time.Now()
	gridRes := 500 // 500x500 pontos no plano complexo
	maxIter := 256

	bestC, bestOffset, bestDist := searchMandelbrot(target, gridRes, maxIter)
	dur := time.Since(start)

	fmt.Println("═══ Resultado ═══")
	fmt.Printf("  Melhor c: (%.6f, %.6fi)\n", real(bestC), imag(bestC))
	fmt.Printf("  Offset da Órbita: %d\n", bestOffset)
	fmt.Printf("  Distância Hamming: %d/%d bytes\n", bestDist, len(target))
	fmt.Printf("  Precisão: %.1f%%\n", (1.0-float64(bestDist)/float64(len(target)))*100)
	fmt.Printf("  Tempo: %v\n", dur)
	fmt.Printf("  Pontos testados: %d\n", gridRes*gridRes)

	// Reconstruir
	orbit := mandelbrotOrbit(bestC, maxIter)
	if bestOffset+len(target) <= len(orbit) {
		reconstructed := orbit[bestOffset : bestOffset+len(target)]
		fmt.Printf("  Reconstruído: %v\n", reconstructed)
		fmt.Printf("  Original:     %v\n", target)
	}

	if bestDist == 0 {
		fmt.Println("\n✔ MATCH PERFEITO: Chunk encontrado na órbita de Mandelbrot!")
		fmt.Printf("  Codebook: c=(%.6f,%.6fi) offset=%d → 24 bytes vs %d bytes\n",
			real(bestC), imag(bestC), bestOffset, len(target))
	} else {
		fmt.Println("\n⚠ Match parcial — Mandelbrot cobre dados correlacionados, não arbitrários")
	}
}

func searchMandelbrot(target []byte, gridRes, maxIter int) (complex128, int, int) {
	bestC := complex(0, 0)
	bestOffset := 0
	bestDist := len(target) + 1

	for xi := 0; xi < gridRes; xi++ {
		for yi := 0; yi < gridRes; yi++ {
			cr := -2.0 + 3.0*float64(xi)/float64(gridRes)
			ci := -1.5 + 3.0*float64(yi)/float64(gridRes)
			c := complex(cr, ci)

			orbit := mandelbrotOrbit(c, maxIter)
			if len(orbit) < len(target) {
				continue
			}

			for off := 0; off <= len(orbit)-len(target); off++ {
				dist := hammingDist(target, orbit[off:off+len(target)])
				if dist < bestDist {
					bestDist = dist
					bestC = c
					bestOffset = off
					if dist == 0 {
						return bestC, bestOffset, 0
					}
				}
			}
		}
	}
	return bestC, bestOffset, bestDist
}

func mandelbrotOrbit(c complex128, maxIter int) []byte {
	z := complex(0, 0)
	orbit := make([]byte, 0, maxIter)
	for i := 0; i < maxIter; i++ {
		z = z*z + c
		if cmplx.Abs(z) > 2.0 {
			break
		}
		// Mapear parte real para byte
		val := math.Mod(math.Abs(real(z))*1000, 256)
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
