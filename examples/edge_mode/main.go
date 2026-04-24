// Package main demonstrates Edge mode compression (lossy).
//
// Edge mode discards the XOR delta, achieving high compression
// at the cost of lossy reconstruction.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MrJc01/crompressor/pkg/cromlib"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <input> <codebook.cromdb> <output.crom>")
		os.Exit(1)
	}

	input := os.Args[1]
	codebook := os.Args[2]
	output := os.Args[3]

	opts := cromlib.DefaultPackOptions()
	opts.Mode = "edge" // Lossy: discard XOR delta

	metrics, err := cromlib.Pack(input, output, codebook, opts)
	if err != nil {
		log.Fatalf("Pack failed: %v", err)
	}

	fmt.Printf("Edge mode compression complete\n")
	fmt.Printf("  Original:   %d bytes\n", metrics.OriginalSize)
	fmt.Printf("  Compressed: %d bytes\n", metrics.PackedSize)
	fmt.Printf("  Ratio:      %.1fx\n", float64(metrics.OriginalSize)/float64(metrics.PackedSize))
	fmt.Printf("  Hit Rate:   %.2f%%\n", metrics.HitRate)
	fmt.Printf("  Entropy:    %.2f bits/byte\n", metrics.Entropy)
}
