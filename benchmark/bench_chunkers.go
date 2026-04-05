package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// ChunkerResult holds the result of a single chunker strategy test.
type ChunkerResult struct {
	Dataset    string
	Chunker    string
	Ratio      float64
	HitRate    float64
	Throughput float64
	PackedSize int64
	Duration   time.Duration
	Integrity  bool
}

// RunChunkerBenchmark compares Fixed vs CDC vs ACAC on structured data.
func RunChunkerBenchmark(workDir string) []ChunkerResult {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🔀 CHUNKER COMPARISON — Fixed vs CDC vs ACAC")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var results []ChunkerResult

	datasets := []struct {
		name string
		gen  func(string, int) error
		size int
	}{
		{"json_api", generateJSON, 5},
		{"server_logs", generateLogs, 5},
		{"go_source", generateGoSource, 5},
	}

	chunkers := []struct {
		name string
		opts func() cromlib.PackOptions
	}{
		{"Fixed-128B", func() cromlib.PackOptions {
			o := cromlib.DefaultPackOptions()
			o.Concurrency = 4
			return o
		}},
		{"FastCDC", func() cromlib.PackOptions {
			o := cromlib.DefaultPackOptions()
			o.UseCDC = true
			o.Concurrency = 4
			return o
		}},
		{"ACAC-newline", func() cromlib.PackOptions {
			o := cromlib.DefaultPackOptions()
			o.UseACAC = true
			o.ACACDelimiter = '\n'
			o.Concurrency = 4
			return o
		}},
	}

	for _, ds := range datasets {
		// Generate dataset once
		dataDir := filepath.Join(workDir, "chunker_"+ds.name)
		os.MkdirAll(dataDir, 0755)
		dataPath := filepath.Join(dataDir, ds.name+".bin")

		fmt.Printf("\n  📦 Dataset: %s (%d MB)\n", ds.name, ds.size)
		if err := ds.gen(dataPath, ds.size); err != nil {
			fmt.Printf("     ❌ Generate failed: %v\n", err)
			continue
		}

		origData, _ := os.ReadFile(dataPath)
		origHash := sha256.Sum256(origData)
		originalSize := int64(len(origData))

		for _, ch := range chunkers {
			// Train a codebook for this chunker+dataset combo
			cbPath := filepath.Join(dataDir, ch.name+".cromdb")
			trainOpts := trainer.DefaultTrainOptions()
			trainOpts.InputDir = dataDir
			trainOpts.OutputPath = cbPath
			trainOpts.MaxCodewords = 8192
			trainOpts.Concurrency = 4

			// Adjust chunk size to match the chunker
			packOpts := ch.opts()
			trainOpts.ChunkSize = packOpts.ChunkSize

			_, err := trainer.Train(trainOpts)
			if err != nil {
				fmt.Printf("     ❌ [%s] Train failed: %v\n", ch.name, err)
				continue
			}

			cromPath := filepath.Join(dataDir, ch.name+".crom")
			restoredPath := filepath.Join(dataDir, ch.name+"_restored.bin")

			packStart := time.Now()
			metrics, err := cromlib.Pack(dataPath, cromPath, cbPath, packOpts)
			packDuration := time.Since(packStart)
			if err != nil {
				fmt.Printf("     ❌ [%s] Pack failed: %v\n", ch.name, err)
				continue
			}

			// Unpack + verify
			err = cromlib.Unpack(cromPath, restoredPath, cbPath, cromlib.DefaultUnpackOptions())
			integrity := false
			if err == nil {
				restored, _ := os.ReadFile(restoredPath)
				restHash := sha256.Sum256(restored)
				integrity = origHash == restHash
			}

			ratio := float64(metrics.OriginalSize) / float64(metrics.PackedSize)
			throughput := float64(originalSize) / (1024 * 1024) / packDuration.Seconds()

			integrityStr := "✅"
			if !integrity {
				integrityStr = "❌"
			}
			fmt.Printf("     [%s] Ratio=%.2fx | Hit=%.1f%% | Speed=%.1f MB/s | %s\n",
				ch.name, ratio, metrics.HitRate, throughput, integrityStr)

			results = append(results, ChunkerResult{
				Dataset:    ds.name,
				Chunker:    ch.name,
				Ratio:      ratio,
				HitRate:    metrics.HitRate,
				Throughput: throughput,
				PackedSize: int64(metrics.PackedSize),
				Duration:   packDuration,
				Integrity:  integrity,
			})

			// Cleanup intermediate files
			os.Remove(cromPath)
			os.Remove(restoredPath)
			os.Remove(cbPath)
		}

		os.Remove(dataPath)
	}

	return results
}
