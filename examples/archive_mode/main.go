// Package main demonstrates Archive mode compression (lossless).
//
// Archive mode stores the XOR delta for bit-exact reconstruction.
// SHA-256 of the restored file equals SHA-256 of the original.
package main

import (
	"crypto/sha256"
	"fmt"
	"io"
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
	restored := output + ".restored"

	// Pack in Archive mode (lossless)
	opts := cromlib.DefaultPackOptions()
	opts.Mode = "archive"

	metrics, err := cromlib.Pack(input, output, codebook, opts)
	if err != nil {
		log.Fatalf("Pack failed: %v", err)
	}

	fmt.Printf("Archive mode compression complete\n")
	fmt.Printf("  Original:   %d bytes\n", metrics.OriginalSize)
	fmt.Printf("  Compressed: %d bytes\n", metrics.PackedSize)
	fmt.Printf("  Ratio:      %.1fx\n", float64(metrics.OriginalSize)/float64(metrics.PackedSize))

	// Unpack to verify lossless round-trip
	unpackOpts := cromlib.DefaultUnpackOptions()
	err = cromlib.Unpack(output, restored, codebook, unpackOpts)
	if err != nil {
		log.Fatalf("Unpack failed: %v", err)
	}

	// Verify SHA-256 match
	hashOrig := sha256File(input)
	hashRestored := sha256File(restored)

	if hashOrig == hashRestored {
		fmt.Printf("  SHA-256:    MATCH ✓ (lossless verified)\n")
	} else {
		fmt.Printf("  SHA-256:    MISMATCH ✗\n")
		fmt.Printf("    Original: %s\n", hashOrig)
		fmt.Printf("    Restored: %s\n", hashRestored)
	}

	os.Remove(restored)
}

func sha256File(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Cannot open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}
