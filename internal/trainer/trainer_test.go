package trainer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrJc01/crompressor/internal/codebook"
)

// createTrainingData writes repetitive files to a directory for testing.
func createTrainingData(t *testing.T, dir string, fileCount int, pattern byte) {
	t.Helper()
	for i := 0; i < fileCount; i++ {
		data := make([]byte, 4096)
		for j := range data {
			data[j] = pattern + byte(j%16)
		}
		err := os.WriteFile(filepath.Join(dir, "file"+string(rune('a'+i))+".bin"), data, 0644)
		if err != nil {
			t.Fatalf("Failed to create training data: %v", err)
		}
	}
}

func TestTrain_Standard(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)
	createTrainingData(t, dataDir, 5, 0x00)

	outPath := filepath.Join(tmpDir, "brain.cromdb")

	opts := DefaultTrainOptions()
	opts.InputDir = dataDir
	opts.OutputPath = outPath
	opts.MaxCodewords = 256
	opts.ChunkSize = 128

	res, err := Train(opts)
	if err != nil {
		t.Fatalf("Standard training failed: %v", err)
	}

	if res.SelectedElite == 0 {
		t.Fatal("Expected SelectedElite > 0")
	}
	if res.TotalFiles != 5 {
		t.Fatalf("Expected 5 files, got %d", res.TotalFiles)
	}
	if res.MergedPatterns != 0 {
		t.Fatalf("Standard mode should have MergedPatterns=0, got %d", res.MergedPatterns)
	}

	// Verify the output file is a valid codebook
	patterns, err := codebook.ReadPatterns(outPath)
	if err != nil {
		t.Fatalf("Failed to read output codebook: %v", err)
	}
	if len(patterns) != res.SelectedElite {
		t.Fatalf("Pattern count mismatch: codebook has %d, result says %d", len(patterns), res.SelectedElite)
	}
}

func TestTrain_IncrementalUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	// Phase 1: Standard training with pattern A
	dataDir1 := filepath.Join(tmpDir, "data1")
	os.MkdirAll(dataDir1, 0755)
	createTrainingData(t, dataDir1, 3, 0x00)

	baseCB := filepath.Join(tmpDir, "base.cromdb")
	opts := DefaultTrainOptions()
	opts.InputDir = dataDir1
	opts.OutputPath = baseCB
	opts.MaxCodewords = 256
	opts.ChunkSize = 128

	res1, err := Train(opts)
	if err != nil {
		t.Fatalf("Phase 1 training failed: %v", err)
	}
	baseElite := res1.SelectedElite

	// Phase 2: Incremental update with new pattern B data
	dataDir2 := filepath.Join(tmpDir, "data2")
	os.MkdirAll(dataDir2, 0755)
	createTrainingData(t, dataDir2, 3, 0x80) // Different pattern family

	updatedCB := filepath.Join(tmpDir, "updated.cromdb")
	opts2 := DefaultTrainOptions()
	opts2.InputDir = dataDir2
	opts2.OutputPath = updatedCB
	opts2.MaxCodewords = 256
	opts2.ChunkSize = 128
	opts2.UpdatePath = baseCB

	res2, err := Train(opts2)
	if err != nil {
		t.Fatalf("Incremental update failed: %v", err)
	}

	if res2.MergedPatterns == 0 {
		t.Fatal("Expected MergedPatterns > 0 in incremental mode")
	}

	t.Logf("Base elite: %d, Updated elite: %d, Merged: %d, Replaced: %d",
		baseElite, res2.SelectedElite, res2.MergedPatterns, res2.ReplacedSlots)

	// The updated codebook should be valid
	patterns, err := codebook.ReadPatterns(updatedCB)
	if err != nil {
		t.Fatalf("Failed to read updated codebook: %v", err)
	}
	if len(patterns) == 0 {
		t.Fatal("Updated codebook is empty")
	}

	// Should have at least as many patterns as the base (incumbency advantage)
	if res2.SelectedElite < baseElite {
		t.Fatalf("Updated codebook (%d) should have >= base patterns (%d)",
			res2.SelectedElite, baseElite)
	}
}

