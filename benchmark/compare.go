package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ExternalResult holds the result of an external compression tool.
type ExternalResult struct {
	Tool           string
	OriginalSize   int64
	CompressedSize int64
	Ratio          float64
	Duration       time.Duration
	Throughput     float64 // MB/s
}

// CompareWithExternalTools compresses the file with gzip and zstd for comparison.
func CompareWithExternalTools(inputPath string) []ExternalResult {
	var results []ExternalResult

	info, err := os.Stat(inputPath)
	if err != nil {
		return results
	}
	originalSize := info.Size()

	// gzip -9
	if r, err := runExternalTool("gzip", []string{"-9", "-k", "-f"}, inputPath, ".gz", originalSize); err == nil {
		results = append(results, r)
	}

	// zstd -19
	if r, err := runExternalTool("zstd", []string{"-19", "-f", "--no-progress"}, inputPath, ".zst", originalSize); err == nil {
		results = append(results, r)
	}

	return results
}

func runExternalTool(tool string, args []string, inputPath, ext string, originalSize int64) (ExternalResult, error) {
	// Check if tool is available
	if _, err := exec.LookPath(tool); err != nil {
		return ExternalResult{}, fmt.Errorf("%s not found", tool)
	}

	outPath := inputPath + ext
	defer os.Remove(outPath)

	fullArgs := append(args, inputPath)
	if tool == "zstd" {
		fullArgs = append(args, inputPath, "-o", outPath)
	}

	start := time.Now()
	cmd := exec.Command(tool, fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return ExternalResult{}, err
	}
	duration := time.Since(start)

	compInfo, err := os.Stat(outPath)
	if err != nil {
		return ExternalResult{}, err
	}

	compressedSize := compInfo.Size()
	ratio := float64(originalSize) / float64(compressedSize)
	throughputMBs := float64(originalSize) / (1024 * 1024) / duration.Seconds()

	// Determine display name
	flagStr := strings.Join(args[:1], "")
	displayName := fmt.Sprintf("%s %s", tool, flagStr)

	return ExternalResult{
		Tool:           displayName,
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Ratio:          ratio,
		Duration:       duration,
		Throughput:     throughputMBs,
	}, nil
}

// CleanupExternal removes compressed artifacts from external tools.
func CleanupExternal(dir string) {
	extensions := []string{".gz", ".zst", ".xz", ".lz4"}
	for _, ext := range extensions {
		files, _ := filepath.Glob(filepath.Join(dir, "*"+ext))
		for _, f := range files {
			os.Remove(f)
		}
	}
}
