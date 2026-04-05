package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// VFSResult holds VFS benchmark metrics.
type VFSResult struct {
	MountLatency     time.Duration // Time to mount and become readable
	FirstByteLatency time.Duration // Time to read the first byte
	SeqReadSpeed     float64       // MB/s sequential read from VFS
	DirectReadSpeed  float64       // MB/s read from disk (baseline)
	Overhead         float64       // Percentage overhead vs direct
	Integrity        bool          // SHA-256 match via VFS read
	FileSize         int64
}

// RunVFSBenchmark mounts a .crom via FUSE and measures I/O performance.
func RunVFSBenchmark(workDir string) *VFSResult {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🗂️  VFS MOUNT BENCHMARK — FUSE I/O Performance")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check prerequisites
	if _, err := exec.LookPath("fusermount"); err != nil {
		fmt.Println("  ⚠️  SKIPPED: fusermount not found")
		return nil
	}

	cromBin := findCromBinary()
	if cromBin == "" {
		fmt.Println("  ⚠️  SKIPPED: crompressor binary not found (run 'make build' first)")
		return nil
	}

	// Setup
	vfsDir := filepath.Join(workDir, "vfs_bench")
	os.MkdirAll(vfsDir, 0755)

	dataPath := filepath.Join(vfsDir, "source.bin")
	cromPath := filepath.Join(vfsDir, "source.crom")
	cbPath := filepath.Join(vfsDir, "source.cromdb")
	mountPoint := filepath.Join(vfsDir, "mnt")
	os.MkdirAll(mountPoint, 0755)

	// Generate 10MB test file
	fmt.Println("  📦 Generating 10 MB test data...")
	if err := generateGoSource(dataPath, 10); err != nil {
		fmt.Printf("  ❌ Generate failed: %v\n", err)
		return nil
	}

	origData, _ := os.ReadFile(dataPath)
	origHash := sha256.Sum256(origData)
	fileSize := int64(len(origData))

	// Baseline: direct disk read speed
	fmt.Println("  📖 Measuring direct disk read speed...")
	directStart := time.Now()
	diskData, _ := os.ReadFile(dataPath)
	directDuration := time.Since(directStart)
	_ = diskData
	directSpeed := float64(fileSize) / (1024 * 1024) / directDuration.Seconds()

	// Train
	fmt.Println("  🧠 Training codebook...")
	trainDir := filepath.Join(vfsDir, "train_data")
	os.MkdirAll(trainDir, 0755)
	os.WriteFile(filepath.Join(trainDir, "source.bin"), origData, 0644)

	trainOpts := trainer.DefaultTrainOptions()
	trainOpts.InputDir = trainDir
	trainOpts.OutputPath = cbPath
	trainOpts.MaxCodewords = 8192
	trainOpts.Concurrency = 4
	if _, err := trainer.Train(trainOpts); err != nil {
		fmt.Printf("  ❌ Train failed: %v\n", err)
		return nil
	}

	// Pack
	fmt.Println("  📦 Packing...")
	if _, err := cromlib.Pack(dataPath, cromPath, cbPath, cromlib.DefaultPackOptions()); err != nil {
		fmt.Printf("  ❌ Pack failed: %v\n", err)
		return nil
	}

	// Mount
	fmt.Println("  🌌 Mounting CROM VFS via FUSE...")
	mountStart := time.Now()

	// Clean any stale mount
	exec.Command("fusermount", "-uz", mountPoint).Run()
	time.Sleep(100 * time.Millisecond)

	cmd := exec.Command(cromBin, "mount", "-i", cromPath, "-m", mountPoint, "-c", cbPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("  ❌ Mount failed: %v\n", err)
		return nil
	}
	defer func() {
		exec.Command("fusermount", "-uz", mountPoint).Run()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Wait for mount to become available (poll for up to 5 seconds)
	var mountLatency time.Duration
	mounted := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		entries, err := os.ReadDir(mountPoint)
		if err == nil && len(entries) > 0 {
			mountLatency = time.Since(mountStart)
			mounted = true
			break
		}
	}

	if !mounted {
		fmt.Println("  ❌ Mount timeout (5s) — FUSE mount did not become available")
		return nil
	}
	fmt.Printf("  ✅ Mounted in %v\n", mountLatency.Round(time.Millisecond))

	// Find the file inside mount
	var vfsFilePath string
	entries, _ := os.ReadDir(mountPoint)
	if len(entries) > 0 {
		vfsFilePath = filepath.Join(mountPoint, entries[0].Name())
	} else {
		fmt.Println("  ❌ No files found in mount point")
		return nil
	}

	// First byte latency
	fmt.Println("  ⏱️  Measuring first-byte latency...")
	fbStart := time.Now()
	f, err := os.Open(vfsFilePath)
	if err != nil {
		fmt.Printf("  ❌ Open failed: %v\n", err)
		return nil
	}
	oneByte := make([]byte, 1)
	f.Read(oneByte)
	firstByteLatency := time.Since(fbStart)
	f.Close()

	// Sequential read
	fmt.Println("  📖 Measuring VFS sequential read speed...")
	seqStart := time.Now()
	vfsData, err := os.ReadFile(vfsFilePath)
	seqDuration := time.Since(seqStart)
	if err != nil {
		fmt.Printf("  ❌ VFS read failed: %v\n", err)
		return nil
	}
	seqSpeed := float64(len(vfsData)) / (1024 * 1024) / seqDuration.Seconds()

	// Integrity
	vfsHash := sha256.Sum256(vfsData)
	integrity := origHash == vfsHash

	overhead := ((directSpeed / seqSpeed) - 1) * 100
	if seqSpeed == 0 {
		overhead = 999
	}

	integrityStr := "✅ MATCH"
	if !integrity {
		integrityStr = "❌ MISMATCH"
	}

	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     Mount latency:     %v\n", mountLatency.Round(time.Millisecond))
	fmt.Printf("     First-byte:        %v\n", firstByteLatency.Round(time.Microsecond))
	fmt.Printf("     VFS seq read:      %.1f MB/s\n", seqSpeed)
	fmt.Printf("     Direct disk read:  %.1f MB/s\n", directSpeed)
	fmt.Printf("     Overhead:          %.1f%%\n", overhead)
	fmt.Printf("     Integrity:         %s\n", integrityStr)

	return &VFSResult{
		MountLatency:     mountLatency,
		FirstByteLatency: firstByteLatency,
		SeqReadSpeed:     seqSpeed,
		DirectReadSpeed:  directSpeed,
		Overhead:         overhead,
		Integrity:        integrity,
		FileSize:         fileSize,
	}
}

// findCromBinary locates the crompressor binary.
func findCromBinary() string {
	candidates := []string{
		"./bin/crompressor",
		"./crompressor",
		"./crompressor-novo",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	// Try PATH
	if p, err := exec.LookPath("crompressor"); err == nil {
		return p
	}
	return ""
}

// verifyIntegrity compares SHA-256 of two files.
func verifyIntegrity(path1, path2 string) bool {
	data1, err := os.ReadFile(path1)
	if err != nil {
		return false
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		return false
	}
	h1 := sha256.Sum256(data1)
	h2 := sha256.Sum256(data2)
	return h1 == h2
}

// readAllFromVFS reads a file from the VFS mount and returns the data.
func readAllFromVFS(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
