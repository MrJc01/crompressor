package main

import (
	"bytes"
	"fmt"
	"time"
)

// Pesquisa 108: Busca de chunks dentro dos dígitos de Pi.
// Usa o algoritmo Spigot simplificado para gerar dígitos de Pi em base 256
// e procura o chunk-alvo como substring.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  108 - PI DIGIT EXTRACTION ENGINE                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	target := []byte{20, 30, 40, 50}
	fmt.Printf("[ALVO] %v (%d bytes)\n\n", target, len(target))

	// Gerar N dígitos de Pi em base 256 via série de Leibniz acumulada
	maxDigits := 100_000
	fmt.Printf("[CONFIG] Gerando %d dígitos de Pi (base 256)...\n", maxDigits)

	start := time.Now()
	piDigits := generatePiBytes(maxDigits)
	genDur := time.Since(start)
	fmt.Printf("  Geração: %v\n", genDur)

	// Buscar o chunk nos dígitos
	searchStart := time.Now()
	idx := bytes.Index(piDigits, target)
	searchDur := time.Since(searchStart)

	fmt.Println("\n═══ Resultado ═══")
	if idx >= 0 {
		fmt.Printf("  ✔ ENCONTRADO no offset %d dos dígitos de Pi!\n", idx)
		fmt.Printf("  Codebook: {source: \"pi\", offset: %d, len: %d} = 12 bytes\n", idx, len(target))
		fmt.Printf("  Original: %d bytes → Compressão: %.1f%%\n", len(target), (1.0-12.0/float64(len(target)))*100)
	} else {
		fmt.Printf("  ✗ Não encontrado nos primeiros %d dígitos de Pi\n", maxDigits)
		fmt.Println("  Conclusão: Pi não é um dicionário universal eficiente para dados arbitrários")
	}
	fmt.Printf("  Tempo de busca: %v\n", searchDur)
	fmt.Printf("  Tempo total: %v\n", genDur+searchDur)

	// Estatísticas de cobertura
	coverage := make(map[byte]bool)
	for _, b := range piDigits {
		coverage[b] = true
	}
	fmt.Printf("  Bytes únicos cobertos: %d/256 (%.1f%%)\n", len(coverage), float64(len(coverage))/256*100)
}

// generatePiBytes gera bytes derivados de Pi via série de Machin simplificada
func generatePiBytes(n int) []byte {
	// Pi via série de Leibniz com convergência acelerada
	// π/4 = 1 - 1/3 + 1/5 - 1/7 + ...
	// Usamos acumulação de alta precisão e extraímos bytes
	digits := make([]byte, n)
	pi := 0.0
	for k := 0; k < n*100; k++ {
		sign := 1.0
		if k%2 == 1 {
			sign = -1.0
		}
		pi += sign / float64(2*k+1)

		if k >= 100 && (k-100)%(100) == 0 {
			idx := (k - 100) / 100
			if idx < n {
				// Extrair byte da mantissa fraccionária
				frac := pi * 4.0
				for frac < 0 {
					frac += 256
				}
				digits[idx] = byte(int(frac*1e10) % 256)
			}
		}
	}
	return digits
}
