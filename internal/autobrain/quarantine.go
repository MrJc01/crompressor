package autobrain

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MrJc01/crompressor/internal/network"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// ValidateAndPromoteBrain takes a raw byte payload from GossipSub, checks its signature and size,
// runs a Proof of Compression against a Canonical Matrix, and if successful, promotes it to the Brains folder.
func ValidateAndPromoteBrain(peerID string, payload []byte) error {
	// 1. Web of Trust Check
	if !network.IsPeerTrusted(peerID) {
		return fmt.Errorf("quarantine: peer %s is not in the Web of Trust (rejected)", peerID)
	}

	// 2. Strict Size Limit (OOM Mitigation - 32 MiB matching format.MaxMicroDictSize)
	if len(payload) > 32*1024*1024 {
		return fmt.Errorf("quarantine: payload exceeds 32MiB safety cap (potential OOM attack)")
	}

	// 3. Sandboxing (Save to Quarantine)
	home, _ := os.UserHomeDir()
	quarantineDir := filepath.Join(home, ".crompressor", "brains", "quarantine")
	os.MkdirAll(quarantineDir, 0700)

	tmpFile := filepath.Join(quarantineDir, fmt.Sprintf("brain_%s_%d.cromdb", peerID, time.Now().Unix()))
	if err := os.WriteFile(tmpFile, payload, 0600); err != nil {
		return fmt.Errorf("quarantine: failed to write sandboxed payload: %w", err)
	}

	// Clean up quarantine on exit (if not promoted, it deletes itself)
	defer os.Remove(tmpFile)

	// 4. Proof of Compression (Darwinian Consensus)
	// We need a canonical sample to compress. If it compresses better than threshold and doesn't crash, it proves validity.
	sampleText := "CROM CANONICAL SAMPLE: Validating Darwinian efficiency."
	for i := 0; i < 1000; i++ {
		sampleText += " Repeated sequence to test basic deduplication and codebook hit rate."
	}
	
	samplePath := filepath.Join(quarantineDir, "sample.txt")
	os.WriteFile(samplePath, []byte(sampleText), 0600)
	defer os.Remove(samplePath)

	outPath := filepath.Join(quarantineDir, "sample.crom")
	
	opts := cromlib.DefaultPackOptions()
	opts.ChunkSize = 64

	metrics, err := cromlib.Pack(samplePath, outPath, tmpFile, opts)
	if err != nil {
		return fmt.Errorf("quarantine: proof of compression failed (malformed or crash): %w", err)
	}
	defer os.Remove(outPath)

	// Minimal Darwinian Threshold
	ratio := float64(metrics.PackedSize) / float64(metrics.OriginalSize)
	if ratio > 0.95 {
		return fmt.Errorf("quarantine: rejected by Proof of Compression (ratio %.2f > 0.95: poor efficiency)", ratio)
	}

	// 5. Promotion
	brainsDir := filepath.Join(home, ".crompressor", "brains")
	os.MkdirAll(brainsDir, 0755)

	finalName := filepath.Join(brainsDir, fmt.Sprintf("trusted_%s.cromdb", peerID))
	
	// Fast Copy to final location (since Rename might fail if quarantine is mounted differently in weird environments, but usually safe in ~/.crompressor)
	if err := os.Rename(tmpFile, finalName); err != nil {
		return fmt.Errorf("quarantine: failed to promote brain: %w", err)
	}

	fmt.Printf("✔ [Hive-Mind] Brain promoted successfully from %s (Ratio: %.2f)\n", peerID, ratio)
	return nil
}
