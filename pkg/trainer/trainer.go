// Package trainer provides public access to CROM codebook training operations.
//
// This package re-exports types and functions from internal/trainer for use by
// satellite repositories (crompressor-gui, etc).
package trainer

import (
	"github.com/MrJc01/crompressor/internal/trainer"
)

// TrainOptions wraps the internal trainer.TrainOptions.
type TrainOptions = trainer.TrainOptions

// TrainResult wraps the internal trainer.TrainResult.
type TrainResult = trainer.TrainResult

// DefaultTrainOptions returns sensible defaults for codebook training.
func DefaultTrainOptions() TrainOptions {
	return trainer.DefaultTrainOptions()
}

// Train trains a codebook from the given options.
func Train(opts TrainOptions) (*TrainResult, error) {
	return trainer.Train(opts)
}
