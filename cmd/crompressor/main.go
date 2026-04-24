// Crompressor — Dictionary-based compression engine
//
// Usage:
//
//	crompressor pack   --input FILE --output FILE --codebook FILE [--mode edge|archive]
//	crompressor unpack --input FILE --output FILE --codebook FILE
//	crompressor verify --original FILE --restored FILE
//	crompressor info   --input FILE.crom
//	crompressor train  --input DIR --output FILE.cromdb
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "crompressor",
		Short: "Crompressor — Dictionary-based compression engine",
		Long: `Crompressor is a compression engine built around three composable primitives:

  CDC  — Content-Defined Chunking via Rabin Fingerprint
  VQ   — Vector Quantization against a pre-trained codebook
  XOR  — Delta encoding for lossless reconstruction

Two modes of operation:
  edge    — Lossy. Discards XOR delta. Fast, compact.
  archive — Lossless. Stores XOR delta. SHA-256 verified.`,
		Version: version,
	}

	rootCmd.AddCommand(packCmd())
	rootCmd.AddCommand(unpackCmd())
	rootCmd.AddCommand(trainCmd())
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(benchmarkCmd())
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(grepCmd())

	// OS-specific commands injected via build tags
	addSystemCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
