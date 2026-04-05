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
	Dataset          DatasetInfo
	OriginalSize     int64
	PackedSize       int64
	Ratio            float64
	TrainDuration    time.Duration
	PackDuration     time.Duration
	UnpackDuration   time.Duration
	PackThroughput   float64
	UnpackThroughput float64
	HitRate          float64
	Entropy          float64
	Integrity        bool
	External         []ExternalResult
}

// GenerateFullReport creates the complete Markdown report with all suites.
func GenerateFullReport(
	basic []BenchmarkResult,
	scaling []ScalingResult,
	chunkers []ChunkerResult,
	vfs *VFSResult,
	docker *DockerResult,
	fractal []FractalResult,
	outputPath string,
) error {
	var sb strings.Builder

	sb.WriteString("# 🧬 Crompressor — Benchmark Results V2\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s  \n", time.Now().Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("**Go:** %s  \n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("**OS/Arch:** %s/%s  \n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("**CPUs:** %d  \n\n", runtime.NumCPU()))

	// ── BASIC ──
	if len(basic) > 0 {
		sb.WriteString("## 📊 Compression Ratio & Throughput\n\n")
		sb.WriteString("| Dataset | Type | Original | Packed | Ratio | Pack | Unpack | Hit Rate | Entropy | SHA-256 |\n")
		sb.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
		for _, r := range basic {
			integrity := "✅"
			if !r.Integrity {
				integrity = "❌"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | **%.2fx** | %.1f MB/s | %.1f MB/s | %.1f%% | %.1f | %s |\n",
				r.Dataset.Name, r.Dataset.Description,
				formatSize(r.OriginalSize), formatSize(r.PackedSize),
				r.Ratio, r.PackThroughput, r.UnpackThroughput,
				r.HitRate, r.Entropy, integrity))
		}

		// External comparison
		hasExt := false
		for _, r := range basic {
			if len(r.External) > 0 {
				hasExt = true
				break
			}
		}
		if hasExt {
			sb.WriteString("\n### 🔄 Comparison vs Standard Tools\n\n")
			sb.WriteString("| Dataset | Crompressor | gzip -9 | zstd -19 | Best |\n")
			sb.WriteString("|---|---|---|---|---|\n")
			for _, r := range basic {
				cromRatio := fmt.Sprintf("%.2fx", r.Ratio)
				gzipRatio, zstdRatio := "N/A", "N/A"
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
		sb.WriteString("\n")
	}

	// ── SCALING ──
	if len(scaling) > 0 {
		sb.WriteString("## 📈 Scaling — Engine Limits\n\n")
		sb.WriteString("How does the engine behave as data grows?\n\n")
		sb.WriteString("| Size | Ratio | Pack Speed | Unpack Speed | Train Time | Hit Rate | Memory | SHA-256 |\n")
		sb.WriteString("|---|---|---|---|---|---|---|---|\n")
		for _, r := range scaling {
			integrity := "✅"
			if !r.Integrity {
				integrity = "❌"
			}
			unpackSpeed := float64(r.OriginalSize) / (1024 * 1024) / r.UnpackDuration.Seconds()
			sb.WriteString(fmt.Sprintf("| %d MB | **%.2fx** | %.1f MB/s | %.1f MB/s | %v | %.1f%% | %.0f MB | %s |\n",
				r.SizeMB, r.Ratio, r.PackThroughput, unpackSpeed,
				r.TrainDuration.Round(time.Millisecond), r.HitRate, r.MemUsedMB, integrity))
		}

		// Find degradation point
		if len(scaling) >= 2 {
			first := scaling[0]
			last := scaling[len(scaling)-1]
			degradation := ((first.PackThroughput - last.PackThroughput) / first.PackThroughput) * 100
			sb.WriteString(fmt.Sprintf("\n**Throughput degradation:** %.1f%% from %d MB to %d MB\n",
				degradation, first.SizeMB, last.SizeMB))
		}
		sb.WriteString("\n")
	}

	// ── CHUNKERS ──
	if len(chunkers) > 0 {
		sb.WriteString("## 🔀 Chunker Comparison — Fixed vs CDC vs ACAC\n\n")
		sb.WriteString("| Dataset | Chunker | Ratio | Hit Rate | Speed | SHA-256 |\n")
		sb.WriteString("|---|---|---|---|---|---|\n")
		for _, r := range chunkers {
			integrity := "✅"
			if !r.Integrity {
				integrity = "❌"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | **%.2fx** | %.1f%% | %.1f MB/s | %s |\n",
				r.Dataset, r.Chunker, r.Ratio, r.HitRate, r.Throughput, integrity))
		}

		// Find best chunker per dataset
		type key struct{ dataset, chunker string }
		bestPerDS := make(map[string]key)
		bestRatioPerDS := make(map[string]float64)
		for _, r := range chunkers {
			if r.Ratio > bestRatioPerDS[r.Dataset] {
				bestRatioPerDS[r.Dataset] = r.Ratio
				bestPerDS[r.Dataset] = key{r.Dataset, r.Chunker}
			}
		}
		sb.WriteString("\n**Best chunker per dataset:**\n")
		for ds, k := range bestPerDS {
			sb.WriteString(fmt.Sprintf("- **%s:** %s (%.2fx)\n", ds, k.chunker, bestRatioPerDS[ds]))
		}
		sb.WriteString("\n")
	}

	// ── VFS ──
	if vfs != nil {
		sb.WriteString("## 🗂️ VFS Mount — FUSE I/O Performance\n\n")
		integrity := "✅ MATCH"
		if !vfs.Integrity {
			integrity = "❌ MISMATCH"
		}
		sb.WriteString("| Metric | Value |\n")
		sb.WriteString("|---|---|\n")
		sb.WriteString(fmt.Sprintf("| File Size | %s |\n", formatSize(vfs.FileSize)))
		sb.WriteString(fmt.Sprintf("| Mount Latency | %v |\n", vfs.MountLatency.Round(time.Millisecond)))
		sb.WriteString(fmt.Sprintf("| First-Byte Latency | %v |\n", vfs.FirstByteLatency.Round(time.Microsecond)))
		sb.WriteString(fmt.Sprintf("| VFS Sequential Read | **%.1f MB/s** |\n", vfs.SeqReadSpeed))
		sb.WriteString(fmt.Sprintf("| Direct Disk Read | %.1f MB/s |\n", vfs.DirectReadSpeed))
		sb.WriteString(fmt.Sprintf("| Overhead | %.1f%% |\n", vfs.Overhead))
		sb.WriteString(fmt.Sprintf("| Integrity | %s |\n", integrity))
		sb.WriteString("\n")
	}

	// ── DOCKER ──
	if docker != nil {
		sb.WriteString("## 🐳 Docker FUSE Cascade — Real System Integration\n\n")
		if docker.Skipped {
			sb.WriteString(fmt.Sprintf("⚠️ **SKIPPED:** %s\n\n", docker.SkipReason))
		} else {
			status := "✅ **SUCCESS**"
			if !docker.Success {
				status = "❌ **FAILED**"
			}
			sb.WriteString(fmt.Sprintf("**Result:** %s  \n", status))
			sb.WriteString(fmt.Sprintf("**Build Time:** %v  \n", docker.BuildDuration.Round(time.Millisecond)))
			if docker.RunOutput != "" {
				sb.WriteString(fmt.Sprintf("**Container Output:**\n```\n%s```\n", docker.RunOutput))
			}
			sb.WriteString("\nThis test proves that a Docker container can be built from source files served\n")
			sb.WriteString("through a 3-layer FUSE cascade: `.crom` → CROM VFS Mount → OverlayFS → `docker build`.\n\n")
		}
	}

	// ── FRACTAL ──
	if len(fractal) > 0 {
		sb.WriteString("## 🧮 Fractal O(1) Engine — Polynomial Pattern Analysis\n\n")
		sb.WriteString("| Pattern | Description | Chunk | Ratio | Hit Rate | O(1)? | SHA-256 |\n")
		sb.WriteString("|---|---|---|---|---|---|---|\n")
		for _, r := range fractal {
			integrity := "✅"
			if !r.Integrity {
				integrity = "❌"
			}
			fractalStr := "🧮"
			if !r.FractalHit {
				fractalStr = "❌"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %dB | **%.2fx** | %.1f%% | %s | %s |\n",
				r.TestName, r.Description, r.ChunkSize, r.Ratio, r.HitRate, fractalStr, integrity))
		}
		sb.WriteString("\n")
	}

	// ── INTEGRITY ──
	sb.WriteString("## 🔒 Integrity Verification\n\n")
	allPassed := true
	for _, r := range basic {
		if !r.Integrity {
			allPassed = false
		}
	}
	for _, r := range scaling {
		if !r.Integrity {
			allPassed = false
		}
	}
	for _, r := range fractal {
		if !r.Integrity {
			allPassed = false
		}
	}
	if allPassed {
		sb.WriteString("✅ **All tests passed SHA-256 roundtrip verification.** Every byte was reconstructed perfectly.\n\n")
	} else {
		sb.WriteString("❌ **Some tests FAILED SHA-256 verification.** See details above.\n\n")
	}

	sb.WriteString("---\n\n*Generated by `go run ./benchmark/`*\n")

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

// GenerateReport is kept for backwards compatibility — delegates to GenerateFullReport.
func GenerateReport(results []BenchmarkResult, outputPath string) error {
	return GenerateFullReport(results, nil, nil, nil, nil, nil, outputPath)
}
