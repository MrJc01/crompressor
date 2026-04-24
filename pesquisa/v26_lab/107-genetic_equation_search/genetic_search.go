package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"
)

// Pesquisa 107: Algoritmo Genético para evoluir equações matemáticas
// que reproduzem um chunk-alvo. Genes = operações + constantes.

type Gene struct {
	Op    byte   // 0=add, 1=mul, 2=xor, 3=mod
	Const byte
}

type Individual struct {
	Genes   []Gene
	Fitness int // Hamming distance (menor = melhor)
}

const (
	popSize    = 500
	geneCount  = 8
	generations = 2000
	mutRate    = 0.15
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║  107 - GENETIC EQUATION SEARCH ENGINE             ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	target := []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160}
	fmt.Printf("[ALVO] %v (%d bytes)\n\n", target, len(target))

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()

	// Inicializar população
	pop := make([]Individual, popSize)
	for i := range pop {
		pop[i] = randomIndividual(rng)
		pop[i].Fitness = evaluate(pop[i], target)
	}

	bestEver := pop[0]
	for g := 0; g < generations; g++ {
		// Seleção por torneio
		newPop := make([]Individual, popSize)
		for i := range newPop {
			a := pop[rng.Intn(popSize)]
			b := pop[rng.Intn(popSize)]
			if a.Fitness <= b.Fitness {
				newPop[i] = crossover(a, b, rng)
			} else {
				newPop[i] = crossover(b, a, rng)
			}
			mutate(&newPop[i], rng)
			newPop[i].Fitness = evaluate(newPop[i], target)

			if newPop[i].Fitness < bestEver.Fitness {
				bestEver = newPop[i]
			}
		}
		pop = newPop

		if bestEver.Fitness == 0 {
			fmt.Printf("  ✔ Solução perfeita na geração %d!\n", g)
			break
		}
		if g%500 == 0 {
			fmt.Printf("  Gen %d: melhor fitness=%d\n", g, bestEver.Fitness)
		}
	}

	dur := time.Since(start)
	output := execute(bestEver, len(target))

	fmt.Println("\n═══ Resultado ═══")
	fmt.Printf("  Melhor Fitness (Hamming): %d/%d\n", bestEver.Fitness, len(target))
	fmt.Printf("  Precisão: %.1f%%\n", (1.0-float64(bestEver.Fitness)/float64(len(target)))*100)
	fmt.Printf("  Genes: %v\n", bestEver.Genes)
	fmt.Printf("  Output:   %v\n", output)
	fmt.Printf("  Original: %v\n", target)
	fmt.Printf("  Tempo: %v\n", dur)

	geneBytes := len(bestEver.Genes) * 2 // Op + Const = 2 bytes cada
	fmt.Printf("\n  Original: %d bytes\n", len(target))
	fmt.Printf("  Genoma:   %d bytes (%d genes × 2)\n", geneBytes, len(bestEver.Genes))
	if bytes.Equal(output, target) {
		fmt.Println("  ✔ MATCH PERFEITO: Genoma reproduz o chunk integralmente!")
	}
}

func randomIndividual(rng *rand.Rand) Individual {
	genes := make([]Gene, geneCount)
	for i := range genes {
		genes[i] = Gene{Op: byte(rng.Intn(4)), Const: byte(rng.Intn(256))}
	}
	return Individual{Genes: genes}
}

func execute(ind Individual, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		val := byte(i)
		for _, g := range ind.Genes {
			switch g.Op {
			case 0:
				val = val + g.Const
			case 1:
				val = val * g.Const
			case 2:
				val = val ^ g.Const
			case 3:
				if g.Const > 0 {
					val = val % g.Const
				}
			}
		}
		out[i] = val
	}
	return out
}

func evaluate(ind Individual, target []byte) int {
	output := execute(ind, len(target))
	dist := 0
	for i := range target {
		if output[i] != target[i] {
			dist++
		}
	}
	return dist
}

func crossover(a, b Individual, rng *rand.Rand) Individual {
	child := Individual{Genes: make([]Gene, len(a.Genes))}
	point := rng.Intn(len(a.Genes))
	for i := range child.Genes {
		if i < point {
			child.Genes[i] = a.Genes[i]
		} else {
			child.Genes[i] = b.Genes[i]
		}
	}
	return child
}

func mutate(ind *Individual, rng *rand.Rand) {
	for i := range ind.Genes {
		if rng.Float64() < mutRate {
			ind.Genes[i].Op = byte(rng.Intn(4))
			ind.Genes[i].Const = byte(rng.Intn(256))
		}
	}
}
