package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MrJc01/crompressor/internal/entropy"
	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/spf13/cobra"
)

func benchmarkCmd() *cobra.Command {
	var input, codebookPath, outputJson string
	var runs int

	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run a full compression and decompression benchmark",
		Long:  `Executes N cycles of Pack + Unpack + Verify with the same data, emitting metrics as JSON.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || codebookPath == "" {
				return fmt.Errorf("flags --input and --codebook are required")
			}

			inputBytes, err := os.ReadFile(input)
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			inputEntropy, _, _ := entropy.Analyze(bytes.NewReader(inputBytes), len(inputBytes))

			type Output struct {
				InputFile     string  `json:"input_file"`
				InputSize     uint64  `json:"input_size"`
				InputEntropy  float64 `json:"input_entropy"`
				PackedSize    uint64  `json:"packed_size"`
				Ratio         float64 `json:"ratio"`
				HitRate       float64 `json:"hit_rate"`
				LiteralChunks int     `json:"literal_chunks"`
				TotalChunks   int     `json:"total_chunks"`
				AvgSimilarity float64 `json:"avg_similarity"`
				PackMs        int64   `json:"pack_ms"`
				UnpackMs      int64   `json:"unpack_ms"`
				Verify        string  `json:"verify"`
				Runs          int     `json:"runs"`
				Engine        string  `json:"engine"`
			}

			out := Output{
				InputFile:    input,
				InputSize:    uint64(len(inputBytes)),
				InputEntropy: inputEntropy,
				Runs:         runs,
				Engine:       "V4",
			}

			var totalPack, totalUnpack time.Duration
			var lastMetrics *cromlib.Metrics

			for i := 0; i < runs; i++ {
				cromFile, _ := os.CreateTemp("", "bench_crom_*")
				restoredFile, _ := os.CreateTemp("", "bench_restored_*")
				cromPath := cromFile.Name()
				restoredPath := restoredFile.Name()
				cromFile.Close()
				restoredFile.Close()

				metrics, err := cromlib.Pack(input, cromPath, codebookPath, cromlib.DefaultPackOptions())
				if err != nil {
					return fmt.Errorf("pack failed run %d: %v", i+1, err)
				}
				totalPack += metrics.Duration
				lastMetrics = metrics

				startUnpack := time.Now()
				err = cromlib.Unpack(cromPath, restoredPath, codebookPath, cromlib.DefaultUnpackOptions())
				if err != nil {
					return fmt.Errorf("unpack failed run %d: %v", i+1, err)
				}
				totalUnpack += time.Since(startUnpack)

				restoredBytes, _ := os.ReadFile(restoredPath)
				if bytes.Equal(inputBytes, restoredBytes) {
					out.Verify = "PASS"
				} else {
					out.Verify = "FAIL"
				}

				os.Remove(cromPath)
				os.Remove(restoredPath)
			}

			if lastMetrics != nil {
				out.PackedSize = lastMetrics.PackedSize
				out.Ratio = float64(out.PackedSize) / float64(out.InputSize) * 100
				out.HitRate = lastMetrics.HitRate
				out.LiteralChunks = lastMetrics.LiteralChunks
				out.TotalChunks = lastMetrics.TotalChunks
				out.AvgSimilarity = lastMetrics.AvgSimilarity
			}
			out.PackMs = totalPack.Milliseconds() / int64(runs)
			out.UnpackMs = totalUnpack.Milliseconds() / int64(runs)

			if out.Verify != "PASS" {
				return fmt.Errorf("integrity verification failed")
			}

			jsonData, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}

			if outputJson != "" {
				if err := os.WriteFile(outputJson, jsonData, 0644); err != nil {
					return err
				}
				fmt.Printf("Benchmark saved to %s\n", outputJson)
			} else {
				fmt.Println(string(jsonData))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input file path")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Codebook path (.cromdb)")
	cmd.Flags().StringVarP(&outputJson, "output-json", "o", "", "Output JSON path (optional)")
	cmd.Flags().IntVarP(&runs, "runs", "r", 1, "Number of benchmark iterations")

	return cmd
}
