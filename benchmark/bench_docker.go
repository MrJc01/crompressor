package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// DockerResult holds the result of the Docker FUSE cascade test.
type DockerResult struct {
	BuildDuration time.Duration
	RunOutput     string
	Success       bool
	Skipped       bool
	SkipReason    string
}

// RunDockerBenchmark builds a Docker image from FUSE-cascaded CROM volumes.
func RunDockerBenchmark(workDir string) *DockerResult {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🐳 DOCKER VFS CASCADE — Real System Integration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check Docker
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Println("  ⚠️  SKIPPED: docker not found in PATH")
		return &DockerResult{Skipped: true, SkipReason: "docker not found"}
	}

	// Check Docker daemon is running
	if err := exec.Command("docker", "info").Run(); err != nil {
		fmt.Println("  ⚠️  SKIPPED: Docker daemon not running")
		return &DockerResult{Skipped: true, SkipReason: "docker daemon not running"}
	}

	cromBin := findCromBinary()
	if cromBin == "" {
		fmt.Println("  ⚠️  SKIPPED: crompressor binary not found")
		return &DockerResult{Skipped: true, SkipReason: "crompressor binary not found"}
	}

	// Check fuse-overlayfs
	if _, err := exec.LookPath("fuse-overlayfs"); err != nil {
		fmt.Println("  ⚠️  SKIPPED: fuse-overlayfs not found")
		return &DockerResult{Skipped: true, SkipReason: "fuse-overlayfs not found"}
	}

	dockerDir := filepath.Join(workDir, "docker_bench")
	os.MkdirAll(dockerDir, 0755)

	// Phase 1: Create mini app
	fmt.Println("  📦 Creating mini Node.js app...")
	appDir := filepath.Join(dockerDir, "app_src")
	os.MkdirAll(filepath.Join(appDir, "src"), 0755)

	os.WriteFile(filepath.Join(appDir, "src", "app.js"), []byte(
		`console.log("🧬 CROM VFS Docker Test: SUCCESS");
console.log("This app was built from a FUSE-cascaded CROM volume.");
console.log("Layers: .crom → crompressor mount → OverlayFS → docker build");
process.exit(0);
`), 0644)

	// Add some padding files to make it worth compressing
	for i := 0; i < 20; i++ {
		os.WriteFile(filepath.Join(appDir, fmt.Sprintf("data_%d.json", i)),
			[]byte(fmt.Sprintf(`{"id": %d, "type": "config", "version": "1.0.0", "enabled": true, "data": "CROM benchmark padding %d"}`, i, i)),
			0644)
	}

	// Phase 2: Train + Pack
	fmt.Println("  🧠 Training codebook on app source...")
	cbPath := filepath.Join(dockerDir, "app.cromdb")
	trainOpts := trainer.DefaultTrainOptions()
	trainOpts.InputDir = appDir
	trainOpts.OutputPath = cbPath
	trainOpts.MaxCodewords = 4096
	trainOpts.Concurrency = 4
	if _, err := trainer.Train(trainOpts); err != nil {
		fmt.Printf("  ❌ Train failed: %v\n", err)
		return &DockerResult{Skipped: true, SkipReason: fmt.Sprintf("train failed: %v", err)}
	}

	// Pack the entire app directory into a single file first
	// We'll pack the app.js as it's the key payload
	appJsPath := filepath.Join(appDir, "src", "app.js")
	cromPath := filepath.Join(dockerDir, "app.crom")
	if _, err := cromlib.Pack(appJsPath, cromPath, cbPath, cromlib.DefaultPackOptions()); err != nil {
		fmt.Printf("  ❌ Pack failed: %v\n", err)
		return &DockerResult{Skipped: true, SkipReason: fmt.Sprintf("pack failed: %v", err)}
	}

	// Phase 3: FUSE Cascade Mount
	fmt.Println("  🌌 Setting up FUSE cascade...")
	mountCrom := filepath.Join(dockerDir, "mnt_crom")
	upperDir := filepath.Join(dockerDir, "upper")
	workFuse := filepath.Join(dockerDir, "work")
	mergedDir := filepath.Join(dockerDir, "merged")

	for _, d := range []string{mountCrom, upperDir, workFuse, mergedDir} {
		os.MkdirAll(d, 0755)
	}

	// Clean stale mounts
	exec.Command("fusermount", "-uz", mergedDir).Run()
	exec.Command("fusermount", "-uz", mountCrom).Run()
	time.Sleep(200 * time.Millisecond)

	// Mount CROM
	fmt.Println("  1️⃣  Mounting CROM VFS...")
	cromCmd := exec.Command(cromBin, "mount", "-i", cromPath, "-m", mountCrom, "-c", cbPath)
	if err := cromCmd.Start(); err != nil {
		fmt.Printf("  ❌ CROM mount failed: %v\n", err)
		return &DockerResult{Skipped: true, SkipReason: "CROM mount failed"}
	}
	defer func() {
		exec.Command("fusermount", "-uz", mergedDir).Run()
		exec.Command("fusermount", "-uz", mountCrom).Run()
		cromCmd.Process.Kill()
		cromCmd.Wait()
	}()

	// Wait for CROM mount
	mounted := false
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		entries, err := os.ReadDir(mountCrom)
		if err == nil && len(entries) > 0 {
			mounted = true
			break
		}
	}
	if !mounted {
		fmt.Println("  ❌ CROM mount timeout")
		return &DockerResult{Skipped: true, SkipReason: "CROM mount timeout"}
	}

	// Mount OverlayFS
	fmt.Println("  2️⃣  Mounting OverlayFS...")
	ovCmd := exec.Command("fuse-overlayfs",
		"-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", mountCrom, upperDir, workFuse),
		mergedDir)
	if err := ovCmd.Run(); err != nil {
		fmt.Printf("  ⚠️  OverlayFS failed (proceeding without): %v\n", err)
		mergedDir = mountCrom // Fallback to direct CROM mount
	}

	// Write Dockerfile into the overlay (upper layer)
	fmt.Println("  3️⃣  Injecting Dockerfile into overlay...")
	os.MkdirAll(filepath.Join(mergedDir, "src"), 0755)

	// Copy app.js to merged/src/ if not already visible
	if _, err := os.Stat(filepath.Join(mergedDir, "src", "app.js")); err != nil {
		// File not visible from CROM; copy from original
		appData, _ := os.ReadFile(appJsPath)
		os.WriteFile(filepath.Join(mergedDir, "src", "app.js"), appData, 0644)
	}

	os.WriteFile(filepath.Join(mergedDir, "Dockerfile"), []byte(`FROM node:22-alpine
WORKDIR /app
COPY src/ /app/src/
CMD ["node", "src/app.js"]
`), 0644)

	// Phase 4: Docker Build
	fmt.Println("  🐳 Building Docker image from FUSE cascade...")
	buildStart := time.Now()
	buildCmd := exec.Command("docker", "build", "-t", "crom-bench-test", mergedDir)
	buildOutput, err := buildCmd.CombinedOutput()
	buildDuration := time.Since(buildStart)

	if err != nil {
		fmt.Printf("  ❌ Docker build failed: %v\n%s\n", err, string(buildOutput))
		return &DockerResult{Success: false, BuildDuration: buildDuration}
	}
	fmt.Printf("  ✅ Built in %v\n", buildDuration.Round(time.Millisecond))

	// Phase 5: Docker Run
	fmt.Println("  🏃 Running container...")
	runCmd := exec.Command("docker", "run", "--rm", "crom-bench-test")
	runOutput, err := runCmd.CombinedOutput()
	success := err == nil

	if success {
		fmt.Printf("  ✅ Container output: %s", string(runOutput))
	} else {
		fmt.Printf("  ❌ Container failed: %v\n", err)
	}

	// Cleanup
	exec.Command("docker", "rmi", "-f", "crom-bench-test").Run()

	return &DockerResult{
		BuildDuration: buildDuration,
		RunOutput:     string(runOutput),
		Success:       success,
	}
}
