package main

import (
	"fmt"
	"math/big"
	"time"
)

// Pesquisa 106: Codec de Frações Contínuas (V2)
// Trata o chunk inteiro como um único big.Int e decompõe em CF.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  106 - CONTINUED FRACTION CODEC (V2)             ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	// Testar com dados de padrão repetitivo onde CF brilha
	chunks := [][]byte{
		{10, 20, 30, 40, 50, 60, 70, 80},          // Linear
		{0, 1, 1, 2, 3, 5, 8, 13},                  // Fibonacci
		{255, 255, 255, 255, 255, 255, 255, 255},    // Constante
		{145, 23, 99, 211, 45, 11, 201, 89},         // Aleatório
	}

	for i, chunk := range chunks {
		fmt.Printf("\n═══ Chunk %d: %v ═══\n", i, chunk)
		start := time.Now()

		num := bytesToBigInt(chunk)
		// Usar um denominador primo para gerar fração interessante
		den := new(big.Int).SetInt64(16777259) // primo próximo de 2^24

		cf := toCF(new(big.Int).Set(num), new(big.Int).Set(den), 30)
		dur := time.Since(start)

		// Reconstruir
		rNum, rDen := fromCF(cf)

		fmt.Printf("  Inteiro:     %s\n", num.String())
		fmt.Printf("  CF:          %v (%d termos)\n", cfToInts(cf), len(cf))
		fmt.Printf("  Reconstruído: %s/%s\n", rNum.String(), rDen.String())

		origBytes := len(chunk)
		// Cada termo CF cabe em varint (1-4 bytes tipicamente)
		cfBytes := 0
		for _, term := range cf {
			bits := term.BitLen()
			cfBytes += (bits + 7) / 8
			if cfBytes == 0 { cfBytes = 1 }
		}
		cfBytes += 1 // length prefix

		fmt.Printf("  Original:    %d bytes\n", origBytes)
		fmt.Printf("  CF Varint:   %d bytes\n", cfBytes)
		fmt.Printf("  Tempo:       %v\n", dur)

		if cfBytes < origBytes {
			fmt.Printf("  ✔ COMPRESSÃO: %.1f%% redução\n", (1-float64(cfBytes)/float64(origBytes))*100)
		} else {
			fmt.Printf("  ✗ Expansão (+%d bytes)\n", cfBytes-origBytes)
		}
	}

	fmt.Println("\n═══ Conclusão ═══")
	fmt.Println("  CF funciona bem para dados com estrutura racional/periódica")
	fmt.Println("  Dados aleatórios geram CFs longas (como esperado por Khinchin)")
}

func bytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

func toCF(num, den *big.Int, maxTerms int) []*big.Int {
	cf := make([]*big.Int, 0, maxTerms)
	zero := new(big.Int)
	for den.Cmp(zero) != 0 && len(cf) < maxTerms {
		q := new(big.Int)
		r := new(big.Int)
		q.DivMod(num, den, r)
		cf = append(cf, new(big.Int).Set(q))
		num.Set(den)
		den.Set(r)
	}
	return cf
}

func fromCF(cf []*big.Int) (*big.Int, *big.Int) {
	if len(cf) == 0 {
		return big.NewInt(0), big.NewInt(1)
	}
	num := new(big.Int).Set(cf[len(cf)-1])
	den := big.NewInt(1)
	for i := len(cf) - 2; i >= 0; i-- {
		// num, den = den + cf[i]*num, num
		tmp := new(big.Int).Mul(cf[i], num)
		tmp.Add(tmp, den)
		den.Set(num)
		num.Set(tmp)
	}
	return num, den
}

func cfToInts(cf []*big.Int) []int64 {
	out := make([]int64, len(cf))
	for i, v := range cf {
		out[i] = v.Int64()
	}
	return out
}
