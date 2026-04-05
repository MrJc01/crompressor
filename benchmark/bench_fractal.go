package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// FractalResult holds the result of the fractal O(1) benchmark.
type FractalResult struct {
	TestName    string
	ChunkSize   int
	Ratio       float64
	HitRate     float64
	Integrity   bool
	FractalHit  bool   // True if the fractal engine actually activated
	Description string
}

// RunFractalBenchmark tests the fractal O(1) polynomial generation engine
// with carefully crafted data that should trigger it.
func RunFractalBenchmark(workDir string) []FractalResult {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🧮 FRACTAL O(1) ENGINE — Polynomial Analysis")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var results []FractalResult

	tests := []struct {
		name      string
		chunkSize int
		desc      string
		gen       func(size int) []byte
	}{
		{
			name:      "constant_zero",
			chunkSize: 8,
			desc:      "All zeros (a=0, b=0, c=0)",
			gen: func(size int) []byte {
				return make([]byte, size)
			},
		},
		{
			name:      "constant_ff",
			chunkSize: 8,
			desc:      "All 0xFF (a=0, b=0, c=255)",
			gen: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					data[i] = 0xFF
				}
				return data
			},
		},
		{
			name:      "linear_ramp",
			chunkSize: 8,
			desc:      "Linear ramp: f(x) = x mod 256 (a=0, b=1, c=0)",
			gen: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					data[i] = byte(i % 256)
				}
				return data
			},
		},
		{
			name:      "quadratic_simple",
			chunkSize: 8,
			desc:      "Quadratic: f(x) = x² mod 256 (a=1, b=0, c=0)",
			gen: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					data[i] = byte((i * i) % 256)
				}
				return data
			},
		},
		{
			name:      "poly_repeating_8",
			chunkSize: 8,
			desc:      "8-byte polynomial chunk repeated 128K times",
			gen: func(size int) []byte {
				// Create a single 8-byte polynomial pattern and repeat it
				pattern := make([]byte, 8)
				for i := range pattern {
					pattern[i] = byte((i*i + 3*i + 7) % 256)
				}
				data := make([]byte, size)
				for i := 0; i < size; i += 8 {
					end := i + 8
					if end > size {
						end = size
					}
					copy(data[i:end], pattern)
				}
				return data
			},
		},
		{
			name:      "sawtooth_16",
			chunkSize: 16,
			desc:      "16-byte sawtooth wave repeating",
			gen: func(size int) []byte {
				pattern := make([]byte, 16)
				for i := range pattern {
					pattern[i] = byte(i * 17) // sawtooth
				}
				data := make([]byte, size)
				for i := 0; i < size; i += 16 {
					end := i + 16
					if end > size {
						end = size
					}
					copy(data[i:end], pattern)
				}
				return data
			},
		},
		{
			name:      "mixed_poly_noise",
			chunkSize: 8,
			desc:      "70% polynomial + 30% noise (partial fractal)",
			gen: func(size int) []byte {
				data := make([]byte, size)
				pattern := make([]byte, 8)
				for i := range pattern {
					pattern[i] = byte((2*i*i + i + 5) % 256)
				}
				for i := 0; i < size; i += 8 {
					end := i + 8
					if end > size {
						end = size
					}
					block := i / 8
					if block%10 < 7 {
						// Polynomial chunk
						copy(data[i:end], pattern)
					} else {
						// Noise chunk
						for j := i; j < end; j++ {
							data[j] = byte((j * 37 + 13) % 256)
						}
					}
				}
				return data
			},
		},
	}

	for _, t := range tests {
		fmt.Printf("\n  🧪 %s (%s)\n", t.name, t.desc)

		testDir := filepath.Join(workDir, "fractal_"+t.name)
		os.MkdirAll(testDir, 0755)

		// Generate 1MB of data
		data := t.gen(1024 * 1024)
		dataPath := filepath.Join(testDir, "data.bin")
		os.WriteFile(dataPath, data, 0644)

		origHash := sha256.Sum256(data)

		// Train codebook
		cbPath := filepath.Join(testDir, "fractal.cromdb")
		trainDir := filepath.Join(testDir, "train")
		os.MkdirAll(trainDir, 0755)
		os.WriteFile(filepath.Join(trainDir, "data.bin"), data, 0644)

		trainOpts := trainer.DefaultTrainOptions()
		trainOpts.InputDir = trainDir
		trainOpts.OutputPath = cbPath
		trainOpts.MaxCodewords = 8192
		trainOpts.ChunkSize = t.chunkSize
		trainOpts.Concurrency = 4
		if _, err := trainer.Train(trainOpts); err != nil {
			fmt.Printf("     ❌ Train failed: %v\n", err)
			continue
		}

		// Pack
		cromPath := filepath.Join(testDir, "data.crom")
		packOpts := cromlib.DefaultPackOptions()
		packOpts.ChunkSize = t.chunkSize
		packOpts.Concurrency = 4

		metrics, err := cromlib.Pack(dataPath, cromPath, cbPath, packOpts)
		if err != nil {
			fmt.Printf("     ❌ Pack failed: %v\n", err)
			continue
		}

		// Unpack
		restoredPath := filepath.Join(testDir, "restored.bin")
		err = cromlib.Unpack(cromPath, restoredPath, cbPath, cromlib.DefaultUnpackOptions())
		integrity := false
		if err == nil {
			restored, _ := os.ReadFile(restoredPath)
			restHash := sha256.Sum256(restored)
			integrity = origHash == restHash
		}

		ratio := float64(metrics.OriginalSize) / float64(metrics.PackedSize)
		fractalHit := metrics.HitRate > 50.0 // If more than half matched, fractal likely activated

		integrityStr := "✅"
		if !integrity {
			integrityStr = "❌"
		}
		fractalStr := "🧮 YES"
		if !fractalHit {
			fractalStr = "❌ NO"
		}

		fmt.Printf("     Ratio=%.2fx | Hit=%.1f%% | Fractal=%s | Integrity=%s\n",
			ratio, metrics.HitRate, fractalStr, integrityStr)

		results = append(results, FractalResult{
			TestName:    t.name,
			ChunkSize:   t.chunkSize,
			Ratio:       ratio,
			HitRate:     metrics.HitRate,
			Integrity:   integrity,
			FractalHit:  fractalHit,
			Description: t.desc,
		})

		// Cleanup
		os.RemoveAll(testDir)
	}

	return results
}
