package main

import (
	"fmt"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func trainCmd() *cobra.Command {
	var inputDir, outputPath, updatePath, basePath string
	var maxCodewords, concurrency, chunkSize int
	var augmentTrain, useBPE bool

	cmd := &cobra.Command{
		Use:   "train",
		Short: "Train a codebook from data in a directory",
		Long:  `Scans files in batch, extracts frequent patterns, and selects an elite set with LSH diversity to form a universal .cromdb codebook.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputDir == "" || outputPath == "" {
				return fmt.Errorf("flags --input and --output are required")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║            CROMPRESSOR TRAIN              ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input Dir: %-29s ║\n", inputDir)
			fmt.Printf("║  Output:    %-29s ║\n", outputPath)
			fmt.Printf("║  Target:    %-29d ║\n", maxCodewords)
			if updatePath != "" {
				fmt.Printf("║  Mode:      %-29s ║\n", "Incremental Update")
			} else if basePath != "" {
				fmt.Printf("║  Mode:      %-29s ║\n", "Transfer Learning")
			} else {
				fmt.Printf("║  Mode:      %-29s ║\n", "Standard")
			}
			if useBPE {
				fmt.Printf("║  Engine:    %-29s ║\n", "BPE (Neural Tokenizer)")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			bar := progressbar.DefaultBytes(-1, "Extracting Patterns")

			opts := trainer.DefaultTrainOptions()
			opts.InputDir = inputDir
			opts.OutputPath = outputPath
			if maxCodewords > 0 {
				opts.MaxCodewords = maxCodewords
			}
			if concurrency > 0 {
				opts.Concurrency = concurrency
			}
			if chunkSize > 0 {
				opts.ChunkSize = chunkSize
			}
			opts.UpdatePath = updatePath
			opts.BasePath = basePath
			opts.DataAugmentation = augmentTrain
			opts.UseBPE = useBPE
			opts.OnProgress = func(n int) {
				bar.Add(n)
			}

			res, err := trainer.Train(opts)
			if err != nil {
				return fmt.Errorf("training error: %v", err)
			}

			fmt.Printf("\n✔ Training completed in %v\n", res.Duration)
			fmt.Printf("  Files Parsed:    %d\n", res.TotalFiles)
			fmt.Printf("  Total Bytes:     %d\n", res.TotalBytes)
			fmt.Printf("  Unique Patterns: %d\n", res.UniquePatterns)
			fmt.Printf("  Elite Selected:  %d\n", res.SelectedElite)
			if res.MergedPatterns > 0 {
				fmt.Printf("  Merged Patterns: %d\n", res.MergedPatterns)
				fmt.Printf("  Replaced Slots:  %d\n", res.ReplacedSlots)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputDir, "input", "i", "", "Directory with training data")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output .cromdb path")
	cmd.Flags().IntVarP(&maxCodewords, "size", "s", 8192, "Maximum codebook entries")
	cmd.Flags().IntVarP(&chunkSize, "chunk-size", "k", 0, "Base chunk size (0 = auto)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Number of parallel goroutines")
	cmd.Flags().StringVar(&updatePath, "update", "", "Existing .cromdb for incremental update")
	cmd.Flags().StringVar(&basePath, "base", "", "Base .cromdb for transfer learning")
	cmd.Flags().BoolVar(&augmentTrain, "augment", false, "Apply stochastic bit-shift augmentation")
	cmd.Flags().BoolVar(&useBPE, "use-bpe", false, "Use BPE tokenizer engine instead of raw frequency")

	return cmd
}
