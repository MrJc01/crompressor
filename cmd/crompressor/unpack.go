package main

import (
	"fmt"

	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/spf13/cobra"
)

func unpackCmd() *cobra.Command {
	var input, output, codebookPath string
	var fuzziness float64
	var encryptionKey string
	var strict bool

	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "Decompress a .crom file",
		Long:  `Reads the .crom file, looks up patterns in the codebook, and reconstructs the original file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || output == "" || codebookPath == "" {
				return fmt.Errorf("flags --input, --output and --codebook are required")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║            CROMPRESSOR UNPACK             ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			fmt.Printf("║  Output:   %-30s ║\n", output)
			fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			if fuzziness > 0 {
				fmt.Printf("║  Fuzziness: %-29.2f ║\n", fuzziness)
			}
			if encryptionKey != "" {
				fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			opts := cromlib.DefaultUnpackOptions()
			opts.Fuzziness = fuzziness
			opts.Strict = strict
			if encryptionKey != "" {
				opts.EncryptionKey = encryptionKey
			}
			if err := cromlib.Unpack(input, output, codebookPath, opts); err != nil {
				return fmt.Errorf("unpack error: %v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input .crom file path")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output restored file path")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Codebook path (.cromdb)")
	cmd.Flags().Float64Var(&fuzziness, "fuzziness", 0.0, "Reconstruction variance (0 = lossless)")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Passphrase for AES-256-GCM decryption")
	cmd.Flags().BoolVar(&strict, "strict", false, "Abort unpacking on any corrupted block")

	return cmd
}
