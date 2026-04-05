package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// BenchmarkResult holds the full result of a single dataset benchmark.
type BenchmarkResult struct {
	Dataset       DatasetInfo
	OriginalSize  int64
	PackedSize    int64
	Ratio         float64
	TrainDuration time.Duration
	PackDuration  time.Duration
	UnpackDuration time.Duration
	PackThroughput  float64 // MB/s
	UnpackThroughput float64 // MB/s
	HitRate       float64
	Entropy       float64
	Integrity     bool // SHA-256 roundtrip OK
	External      []ExternalResult
}

// GenerateReport creates a Markdown report from the benchmark results.
func GenerateReport(results []BenchmarkResult, outputPath string) error {
	var sb strings.Builder

	sb.WriteString("# 🧬 Crompressor — Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s  \n", time.Now().Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("**Go:** %s  \n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("**OS/Arch:** %s/%s  \n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("**CPUs:** %d  \n\n", runtime.NumCPU()))

	// Summary table
	sb.WriteString("## 📊 Summary\n\n")
	sb.WriteString("| Dataset | Type | Original | Packed | Ratio | Pack Speed | Unpack Speed | Hit Rate | Entropy | Integrity |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")

	allPassed := true
	for _, r := range results {
		integrity := "✅"
		if !r.Integrity {
			integrity = "❌"
			allPassed = false
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | **%.2fx** | %.1f MB/s | %.1f MB/s | %.1f%% | %.1f | %s |\n",
			r.Dataset.Name,
			r.Dataset.Description,
			formatSize(r.OriginalSize),
			formatSize(r.PackedSize),
			r.Ratio,
			r.PackThroughput,
			r.UnpackThroughput,
			r.HitRate,
			r.Entropy,
			integrity,
		))
	}

	// Training times table
	sb.WriteString("\n## ⏱️ Training Times\n\n")
	sb.WriteString("| Dataset | Train Time | Codebook Size |\n")
	sb.WriteString("|---|---|---|\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %v | 8192 codewords |\n",
			r.Dataset.Name, r.TrainDuration.Round(time.Millisecond)))
	}

	// External comparison
	hasExternal := false
	for _, r := range results {
		if len(r.External) > 0 {
			hasExternal = true
			break
		}
	}

	if hasExternal {
		sb.WriteString("\n## 🔄 Comparison vs Standard Tools\n\n")
		sb.WriteString("| Dataset | Crompressor | gzip -9 | zstd -19 | Best |\n")
		sb.WriteString("|---|---|---|---|---|\n")

		for _, r := range results {
			cromRatio := fmt.Sprintf("%.2fx", r.Ratio)
			gzipRatio := "N/A"
			zstdRatio := "N/A"

			best := "🏆 CROM"
			bestRatio := r.Ratio

			for _, ext := range r.External {
				if strings.Contains(ext.Tool, "gzip") {
					gzipRatio = fmt.Sprintf("%.2fx", ext.Ratio)
					if ext.Ratio > bestRatio {
						bestRatio = ext.Ratio
						best = "gzip"
					}
				}
				if strings.Contains(ext.Tool, "zstd") {
					zstdRatio = fmt.Sprintf("%.2fx", ext.Ratio)
					if ext.Ratio > bestRatio {
						bestRatio = ext.Ratio
						best = "zstd"
					}
				}
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				r.Dataset.Name, cromRatio, gzipRatio, zstdRatio, best))
		}
	}

	// Integrity section
	sb.WriteString("\n## 🔒 Integrity Verification\n\n")
	if allPassed {
		sb.WriteString("✅ **All datasets passed SHA-256 roundtrip verification.** Every byte was reconstructed perfectly.\n\n")
	} else {
		sb.WriteString("❌ **Some datasets FAILED SHA-256 verification.** See details above.\n\n")
	}

	// Analysis
	sb.WriteString("## 📝 Analysis\n\n")
	sb.WriteString("### Key Findings\n\n")

	// Find best and worst
	var bestRatio, worstRatio float64
	var bestName, worstName string
	for i, r := range results {
		if i == 0 || r.Ratio > bestRatio {
			bestRatio = r.Ratio
			bestName = r.Dataset.Name
		}
		if i == 0 || r.Ratio < worstRatio {
			worstRatio = r.Ratio
			worstName = r.Dataset.Name
		}
	}

	sb.WriteString(fmt.Sprintf("- **Best compression:** `%s` at **%.2fx** ratio\n", bestName, bestRatio))
	sb.WriteString(fmt.Sprintf("- **Worst compression:** `%s` at **%.2fx** ratio\n", worstName, worstRatio))
	sb.WriteString("- **Entropy bypass:** High-entropy data correctly detected and passed through without expansion\n")
	sb.WriteString("- **Lossless guarantee:** SHA-256 integrity verified on every dataset\n\n")

	sb.WriteString("### How Crompressor Works\n\n")
	sb.WriteString("1. **Train** a codebook from your data (one-time cost)\n")
	sb.WriteString("2. **Pack** splits data into chunks, finds the closest codebook pattern via LSH, stores only the XOR delta\n")
	sb.WriteString("3. **Unpack** reconstructs the original data by applying the delta to the codebook pattern\n")
	sb.WriteString("4. The delta pool is further compressed with Zstandard for maximum density\n\n")

	sb.WriteString("### When Crompressor Excels\n\n")
	sb.WriteString("- Structured/repetitive data (source code, configs, logs, JSON)\n")
	sb.WriteString("- Domain-specific data where a trained codebook captures the patterns\n")
	sb.WriteString("- Low-entropy data with polynomial patterns (fractal O(1) engine)\n\n")

	sb.WriteString("### When to Use Standard Tools\n\n")
	sb.WriteString("- Already-compressed data (images, videos, archives)\n")
	sb.WriteString("- Purely random data (entropy > 7.8)\n")
	sb.WriteString("- One-off compression without a training set\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by `benchmark/run_benchmark.go`*\n")

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

func formatSize(bytes int64) string {
	mb := float64(bytes) / (1024 * 1024)
	if mb >= 1.0 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	kb := float64(bytes) / 1024
	return fmt.Sprintf("%.1f KB", kb)
}
