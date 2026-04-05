package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// ScalingResult holds metrics for a single scaling step.
type ScalingResult struct {
	SizeMB         int
	OriginalSize   int64
	PackedSize     int64
	Ratio          float64
	TrainDuration  time.Duration
	PackDuration   time.Duration
	UnpackDuration time.Duration
	PackThroughput float64
	HitRate        float64
	MemUsedMB      float64
	Integrity      bool
}

// RunScalingBenchmark tests progressive dataset sizes to find the engine's limits.
func RunScalingBenchmark(workDir string) []ScalingResult {
	sizes := []int{1, 10, 50, 100, 250, 500, 1024} // MB
	var results []ScalingResult

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  📈 SCALING BENCHMARK — Finding the Ceiling")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, sizeMB := range sizes {
		// Check available memory before proceeding
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// Safety: skip if dataset would use more than 70% of available RAM
		if sizeMB > 500 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if float64(sizeMB)*3 > float64(m.Sys)/(1024*1024)*0.7 {
				fmt.Printf("  ⚠️  %4d MB — SKIPPED (would exceed 70%% RAM safety limit)\n", sizeMB)
				continue
			}
		}

		// Check disk space
		if sizeMB > 200 {
			// Rough check: we need ~3x the dataset size (original + crom + restored)
			var stat struct{}
			_ = stat
			// Just proceed with a warning
			fmt.Printf("  ⚠️  Large dataset (%d MB) — ensure sufficient disk space\n", sizeMB)
		}

		fmt.Printf("\n  📊 Testing %d MB...\n", sizeMB)
		result := runSingleScaleTest(workDir, sizeMB)
		if result != nil {
			results = append(results, *result)
			fmt.Printf("     ✅ Ratio=%.2fx | Pack=%.1f MB/s | Unpack=%.1f MB/s | Mem=%.0f MB | Hit=%.1f%%\n",
				result.Ratio, result.PackThroughput,
				float64(result.OriginalSize)/(1024*1024)/result.UnpackDuration.Seconds(),
				result.MemUsedMB, result.HitRate)
		}

		// Force GC between tests to get clean memory readings
		runtime.GC()
	}

	return results
}

func runSingleScaleTest(workDir string, sizeMB int) *ScalingResult {
	dataDir := filepath.Join(workDir, fmt.Sprintf("scale_%dMB", sizeMB))
	os.MkdirAll(dataDir, 0755)

	// Generate semi-structured data (like server logs — compressible)
	dataPath := filepath.Join(dataDir, "data.bin")
	if err := generateLogs(dataPath, sizeMB); err != nil {
		fmt.Printf("     ❌ Generate failed: %v\n", err)
		return nil
	}

	info, _ := os.Stat(dataPath)
	originalSize := info.Size()

	// Train
	cbPath := filepath.Join(dataDir, "scale.cromdb")
	trainOpts := trainer.DefaultTrainOptions()
	trainOpts.InputDir = dataDir
	trainOpts.OutputPath = cbPath
	trainOpts.MaxCodewords = 8192
	trainOpts.Concurrency = 4

	trainStart := time.Now()
	_, err := trainer.Train(trainOpts)
	trainDuration := time.Since(trainStart)
	if err != nil {
		fmt.Printf("     ❌ Train failed: %v\n", err)
		return nil
	}

	// Get memory baseline
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Pack
	cromPath := filepath.Join(dataDir, "data.crom")
	packOpts := cromlib.DefaultPackOptions()
	packOpts.Concurrency = 4

	packStart := time.Now()
	metrics, err := cromlib.Pack(dataPath, cromPath, cbPath, packOpts)
	packDuration := time.Since(packStart)
	if err != nil {
		fmt.Printf("     ❌ Pack failed: %v\n", err)
		return nil
	}

	// Get memory after
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	memUsed := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
	if memUsed < 0 {
		memUsed = float64(memAfter.Alloc) / (1024 * 1024)
	}

	// Unpack
	restoredPath := filepath.Join(dataDir, "restored.bin")
	unpackStart := time.Now()
	err = cromlib.Unpack(cromPath, restoredPath, cbPath, cromlib.DefaultUnpackOptions())
	unpackDuration := time.Since(unpackStart)
	if err != nil {
		fmt.Printf("     ❌ Unpack failed: %v\n", err)
		return nil
	}

	// Integrity check
	integrity := verifyIntegrity(dataPath, restoredPath)

	// Cleanup large files to save disk
	os.Remove(restoredPath)
	os.Remove(dataPath)
	os.Remove(cromPath)
	os.Remove(cbPath)
	os.Remove(dataDir)

	ratio := float64(metrics.OriginalSize) / float64(metrics.PackedSize)

	return &ScalingResult{
		SizeMB:         sizeMB,
		OriginalSize:   originalSize,
		PackedSize:     int64(metrics.PackedSize),
		Ratio:          ratio,
		TrainDuration:  trainDuration,
		PackDuration:   packDuration,
		UnpackDuration: unpackDuration,
		PackThroughput: float64(originalSize) / (1024 * 1024) / packDuration.Seconds(),
		HitRate:        metrics.HitRate,
		MemUsedMB:      memUsed,
		Integrity:      integrity,
	}
}
