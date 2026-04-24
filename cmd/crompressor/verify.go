package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func verifyCmd() *cobra.Command {
	var original, restored string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify bit-exact integrity between two files",
		Long:  `Compares SHA-256 hashes of two files to confirm lossless fidelity.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if original == "" || restored == "" {
				return fmt.Errorf("flags --original and --restored are required")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║            CROMPRESSOR VERIFY             ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Original: %-30s ║\n", original)
			fmt.Printf("║  Restored: %-30s ║\n", restored)
			fmt.Println("╚═══════════════════════════════════════════╝")

			origBytes, err := os.ReadFile(original)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", original, err)
			}
			restBytes, err := os.ReadFile(restored)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", restored, err)
			}

			origHash := sha256.Sum256(origBytes)
			restHash := sha256.Sum256(restBytes)

			if origHash != restHash {
				return fmt.Errorf("FAIL: SHA-256 mismatch!\n  Original: %x\n  Restored: %x", origHash[:8], restHash[:8])
			}

			fmt.Println("✔ INTEGRITY CONFIRMED: SHA-256 match (bit-exact fidelity)")
			return nil
		},
	}

	cmd.Flags().StringVar(&original, "original", "", "Original file path")
	cmd.Flags().StringVar(&restored, "restored", "", "Restored file path")

	return cmd
}
