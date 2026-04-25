// Package codebook provides public access to CROM codebook operations.
//
// This package re-exports types and functions from internal/codebook
// for use by satellite repositories (crompressor-sync, crompressor-wasm, etc).
package codebook

import (
	"github.com/MrJc01/crompressor/internal/codebook"
)

// Reader wraps the internal codebook.Reader for public use.
type Reader = codebook.Reader

// Header wraps the internal codebook.Header for public use.
type Header = codebook.Header

// Open opens a .cromdb codebook file and returns a Reader.
func Open(path string) (*Reader, error) {
	return codebook.Open(path)
}

// OpenFromBytes creates a Reader from raw bytes in memory (no file I/O).
// This is the primary entry point for WASM environments.
func OpenFromBytes(data []byte) (*Reader, error) {
	return codebook.OpenFromBytes(data)
}
