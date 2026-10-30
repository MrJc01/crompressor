//go:build cgo && !wasm

package delta

import (
	"fmt"
	"runtime"

	czstd "github.com/DataDog/zstd"
)

const maxCompressionLevel = 19

// MaxCompressionAvailable reports whether this build includes the C zstd encoder.
const MaxCompressionAvailable = true

// CompressPoolMax is CompressPoolDict at zstd level 19, much slower to encode. The
// frames are standard, so the pure Go decoder reads them.
func CompressPoolMax(pool, dict []byte) ([]byte, error) {
	if len(pool) == 0 {
		return []byte{}, nil
	}
	if len(dict) == 0 {
		return czstd.CompressLevel(nil, pool, maxCompressionLevel)
	}
	proc, err := czstd.NewBulkProcessor(dict, maxCompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("delta: init max compression: %w", err)
	}
	out, err := proc.Compress(nil, pool)
	runtime.KeepAlive(proc) // Its finalizer must not free the C dictionary during the call
	return out, err
}
