// Package main demonstrates training a custom codebook.
//
// A codebook is a dictionary of patterns extracted from a data corpus.
// Once trained, it can be used with both Edge and Archive modes.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MrJc01/crompressor/internal/trainer"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <input_dir> <output.cromdb>")
		fmt.Println("")
		fmt.Println("  input_dir    Directory containing training data files")
		fmt.Println("  output       Path for the generated codebook (.cromdb)")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputPath := os.Args[2]

	opts := trainer.DefaultTrainOptions()
	opts.InputDir = inputDir
	opts.OutputPath = outputPath
	opts.MaxCodewords = 4096

	fmt.Printf("Training codebook from %s\n", inputDir)
	fmt.Printf("  Max codewords: %d\n", opts.MaxCodewords)

	result, err := trainer.Train(opts)
	if err != nil {
		log.Fatalf("Training failed: %v", err)
	}

	fmt.Printf("\nTraining complete\n")
	fmt.Printf("  Unique patterns: %d\n", result.UniquePatterns)
	fmt.Printf("  Selected elite:  %d\n", result.SelectedElite)
	fmt.Printf("  Total bytes:     %d\n", result.TotalBytes)
	fmt.Printf("  Output:          %s\n", outputPath)
}
