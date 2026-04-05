// Crompressor Benchmark Suite V2 — E2E Pipeline + Advanced Tests
//
// Usage:
//   go run ./benchmark/                    # All benchmarks
//   go run ./benchmark/ --suite=basic      # Basic ratio tests only
//   go run ./benchmark/ --suite=scaling    # Scaling limits test
//   go run ./benchmark/ --suite=chunkers   # Chunker comparison
//   go run ./benchmark/ --suite=vfs        # VFS mount I/O test
//   go run ./benchmark/ --suite=docker     # Docker FUSE cascade
//   go run ./benchmark/ --suite=fractal    # Fractal O(1) engine
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

func main() {
	suite := flag.String("suite", "all", "Benchmark suite to run: all, basic, scaling, chunkers, vfs, docker, fractal")
	flag.Parse()

	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║    🧬 CROMPRESSOR BENCHMARK SUITE V2             ║")
	fmt.Println("╠═══════════════════════════════════════════════════╣")
	fmt.Println("║  Pipeline: Generate → Train → Pack → Unpack      ║")
	fmt.Println("║  Verify:   SHA-256 Roundtrip Integrity            ║")
	fmt.Println("║  Compare:  vs gzip -9, zstd -19                   ║")
	fmt.Printf("║  Suite:    %-40s ║\n", *suite)
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()

	workDir, err := os.MkdirTemp("", "crom_bench_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create work dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	var basicResults []BenchmarkResult
	var scalingResults []ScalingResult
	var chunkerResults []ChunkerResult
	var vfsResult *VFSResult
	var dockerResult *DockerResult
	var fractalResults []FractalResult

	runBasic := *suite == "all" || *suite == "basic"
	runScaling := *suite == "all" || *suite == "scaling"
	runChunkers := *suite == "all" || *suite == "chunkers"
	runVFS := *suite == "all" || *suite == "vfs"
	runDocker := *suite == "all" || *suite == "docker"
	runFractal := *suite == "all" || *suite == "fractal"

	// ═══════════════════════════════════════════════
	// BASIC: Ratio + Throughput (V1)
	// ═══════════════════════════════════════════════
	if runBasic {
		dataDir := filepath.Join(workDir, "datasets")
		fmt.Println("📦 Phase 1: Generating datasets...")
		datasets, err := GenerateAllDatasets(dataDir, 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate datasets: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   Generated %d datasets\n\n", len(datasets))

		for i, ds := range datasets {
			fmt.Printf("━━━ [%d/%d] %s (%s) ━━━\n", i+1, len(datasets), ds.Name, formatSize(ds.Size))

			result := BenchmarkResult{
				Dataset:      ds,
				OriginalSize: ds.Size,
			}

			cbPath := filepath.Join(workDir, ds.Name+".cromdb")
			trainStart := time.Now()

			trainOpts := trainer.DefaultTrainOptions()
			trainOpts.OutputPath = cbPath
			trainOpts.MaxCodewords = 8192
			trainOpts.Concurrency = 4

			trainDir := filepath.Join(workDir, "train_"+ds.Name)
			os.MkdirAll(trainDir, 0755)

			data, err := os.ReadFile(ds.Path)
			if err != nil {
				fmt.Printf("  ❌ read error: %v\n", err)
				continue
			}
			os.WriteFile(filepath.Join(trainDir, ds.Name+".bin"), data, 0644)
			trainOpts.InputDir = trainDir

			trainResult, err := trainer.Train(trainOpts)
			trainDuration := time.Since(trainStart)
			if err != nil {
				fmt.Printf("  ❌ Train: %v\n", err)
				continue
			}
			result.TrainDuration = trainDuration
			fmt.Printf("  🧠 Training: %d → %d elite in %v\n",
				trainResult.UniquePatterns, trainResult.SelectedElite, trainDuration.Round(time.Millisecond))

			cromPath := filepath.Join(workDir, ds.Name+".crom")
			packStart := time.Now()
			packOpts := cromlib.DefaultPackOptions()
			packOpts.Concurrency = 4
			metrics, err := cromlib.Pack(ds.Path, cromPath, cbPath, packOpts)
			packDuration := time.Since(packStart)
			if err != nil {
				fmt.Printf("  ❌ Pack: %v\n", err)
				continue
			}

			result.PackDuration = packDuration
			result.PackedSize = int64(metrics.PackedSize)
			result.Ratio = float64(metrics.OriginalSize) / float64(metrics.PackedSize)
			result.HitRate = metrics.HitRate
			result.Entropy = metrics.Entropy
			result.PackThroughput = float64(metrics.OriginalSize) / (1024 * 1024) / packDuration.Seconds()
			fmt.Printf("  📦 Pack: %s → %s (%.2fx) in %v [%.1f MB/s]\n",
				formatSize(int64(metrics.OriginalSize)), formatSize(int64(metrics.PackedSize)),
				result.Ratio, packDuration.Round(time.Millisecond), result.PackThroughput)

			restoredPath := filepath.Join(workDir, ds.Name+"_restored.bin")
			unpackStart := time.Now()
			err = cromlib.Unpack(cromPath, restoredPath, cbPath, cromlib.DefaultUnpackOptions())
			unpackDuration := time.Since(unpackStart)
			if err != nil {
				fmt.Printf("  ❌ Unpack: %v\n", err)
				continue
			}
			result.UnpackDuration = unpackDuration
			result.UnpackThroughput = float64(ds.Size) / (1024 * 1024) / unpackDuration.Seconds()

			origHash := sha256.Sum256(data)
			restored, _ := os.ReadFile(restoredPath)
			restHash := sha256.Sum256(restored)
			result.Integrity = origHash == restHash

			intStr := "✅"
			if !result.Integrity {
				intStr = "❌"
			}
			fmt.Printf("  🔒 SHA-256: %s | Unpack: %v [%.1f MB/s]\n", intStr,
				unpackDuration.Round(time.Millisecond), result.UnpackThroughput)

			result.External = CompareWithExternalTools(ds.Path)
			if len(result.External) > 0 {
				for _, ext := range result.External {
					fmt.Printf("  🔄 %s: %.2fx (%.1f MB/s)\n", ext.Tool, ext.Ratio, ext.Throughput)
				}
			}

			basicResults = append(basicResults, result)
			fmt.Println()
		}
	}

	// ═══════════════════════════════════════════════
	// SCALING: Progressive size limits
	// ═══════════════════════════════════════════════
	if runScaling {
		scalingResults = RunScalingBenchmark(workDir)
	}

	// ═══════════════════════════════════════════════
	// CHUNKERS: Fixed vs CDC vs ACAC
	// ═══════════════════════════════════════════════
	if runChunkers {
		chunkerResults = RunChunkerBenchmark(workDir)
	}

	// ═══════════════════════════════════════════════
	// VFS: FUSE mount I/O performance
	// ═══════════════════════════════════════════════
	if runVFS {
		vfsResult = RunVFSBenchmark(workDir)
	}

	// ═══════════════════════════════════════════════
	// DOCKER: FUSE cascade integration
	// ═══════════════════════════════════════════════
	if runDocker {
		dockerResult = RunDockerBenchmark(workDir)
	}

	// ═══════════════════════════════════════════════
	// FRACTAL: O(1) polynomial engine
	// ═══════════════════════════════════════════════
	if runFractal {
		fractalResults = RunFractalBenchmark(workDir)
	}

	// ═══════════════════════════════════════════════
	// GENERATE REPORT
	// ═══════════════════════════════════════════════
	reportPath := "benchmark/RESULTS.md"
	fmt.Printf("\n📝 Generating report: %s\n", reportPath)
	if err := GenerateFullReport(basicResults, scalingResults, chunkerResults,
		vfsResult, dockerResult, fractalResults, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate report: %v\n", err)
	}

	// Summary
	fmt.Println("\n╔═══════════════════════════════════════════════════╗")
	fmt.Println("║                  FINAL SUMMARY                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")

	if len(basicResults) > 0 {
		fmt.Println("  📊 Basic: ", len(basicResults), "datasets tested")
	}
	if len(scalingResults) > 0 {
		fmt.Printf("  📈 Scaling: tested up to %d MB\n", scalingResults[len(scalingResults)-1].SizeMB)
	}
	if len(chunkerResults) > 0 {
		fmt.Println("  🔀 Chunkers: ", len(chunkerResults), "comparisons")
	}
	if vfsResult != nil {
		fmt.Printf("  🗂️  VFS: %.1f MB/s (overhead %.1f%%)\n", vfsResult.SeqReadSpeed, vfsResult.Overhead)
	}
	if dockerResult != nil && !dockerResult.Skipped {
		status := "✅ SUCCESS"
		if !dockerResult.Success {
			status = "❌ FAILED"
		}
		fmt.Printf("  🐳 Docker: %s (build in %v)\n", status, dockerResult.BuildDuration.Round(time.Millisecond))
	}
	if len(fractalResults) > 0 {
		hits := 0
		for _, f := range fractalResults {
			if f.FractalHit {
				hits++
			}
		}
		fmt.Printf("  🧮 Fractal: %d/%d patterns triggered O(1)\n", hits, len(fractalResults))
	}

	fmt.Printf("\n📄 Full report saved to: %s\n", reportPath)
}
