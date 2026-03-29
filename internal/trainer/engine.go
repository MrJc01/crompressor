package trainer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
)

// TrainOptions configures the training process.
type TrainOptions struct {
	InputDir     string
	OutputPath   string
	MaxCodewords int // Number of codewords in the final codebook
	MaxPerBucket int // Max codewords per LSH bucket (diversity control)
	Concurrency  int
	ChunkSize    int // Size of chunks used for pattern extraction
	OnProgress   func(bytesProcessed int)
	DataAugmentation bool // Applies bit shifts before elite selection to combat overfitting

	// UpdatePath: path to an existing .cromdb to update incrementally.
	// Existing patterns are seeded into the frequency table with a high
	// initial count so they survive unless new data provides better alternatives.
	UpdatePath string

	// BasePath: path to a base .cromdb for transfer learning.
	// Base patterns are used as initial elite seeds. New patterns from
	// InputDir replace the least-frequent base patterns.
	BasePath string
}

// TrainResult contains metrics from the training run.
type TrainResult struct {
	TotalBytes      uint64
	TotalFiles      int
	UniquePatterns  int
	SelectedElite   int
	Duration        time.Duration
	MergedPatterns  int  // Patterns carried over from --update or --base
	ReplacedSlots   int  // Patterns replaced by new data during merge
}

// DefaultTrainOptions returns sensible defaults.
func DefaultTrainOptions() TrainOptions {
	return TrainOptions{
		MaxCodewords: 8192,
		MaxPerBucket: 64,
		Concurrency:  4,
		ChunkSize:    chunker.DefaultChunkSize,
		OnProgress:   func(n int) {},
	}
}

// Train crawls a directory, extracts pattern frequencies, selects the elite
// patterns, and writes a .cromdb codebook file.
func Train(opts TrainOptions) (*TrainResult, error) {
	start := time.Now()

	if opts.InputDir == "" || opts.OutputPath == "" {
		return nil, fmt.Errorf("trainer: InputDir and OutputPath are required")
	}
	if opts.MaxCodewords <= 0 {
		opts.MaxCodewords = 8192
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 4
	}
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = chunker.DefaultChunkSize
	}

	// Phase 1: Discover all files
	var files []string
	err := filepath.WalkDir(opts.InputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("trainer: walk directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("trainer: no files found in %s", opts.InputDir)
	}

	// Phase 2: Concurrent chunking and frequency counting
	ft := NewFrequencyTable()
	fc := chunker.NewFixedChunker(opts.ChunkSize)

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	var totalBytes uint64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < opts.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 16*1024*1024) // 16MB read buffer

			for path := range fileChan {
				f, err := os.Open(path)
				if err != nil {
					continue
				}

				for {
					n, errRead := f.Read(buf)
					if n > 0 {
						chunks := fc.Split(buf[:n])
						for _, c := range chunks {
							// Only record full-size chunks for consistent codebook entries
							if len(c.Data) == opts.ChunkSize {
								ft.Record(c.Data)
							}
						}
						mu.Lock()
						totalBytes += uint64(n)
						mu.Unlock()
						opts.OnProgress(n)
					}
					if errRead == io.EOF {
						break
					}
					if errRead != nil {
						break
					}
				}
				f.Close()
			}
		}()
	}
	wg.Wait()

	uniquePatterns := ft.Len()

	// Phase 2.5: Data Augmentation (Sprint 5.3)
	if opts.DataAugmentation {
		// Augment top 50% of the target words
		AugmentPatterns(ft, opts.MaxCodewords/2)
	}

	var mergedPatterns, replacedSlots int

	// Phase 3: Merge logic
	var selected [][]byte

	if opts.UpdatePath != "" {
		// --- INCREMENTAL UPDATE ---
		// Load existing patterns and seed them into the frequency table
		// with a base count boost so incumbents survive unless new data
		// provides significantly better alternatives.
		existingPatterns, err := codebook.ReadPatterns(opts.UpdatePath)
		if err != nil {
			return nil, fmt.Errorf("trainer: load update codebook: %w", err)
		}

		for _, p := range existingPatterns {
			if len(p) == opts.ChunkSize {
				// Seed with a boost count so existing patterns have incumbency advantage
				ft.RecordWithCount(p, 100)
				mergedPatterns++
			}
		}

		// Now select the best of old + new combined
		selected = SelectElite(ft, opts.MaxCodewords, opts.MaxPerBucket)
		replacedSlots = mergedPatterns - countOverlap(existingPatterns, selected)

	} else if opts.BasePath != "" {
		// --- TRANSFER LEARNING ---
		// Load base patterns as initial elite seeds.
		// Replace the least-frequent base slots with the best new patterns.
		basePatterns, err := codebook.ReadPatterns(opts.BasePath)
		if err != nil {
			return nil, fmt.Errorf("trainer: load base codebook: %w", err)
		}

		mergedPatterns = len(basePatterns)

		// Select new elite from fresh data only
		newElite := SelectElite(ft, opts.MaxCodewords, opts.MaxPerBucket)

		// Merge: base patterns fill the codebook first, then the best new
		// patterns replace the weakest base slots.
		selected = mergeBaseWithNew(basePatterns, newElite, opts.MaxCodewords)
		replacedSlots = len(selected) - countPresent(basePatterns, selected)

	} else {
		// --- STANDARD TRAINING ---
		selected = SelectElite(ft, opts.MaxCodewords, opts.MaxPerBucket)
	}

	// Phase 4: Write codebook
	if err := WriteCodebook(opts.OutputPath, selected); err != nil {
		return nil, err
	}

	return &TrainResult{
		TotalBytes:     totalBytes,
		TotalFiles:     len(files),
		UniquePatterns: uniquePatterns,
		SelectedElite:  len(selected),
		MergedPatterns: mergedPatterns,
		ReplacedSlots:  replacedSlots,
		Duration:       time.Since(start),
	}, nil
}

// countOverlap counts how many patterns from 'original' are still present in 'selected'.
func countOverlap(original [][]byte, selected [][]byte) int {
	set := make(map[uint64]bool, len(selected))
	for _, p := range selected {
		set[hashPattern(p)] = true
	}
	count := 0
	for _, p := range original {
		if set[hashPattern(p)] {
			count++
		}
	}
	return count
}

// countPresent counts how many base patterns survived into the final selection.
func countPresent(base [][]byte, selected [][]byte) int {
	return countOverlap(base, selected)
}

// mergeBaseWithNew combines base patterns with new patterns.
// Base patterns get priority; new patterns fill remaining slots.
// If there are more new candidates than remaining slots, only the best survive.
func mergeBaseWithNew(base, newPatterns [][]byte, maxCodewords int) [][]byte {
	// Deduplicate: build a set of base hashes
	baseSet := make(map[uint64]bool, len(base))
	for _, p := range base {
		baseSet[hashPattern(p)] = true
	}

	// Start with all base patterns (up to maxCodewords)
	result := make([][]byte, 0, maxCodewords)
	for i, p := range base {
		if i >= maxCodewords {
			break
		}
		result = append(result, p)
	}

	// Fill remaining slots with new patterns not in base
	for _, p := range newPatterns {
		if len(result) >= maxCodewords {
			break
		}
		if !baseSet[hashPattern(p)] {
			result = append(result, p)
		}
	}

	return result
}

// hashPattern returns a quick hash of a pattern for set operations.
func hashPattern(data []byte) uint64 {
	// Simple FNV-1a style hash for dedup
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}
