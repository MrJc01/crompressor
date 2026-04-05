package cromlib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MrJc01/crompressor/internal/entropy"
	"github.com/MrJc01/crompressor/internal/trainer"
)

// AutoPack compresses a file without requiring a pre-trained codebook.
// It performs a quick BPE training pass on the input file itself, generates
// an ephemeral codebook, and then packs using that self-derived brain.
// The brain is embedded in-band via V8 MicroDictionary for self-contained decompression.
func AutoPack(inputPath, outputPath string, opts PackOptions) (*Metrics, error) {
	// 1. Create a temporary directory for the ephemeral brain
	tmpDir, err := os.MkdirTemp("", "crom_autotrain_*")
	if err != nil {
		return nil, fmt.Errorf("autotrain: failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Create a temporary copy of the input file for training
	// (The trainer expects a directory, so we put the file in one)
	trainDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(trainDir, 0755)

	// Sample the input: copy up to 2MB for fast training
	srcFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("autotrain: open input: %w", err)
	}
	defer srcFile.Close()

	// EXPERT ROUTING: Early Entropy Guard
	eScore, startBytes, err := entropy.Analyze(srcFile, 65536)
	if err != nil {
		return nil, fmt.Errorf("autotrain: entropy analysis: %w", err)
	}

	// If it's urandom (High Entropy) or zeros (Low Entropy), skip BPE loop!
	if entropy.DetectHeuristicBypass(eScore, startBytes) || entropy.IsLowEntropy(eScore) {
		// Call Pack directly, which will handle the passthrough or fast-path
		return Pack(inputPath, outputPath, "", opts)
	}
	
	srcFile.Seek(0, 0)

	samplePath := filepath.Join(trainDir, filepath.Base(inputPath))
	dstFile, err := os.Create(samplePath)
	if err != nil {
		srcFile.Close()
		return nil, fmt.Errorf("autotrain: create sample: %w", err)
	}

	maxSample := int64(2 * 1024 * 1024)
	io.CopyN(dstFile, srcFile, maxSample)
	dstFile.Close()

	// 3. Train an ephemeral BPE codebook
	brainPath := filepath.Join(tmpDir, "ephemeral.cromdb")

	trainOpts := trainer.DefaultTrainOptions()
	trainOpts.InputDir = trainDir
	trainOpts.OutputPath = brainPath
	trainOpts.MaxCodewords = 2048 // Smaller for speed
	trainOpts.UseBPE = true
	trainOpts.ChunkSize = opts.ChunkSize
	if trainOpts.ChunkSize <= 0 {
		trainOpts.ChunkSize = 128
	}
	trainOpts.OnProgress = func(n int) {} // Silent

	_, err = trainer.Train(trainOpts)
	if err != nil {
		return nil, fmt.Errorf("autotrain: training failed: %w", err)
	}

	// 4. Pack using the ephemeral brain with Epigenesis enabled
	// This embeds the brain in-band via V8 MicroDictionary
	opts.AllowEpigenesis = true

	return Pack(inputPath, outputPath, brainPath, opts)
}
