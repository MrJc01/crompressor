//go:build !cgo || wasm

package delta

import "errors"

// MaxCompressionAvailable reports whether this build includes the C zstd encoder.
const MaxCompressionAvailable = false

// CompressPoolMax needs the C zstd encoder, which this build does not include.
func CompressPoolMax(pool, dict []byte) ([]byte, error) {
	return nil, errors.New("delta: --max-compression exige um binário compilado com CGO")
}
