package main

import (
	"fmt"
	"math"
	"time"
)

// Pesquisa 104: Regressão por Mínimos Quadrados (Vandermonde) para chunks de dados.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  104 - TAYLOR SERIES REGRESSION ENGINE (V2)      ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	target := []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160}
	fmt.Printf("[ALVO] %v (%d bytes)\n\n", target, len(target))

	start := time.Now()

	// Testar ordens crescentes até achar match perfeito
	for order := 1; order <= 6; order++ {
		coeffs := leastSquaresFit(target, order)
		reconstructed := polyGenerate(coeffs, len(target))
		matches := countMatches(target, reconstructed)
		mse := calcMSE(coeffs, target)

		fmt.Printf("  Ordem %d: %d/%d bytes match (MSE=%.2f) coeffs=%v\n",
			order, matches, len(target), mse, formatCoeffs(coeffs))

		if matches == len(target) {
			dur := time.Since(start)
			coeffBytes := len(coeffs) * 8 // float64 = 8 bytes
			fmt.Printf("\n  ✔ MATCH PERFEITO na ordem %d!\n", order)
			fmt.Printf("  Reconstruído: %v\n", reconstructed)
			fmt.Printf("  Coeficientes: %d × 8 = %d bytes vs Original: %d bytes\n",
				len(coeffs), coeffBytes, len(target))
			if coeffBytes < len(target) {
				fmt.Printf("  ✔ COMPRESSÃO REAL: %.1f%% redução\n", (1-float64(coeffBytes)/float64(len(target)))*100)
			} else {
				fmt.Printf("  ⚠ Sem compressão (coeficientes ≥ dados originais)\n")
			}
			fmt.Printf("  Tempo: %v\n", dur)
			return
		}
	}

	dur := time.Since(start)
	fmt.Printf("\n  ⚠ Nenhuma ordem até 6 produziu match perfeito\n")
	fmt.Printf("  Conclusão: Dados não são polinomiais — fallback para literal\n")
	fmt.Printf("  Tempo: %v\n", dur)
}

// Mínimos quadrados via equações normais: (VᵀV)c = Vᵀy
func leastSquaresFit(data []byte, order int) []float64 {
	n := len(data)
	m := order + 1

	// Construir Vandermonde matrix V e vetor y
	y := make([]float64, n)
	for i, b := range data {
		y[i] = float64(b)
	}

	// VᵀV (m×m)
	vtv := make([][]float64, m)
	for i := range vtv {
		vtv[i] = make([]float64, m)
	}
	// Vᵀy (m×1)
	vty := make([]float64, m)

	for i := 0; i < n; i++ {
		x := float64(i)
		for j := 0; j < m; j++ {
			xj := math.Pow(x, float64(j))
			vty[j] += xj * y[i]
			for k := 0; k < m; k++ {
				vtv[j][k] += xj * math.Pow(x, float64(k))
			}
		}
	}

	// Resolver via Gauss elimination
	return gaussSolve(vtv, vty, m)
}

func gaussSolve(a [][]float64, b []float64, n int) []float64 {
	// Augmented matrix
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i], a[i])
		aug[i][n] = b[i]
	}

	for col := 0; col < n; col++ {
		// Pivoting
		maxRow := col
		for row := col + 1; row < n; row++ {
			if math.Abs(aug[row][col]) > math.Abs(aug[maxRow][col]) {
				maxRow = row
			}
		}
		aug[col], aug[maxRow] = aug[maxRow], aug[col]

		if math.Abs(aug[col][col]) < 1e-12 {
			continue
		}

		// Eliminate
		for row := col + 1; row < n; row++ {
			factor := aug[row][col] / aug[col][col]
			for j := col; j <= n; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = aug[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= aug[i][j] * x[j]
		}
		if math.Abs(aug[i][i]) > 1e-12 {
			x[i] /= aug[i][i]
		}
	}
	return x
}

func polyGenerate(coeffs []float64, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		val := 0.0
		x := float64(i)
		for k, c := range coeffs {
			val += c * math.Pow(x, float64(k))
		}
		clamped := math.Round(val)
		if clamped < 0 { clamped = 0 }
		if clamped > 255 { clamped = 255 }
		out[i] = byte(clamped)
	}
	return out
}

func countMatches(a, b []byte) int {
	c := 0
	for i := range a {
		if a[i] == b[i] { c++ }
	}
	return c
}

func calcMSE(coeffs []float64, target []byte) float64 {
	total := 0.0
	for i, t := range target {
		val := 0.0
		x := float64(i)
		for k, c := range coeffs {
			val += c * math.Pow(x, float64(k))
		}
		diff := val - float64(t)
		total += diff * diff
	}
	return total / float64(len(target))
}

func formatCoeffs(c []float64) string {
	s := "["
	for i, v := range c {
		if i > 0 { s += ", " }
		s += fmt.Sprintf("%.4f", v)
	}
	return s + "]"
}
