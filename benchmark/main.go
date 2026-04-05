// Crompressor Benchmark Suite — E2E Pipeline
//
// Usage: go run ./benchmark/
//
// This script generates deterministic datasets, trains codebooks,
// runs the full Pack→Unpack pipeline, verifies SHA-256 integrity,
// and produces a Markdown report comparing against gzip/zstd.
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

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║    🧬 CROMPRESSOR BENCHMARK SUITE                ║")
	fmt.Println("╠═══════════════════════════════════════════════════╣")
	fmt.Println("║  Pipeline: Generate → Train → Pack → Unpack      ║")
	fmt.Println("║  Verify:   SHA-256 Roundtrip Integrity            ║")
	fmt.Println("║  Compare:  vs gzip -9, zstd -19                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()

	workDir, err := os.MkdirTemp("", "crom_bench_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create work dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	dataDir := filepath.Join(workDir, "datasets")
	fmt.Println("📦 Phase 1: Generating datasets...")
	datasets, err := GenerateAllDatasets(dataDir, 50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate datasets: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Generated %d datasets\n\n", len(datasets))

	var results []BenchmarkResult

	for i, ds := range datasets {
		fmt.Printf("━━━ [%d/%d] %s (%s) ━━━\n", i+1, len(datasets), ds.Name, formatSize(ds.Size))

		result := BenchmarkResult{
			Dataset:      ds,
			OriginalSize: ds.Size,
		}

		// Phase 2: Train codebook
		cbPath := filepath.Join(workDir, ds.Name+".cromdb")
		fmt.Printf("  🧠 Training codebook... ")
		trainStart := time.Now()

		trainOpts := trainer.DefaultTrainOptions()
		trainOpts.InputDir = filepath.Dir(ds.Path)
		trainOpts.OutputPath = cbPath
		trainOpts.MaxCodewords = 8192
		trainOpts.Concurrency = 4

		// Create a temp dir with just this dataset for training
		trainDir := filepath.Join(workDir, "train_"+ds.Name)
		os.MkdirAll(trainDir, 0755)

		// Copy dataset to its own training dir
		data, err := os.ReadFile(ds.Path)
		if err != nil {
			fmt.Printf("❌ read error: %v\n", err)
			continue
		}
		os.WriteFile(filepath.Join(trainDir, ds.Name+".bin"), data, 0644)
		trainOpts.InputDir = trainDir

		trainResult, err := trainer.Train(trainOpts)
		trainDuration := time.Since(trainStart)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			continue
		}
		result.TrainDuration = trainDuration
		fmt.Printf("✅ %d patterns → %d elite in %v\n",
			trainResult.UniquePatterns, trainResult.SelectedElite, trainDuration.Round(time.Millisecond))

		// Phase 3: Pack
		cromPath := filepath.Join(workDir, ds.Name+".crom")
		fmt.Printf("  📦 Packing... ")
		packStart := time.Now()

		packOpts := cromlib.DefaultPackOptions()
		packOpts.Concurrency = 4

		metrics, err := cromlib.Pack(ds.Path, cromPath, cbPath, packOpts)
		packDuration := time.Since(packStart)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			continue
		}

		result.PackDuration = packDuration
		result.PackedSize = int64(metrics.PackedSize)
		result.Ratio = float64(metrics.OriginalSize) / float64(metrics.PackedSize)
		result.HitRate = metrics.HitRate
		result.Entropy = metrics.Entropy
		result.PackThroughput = float64(metrics.OriginalSize) / (1024 * 1024) / packDuration.Seconds()

		fmt.Printf("✅ %s → %s (%.2fx) in %v [%.1f MB/s]\n",
			formatSize(int64(metrics.OriginalSize)),
			formatSize(int64(metrics.PackedSize)),
			result.Ratio,
			packDuration.Round(time.Millisecond),
			result.PackThroughput,
		)

		// Phase 4: Unpack
		restoredPath := filepath.Join(workDir, ds.Name+"_restored.bin")
		fmt.Printf("  📂 Unpacking... ")
		unpackStart := time.Now()

		unpackOpts := cromlib.DefaultUnpackOptions()
		err = cromlib.Unpack(cromPath, restoredPath, cbPath, unpackOpts)
		unpackDuration := time.Since(unpackStart)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			continue
		}
		result.UnpackDuration = unpackDuration
		result.UnpackThroughput = float64(ds.Size) / (1024 * 1024) / unpackDuration.Seconds()
		fmt.Printf("✅ in %v [%.1f MB/s]\n", unpackDuration.Round(time.Millisecond), result.UnpackThroughput)

		// Phase 5: Verify SHA-256
		fmt.Printf("  🔒 Verifying SHA-256... ")
		origHash := sha256.Sum256(data)
		restored, err := os.ReadFile(restoredPath)
		if err != nil {
			fmt.Printf("❌ read error: %v\n", err)
			continue
		}
		restHash := sha256.Sum256(restored)

		if origHash == restHash {
			result.Integrity = true
			fmt.Printf("✅ MATCH (%x...)\n", origHash[:4])
		} else {
			result.Integrity = false
			fmt.Printf("❌ MISMATCH! orig=%x... rest=%x...\n", origHash[:4], restHash[:4])
		}

		// Phase 6: External comparison
		fmt.Printf("  🔄 Comparing vs gzip/zstd... ")
		result.External = CompareWithExternalTools(ds.Path)
		if len(result.External) > 0 {
			for _, ext := range result.External {
				fmt.Printf("\n     %s: %.2fx (%.1f MB/s)", ext.Tool, ext.Ratio, ext.Throughput)
			}
			fmt.Println()
		} else {
			fmt.Println("⚠ External tools not available")
		}

		results = append(results, result)
		fmt.Println()
	}

	// Generate report
	reportPath := "benchmark/RESULTS.md"
	fmt.Printf("📝 Generating report: %s\n", reportPath)
	if err := GenerateReport(results, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate report: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("\n╔═══════════════════════════════════════════════════╗")
	fmt.Println("║                    SUMMARY                        ║")
	fmt.Println("╠═══════════════════════════════════════════════════╣")
	allOK := true
	for _, r := range results {
		status := "✅"
		if !r.Integrity {
			status = "❌"
			allOK = false
		}
		fmt.Printf("║  %s %-18s %7s → %7s  %5.2fx   ║\n",
			status, r.Dataset.Name, formatSize(r.OriginalSize), formatSize(r.PackedSize), r.Ratio)
	}
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	if allOK {
		fmt.Println("\n✅ All integrity checks PASSED.")
	} else {
		fmt.Println("\n❌ Some integrity checks FAILED!")
		os.Exit(1)
	}

	fmt.Printf("\n📄 Full report saved to: %s\n", reportPath)
}
