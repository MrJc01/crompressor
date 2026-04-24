package main

import (
	"fmt"

	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/spf13/cobra"
)

func grepCmd() *cobra.Command {
	var inputPath, codebookPath string

	cmd := &cobra.Command{
		Use:   "grep <query>",
		Short: "Search inside a .crom file without decompressing",
		Long:  `Translates a query to its BPE numeric ID and scans the occurrence matrix, skipping local decompression.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputPath == "" || codebookPath == "" {
				return fmt.Errorf("flags --input and --codebook are required")
			}
			target := args[0]

			isCloud := len(inputPath) > 7 && (inputPath[:7] == "http://" || inputPath[:8] == "https://")

			fmt.Println("╔═══════════════════════════════════════════╗")
			if isCloud {
				fmt.Println("║       CROMPRESSOR GREP (Remote)           ║")
			} else {
				fmt.Println("║       CROMPRESSOR GREP (Local)            ║")
			}
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Target:   %-30s ║\n", target)
			fmt.Printf("║  Input:    %-30s ║\n", inputPath)
			fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			fmt.Println("╚═══════════════════════════════════════════╝")

			return cromlib.Grep(target, inputPath, codebookPath)
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input .crom file path (supports HTTP/HTTPS URLs)")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Codebook path (.cromdb)")

	return cmd
}