func TestTrain_TransferLearning(t *testing.T) {
	tmpDir := t.TempDir()

	// Phase 1: Create a base codebook from generic data
	dataDir1 := filepath.Join(tmpDir, "generic")
	os.MkdirAll(dataDir1, 0755)
	createTrainingData(t, dataDir1, 5, 0x10)

	baseCB := filepath.Join(tmpDir, "generic.cromdb")
	opts := DefaultTrainOptions()
	opts.InputDir = dataDir1
	opts.OutputPath = baseCB
	opts.MaxCodewords = 128
	opts.ChunkSize = 128

	res1, err := Train(opts)
	if err != nil {
		t.Fatalf("Base training failed: %v", err)
	}
	baseCount := res1.SelectedElite

	// Phase 2: Transfer learning — fine-tune with domain-specific data
	dataDir2 := filepath.Join(tmpDir, "domain")
	os.MkdirAll(dataDir2, 0755)
	createTrainingData(t, dataDir2, 5, 0xA0) // Very different domain

	transferCB := filepath.Join(tmpDir, "domain.cromdb")
	opts2 := DefaultTrainOptions()
	opts2.InputDir = dataDir2
	opts2.OutputPath = transferCB
	opts2.MaxCodewords = 128
	opts2.ChunkSize = 128
	opts2.BasePath = baseCB

	res2, err := Train(opts2)
	if err != nil {
		t.Fatalf("Transfer learning failed: %v", err)
	}

	if res2.MergedPatterns == 0 {
		t.Fatal("Expected MergedPatterns > 0 in transfer learning mode")
	}

	t.Logf("Base: %d patterns, Transfer: %d patterns, Merged: %d, ReplacedSlots: %d",
		baseCount, res2.SelectedElite, res2.MergedPatterns, res2.ReplacedSlots)

	// The transfer codebook should contain some base patterns + new ones
	patterns, err := codebook.ReadPatterns(transferCB)
	if err != nil {
		t.Fatalf("Failed to read transfer codebook: %v", err)
	}
	if len(patterns) == 0 {
		t.Fatal("Transfer codebook is empty")
	}

	// Should fill up to MaxCodewords
	if len(patterns) > 128 {
		t.Fatalf("Transfer codebook exceeded MaxCodewords: %d > 128", len(patterns))
	}
}

func TestReadPatterns_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create patterns
	patterns := make([][]byte, 64)
	for i := range patterns {
		p := make([]byte, 128)
		for j := range p {
			p[j] = byte(i*7 + j%19)
		}
		patterns[i] = p
	}

	// Write codebook
	cbPath := filepath.Join(tmpDir, "test.cromdb")
	if err := WriteCodebook(cbPath, patterns); err != nil {
		t.Fatalf("WriteCodebook failed: %v", err)
	}

	// Read back
	readBack, err := codebook.ReadPatterns(cbPath)
	if err != nil {
		t.Fatalf("ReadPatterns failed: %v", err)
	}

	if len(readBack) != len(patterns) {
		t.Fatalf("Pattern count mismatch: wrote %d, read %d", len(patterns), len(readBack))
	}

	// Note: patterns were sorted by LSH bucket during write, so we can't
	// compare index-by-index. Instead, verify all original patterns exist.
	readSet := make(map[uint64]bool)
	for _, p := range readBack {
		readSet[hashPattern(p)] = true
	}
	for i, p := range patterns {
		if !readSet[hashPattern(p)] {
			t.Fatalf("Pattern %d not found in roundtrip", i)
		}
	}
}

func TestFrequencyTable_RecordWithCount(t *testing.T) {
	ft := NewFrequencyTable()

	data := make([]byte, 128)
	for i := range data {
		data[i] = byte(i)
	}

	// Record with count 100
	ft.RecordWithCount(data, 100)
	if ft.Len() != 1 {
		t.Fatalf("Expected 1 entry, got %d", ft.Len())
	}

	// Record same pattern again with count 50
	ft.RecordWithCount(data, 50)
	if ft.Len() != 1 {
		t.Fatalf("Expected still 1 entry, got %d", ft.Len())
	}

	// Check total count
	all := ft.All()
	if all[0].Count != 150 {
		t.Fatalf("Expected count 150, got %d", all[0].Count)
	}

	// Normal Record should add 1
	ft.Record(data)
	all = ft.All()
	if all[0].Count != 151 {
		t.Fatalf("Expected count 151, got %d", all[0].Count)
	}
}
