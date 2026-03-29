package sdk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// TrainCommand holds arguments for training a codebook.
type TrainCommand struct {
	InputDir       string
	OutputPath     string
	MaxCodewords   int
}

// DefaultCompressor is the concrete SDK implementation.
type DefaultCompressor struct {
	eventBus *EventBus
}

func NewCompressor(bus *EventBus) Compressor {
	return &DefaultCompressor{
		eventBus: bus,
	}
}

func (c *DefaultCompressor) Pack(ctx context.Context, cmd PackCommand) (<-chan ProgressEvent, error) {
	progressCh := make(chan ProgressEvent, 100)

	go func() {
		defer close(progressCh)

		opts := cromlib.DefaultPackOptions()
		opts.Concurrency = cmd.Concurrency
		opts.EncryptionKey = cmd.EncryptionKey

		opts.OnProgress = func(processedBytes int) {
			progressCh <- ProgressEvent{
				Phase:     "packing",
				BytesRead: uint64(processedBytes),
			}
			if c.eventBus != nil {
				c.eventBus.Emit("pack_progress", map[string]interface{}{
					"bytes": processedBytes,
				})
			}
		}

		start := time.Now()
		metrics, err := cromlib.Pack(cmd.InputPath, cmd.OutputPath, cmd.CodebookPath, opts)

		if err != nil {
			progressCh <- ProgressEvent{Phase: fmt.Sprintf("error: %v", err)}
			if c.eventBus != nil {
				c.eventBus.Emit("pack_error", err.Error())
			}
			return
		}

		result := map[string]interface{}{
			"original":       metrics.OriginalSize,
			"packed":         metrics.PackedSize,
			"hitRate":        metrics.HitRate,
			"literalChunks":  metrics.LiteralChunks,
			"totalChunks":    metrics.TotalChunks,
			"avgSimilarity":  metrics.AvgSimilarity,
			"duration":       time.Since(start).String(),
			"ratio":          fmt.Sprintf("%.1f%%", float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100),
		}

		if c.eventBus != nil {
			c.eventBus.Emit("pack_done", result)
		}

		progressCh <- ProgressEvent{
			Phase:      "completed",
			Percentage: 100.0,
			TotalBytes: uint64(metrics.PackedSize),
		}
	}()

	return progressCh, nil
}

func (c *DefaultCompressor) Unpack(ctx context.Context, cmd UnpackCommand) (<-chan ProgressEvent, error) {
	progressCh := make(chan ProgressEvent, 100)

	go func() {
		defer close(progressCh)

		opts := cromlib.DefaultUnpackOptions()
		opts.Fuzziness = cmd.Fuzziness
		opts.EncryptionKey = cmd.EncryptionKey

		err := cromlib.Unpack(cmd.InputPath, cmd.OutputPath, cmd.CodebookPath, opts)
		if err != nil {
			progressCh <- ProgressEvent{Phase: fmt.Sprintf("error: %v", err)}
			if c.eventBus != nil {
				c.eventBus.Emit("unpack_error", err.Error())
			}
			return
		}

		if c.eventBus != nil {
			c.eventBus.Emit("unpack_done", map[string]string{
				"output": cmd.OutputPath,
			})
		}

		progressCh <- ProgressEvent{
			Phase:      "completed",
			Percentage: 100.0,
		}
	}()

	return progressCh, nil
}

func (c *DefaultCompressor) Train(ctx context.Context, cmd TrainCommand) (<-chan ProgressEvent, error) {
	progressCh := make(chan ProgressEvent, 100)

	go func() {
		defer close(progressCh)

		opts := trainer.DefaultTrainOptions()
		opts.InputDir = cmd.InputDir
		opts.OutputPath = cmd.OutputPath
		if cmd.MaxCodewords == 0 {
			cmd.MaxCodewords = 8192
		}
		opts.MaxCodewords = cmd.MaxCodewords

		opts.OnProgress = func(processedBytes int) {
			progressCh <- ProgressEvent{
				Phase:     "training",
				BytesRead: uint64(processedBytes),
			}
			if c.eventBus != nil {
				c.eventBus.Emit("train_progress", map[string]interface{}{
					"bytes": processedBytes,
				})
			}
		}

		start := time.Now()
		res, err := trainer.Train(opts)
		if err != nil {
			progressCh <- ProgressEvent{Phase: fmt.Sprintf("error: %v", err)}
			if c.eventBus != nil {
				c.eventBus.Emit("train_error", err.Error())
			}
			return
		}

		if c.eventBus != nil {
			c.eventBus.Emit("train_done", map[string]interface{}{
				"patterns": res.UniquePatterns,
				"elite":    res.SelectedElite,
				"bytes":    res.TotalBytes,
				"output":   cmd.OutputPath,
				"duration": time.Since(start).String(),
			})
		}

		progressCh <- ProgressEvent{
			Phase:      "completed",
			Percentage: 100.0,
		}
	}()

	return progressCh, nil
}

// Verify compares SHA-256 hashes of two files to confirm bit-perfect restoration.
func (c *DefaultCompressor) Verify(ctx context.Context, originalPath, restoredPath string) (bool, error) {
	hashA, err := hashFile(originalPath)
	if err != nil {
		return false, fmt.Errorf("verify original: %w", err)
	}
	hashB, err := hashFile(restoredPath)
	if err != nil {
		return false, fmt.Errorf("verify restored: %w", err)
	}

	match := hashA == hashB

	if c.eventBus != nil {
		c.eventBus.Emit("verify_done", map[string]interface{}{
			"match":    match,
			"original": hashA,
			"restored": hashB,
		})
	}

	return match, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
