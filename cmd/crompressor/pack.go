package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MrJc01/crompressor/internal/autobrain"
	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func packCmd() *cobra.Command {
	var input, output, codebookPath string
	var concurrency, chunkSize int
	var useCDC bool
	var encryptionKey string
	var autoBrain bool
	var multiPass bool
	var streamMode bool
	var brainDir string
	var packMode string

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Compress a file using a codebook",
		Long:  `Splits the file into chunks, matches patterns in the codebook, and generates a compact .crom file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || output == "" {
				return fmt.Errorf("flags --input and --output are required")
			}

			var det *autobrain.DetectionResult
			var useAutoTrain bool
			if autoBrain {
				if codebookPath != "" {
					return fmt.Errorf("--auto-brain and --codebook cannot be used together")
				}
				router, err := autobrain.NewBrainRouter(brainDir)
				if err != nil {
					return fmt.Errorf("failed to initialize auto-brain: %w", err)
				}
				cb, result, err := router.SelectBrain(input)
				if err != nil {
					fmt.Printf("No suitable brain found for '%s'. Activating Auto-Training...\n", result.Category)
					useAutoTrain = true
					det = result
				} else {
					codebookPath = cb
					det = result
				}
			} else if codebookPath == "" {
				fmt.Println("Zero-Config mode: No codebook provided. Activating Auto-Training...")
				useAutoTrain = true
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║              CROMPRESSOR PACK             ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			fmt.Printf("║  Output:   %-30s ║\n", output)
			fmt.Printf("║  Mode:     %-30s ║\n", packMode)
			if autoBrain && det != nil {
				fmt.Printf("║  AutoBrain: %-29s ║\n", det.Category)
				fmt.Printf("║   ↳ Codebook: %-27s ║\n", filepath.Base(codebookPath))
			} else if codebookPath != "" {
				fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			}
			if encryptionKey != "" {
				fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			info, err := os.Stat(input)
			if err != nil {
				return err
			}

			bar := progressbar.DefaultBytes(
				info.Size(),
				"Compressing",
			)

			opts := cromlib.DefaultPackOptions()
			if concurrency > 0 {
				opts.Concurrency = concurrency
			}
			if chunkSize > 0 {
				opts.ChunkSize = chunkSize
			}
			opts.UseCDC = useCDC
			opts.MultiPass = multiPass
			opts.Mode = packMode
			if encryptionKey != "" {
				opts.EncryptionKey = encryptionKey
			}
			opts.OnProgress = func(n int) {
				bar.Add(n)
			}

			if streamMode {
				var reader io.Reader
				if input == "-" {
					reader = os.Stdin
				} else {
					f, err := os.Open(input)
					if err != nil {
						return err
					}
					defer f.Close()
					reader = f
				}

				metrics, err := cromlib.PackStream(reader, output, codebookPath, opts)
				if err != nil {
					return fmt.Errorf("stream pack error: %v", err)
				}

				fmt.Printf("\n✔ Stream Pack completed\n")
				fmt.Printf("  Original Size: %d bytes\n", metrics.OriginalSize)
				fmt.Printf("  Packed Size:   %d bytes (%.2f%% ratio)\n",
					metrics.PackedSize,
					float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100)
				fmt.Printf("  Hit Rate:      %.2f%%\n", metrics.HitRate)
				return nil
			}

			var metrics *cromlib.Metrics
			if useAutoTrain {
				metrics, err = cromlib.AutoPack(input, output, opts)
			} else {
				metrics, err = cromlib.Pack(input, output, codebookPath, opts)
			}
			if err != nil {
				return fmt.Errorf("pack error: %v", err)
			}

			fmt.Printf("\n✔ Pack completed in %v\n", metrics.Duration)
			fmt.Printf("  Original Size: %d bytes\n", metrics.OriginalSize)
			fmt.Printf("  Packed Size:   %d bytes (%.2f%% ratio)\n",
				metrics.PackedSize,
				float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100)
			fmt.Printf("  Hit Rate:      %.2f%%\n", metrics.HitRate)
			fmt.Printf("  Data Entropy:  %.2f bits/byte\n", metrics.Entropy)

			var litPct float64
			if metrics.TotalChunks > 0 {
				litPct = float64(metrics.LiteralChunks) / float64(metrics.TotalChunks) * 100
			}
			fmt.Printf("  Literal Chunks: %d/%d (%.2f%%)\n", metrics.LiteralChunks, metrics.TotalChunks, litPct)
			fmt.Printf("  Avg Similarity: %.2f%%\n", metrics.AvgSimilarity*100)

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input file path")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output .crom file path")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Codebook path (.cromdb)")
	cmd.Flags().BoolVar(&autoBrain, "auto-brain", false, "Automatically select codebook based on file content")
	cmd.Flags().StringVar(&brainDir, "brain-dir", filepath.Join(os.Getenv("HOME"), ".crompressor", "brains"), "Directory containing codebooks for Auto-Brain")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Number of parallel goroutines")
	cmd.Flags().IntVarP(&chunkSize, "chunk-size", "k", 0, "Base chunk size (0 = auto)")
	cmd.Flags().BoolVar(&useCDC, "cdc", false, "Enable Content-Defined Chunking")
	cmd.Flags().BoolVar(&multiPass, "multi-pass", false, "Enable two-pass LSH Top-K compression")
	cmd.Flags().BoolVar(&streamMode, "stream", false, "Stream mode — compress pipes/stdin without Seek")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Passphrase for AES-256-GCM encryption")
	cmd.Flags().StringVar(&packMode, "mode", "archive", "Operation mode: 'archive' (lossless) or 'edge' (lossy)")

	return cmd
}
