//go:build ignore

// gen_mini_codebook generates a test .cromdb file with deterministic random patterns.
//
// Usage: go run scripts/gen_mini_codebook.go [output_path]
// Default output: testdata/mini.cromdb
//
// Generates a ~1MB codebook with 8192 codewords of 128 bytes each.
// Uses a fixed seed for reproducibility.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	headerSize    = 512
	magicString   = "CROMDB"
	version       = 1
	codewordSize  = 128
	codewordCount = 8192
	seed          = 42 // Fixed seed for deterministic output
)

func main() {
	outputPath := "testdata/mini.cromdb"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Generate codeword data with fixed seed
	rng := rand.New(rand.NewSource(seed))
	dataSize := codewordSize * codewordCount
	codewordData := make([]byte, dataSize)
	rng.Read(codewordData)

	// Calculate SHA-256 of codeword data (Build Hash)
	buildHash := sha256.Sum256(codewordData)

	// Build header
	header := make([]byte, headerSize)
	copy(header[0:6], magicString)                                    // Magic
	binary.LittleEndian.PutUint16(header[6:8], version)               // Version
	binary.LittleEndian.PutUint16(header[8:10], codewordSize)         // Codeword Size
	binary.LittleEndian.PutUint64(header[10:18], codewordCount)       // Codeword Count
	binary.LittleEndian.PutUint64(header[18:26], headerSize)          // Data Offset
	copy(header[26:58], buildHash[:])                                 // Build Hash

	// Write file
	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if _, err := f.Write(header); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing header: %v\n", err)
		os.Exit(1)
	}
	if _, err := f.Write(codewordData); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing codewords: %v\n", err)
		os.Exit(1)
	}

	totalSize := headerSize + dataSize
	fmt.Printf("╔═══════════════════════════════════════════════╗\n")
	fmt.Printf("║         CROM MINI CODEBOOK GENERATOR          ║\n")
	fmt.Printf("╠═══════════════════════════════════════════════╣\n")
	fmt.Printf("║  Output:       %-30s ║\n", outputPath)
	fmt.Printf("║  Codewords:    %-30d ║\n", codewordCount)
	fmt.Printf("║  Codeword Size:%-30d ║\n", codewordSize)
	fmt.Printf("║  Total Size:   %-30s ║\n", formatSize(totalSize))
	fmt.Printf("║  Build Hash:   %x...  ║\n", buildHash[:8])
	fmt.Printf("║  Seed:         %-30d ║\n", seed)
	fmt.Printf("╚═══════════════════════════════════════════════╝\n")
}

func formatSize(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
