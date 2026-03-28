package trainer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MrJc01/crompressor/internal/chunker"
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
}

// TrainResult contains metrics from the training run.
type TrainResult struct {
	TotalBytes     uint64
	TotalFiles     int
	UniquePatterns int
	SelectedElite  int
	Duration       time.Duration
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

	// Phase 3: Select elite patterns
	selected := SelectElite(ft, opts.MaxCodewords, opts.MaxPerBucket)

	// Phase 4: Write codebook
	if err := WriteCodebook(opts.OutputPath, selected); err != nil {
		return nil, err
	}

	return &TrainResult{
		TotalBytes:     totalBytes,
		TotalFiles:     len(files),
		UniquePatterns: uniquePatterns,
		SelectedElite:  len(selected),
		Duration:       time.Since(start),
	}, nil
}
