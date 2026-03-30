// Package main provides the WebAssembly entry point for browser-based CROM compression.
//
// Build with: GOOS=js GOARCH=wasm go build -o examples/www/crompressor.wasm ./pkg/wasm
//
//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/search"
	"github.com/MrJc01/crompressor/pkg/format"
)

func main() {
	fmt.Println("[CROM WASM] Crompressor WebAssembly module loaded.")

	js.Global().Set("CromPack", js.FuncOf(cromPack))
	js.Global().Set("CromVersion", js.FuncOf(cromVersion))

	// Block forever — required for Go WASM to stay alive
	select {}
}

// cromVersion returns the engine version string.
func cromVersion(this js.Value, args []js.Value) interface{} {
	return fmt.Sprintf("Crompressor V7 (WASM) — Format %d", format.Version5)
}

// cromPack takes two Uint8Arrays (inputData, codebookData) and returns a packed Uint8Array.
// This is a simplified in-memory pack that does NOT use encryption or multi-pass.
func cromPack(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf("error: CromPack requires 2 arguments (inputData, codebookData)")
	}

	// 1. Copy input bytes from JS
	inputJS := args[0]
	inputLen := inputJS.Get("length").Int()
	inputData := make([]byte, inputLen)
	js.CopyBytesToGo(inputData, inputJS)

	// 2. Copy codebook bytes from JS
	cbJS := args[1]
	cbLen := cbJS.Get("length").Int()
	cbData := make([]byte, cbLen)
	js.CopyBytesToGo(cbData, cbJS)

	// 3. Open codebook from memory
	cb, err := codebook.OpenFromBytes(cbData)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: open codebook: %v", err))
	}
	defer cb.Close()

	// 4. Chunk & compress
	fc := chunker.NewFastCDCChunker(128)
	searcher := search.NewLSHSearcher(cb)
	chunks := fc.Split(inputData)

	var packedDeltas []byte
	var entries []format.ChunkEntry

	for _, c := range chunks {
		match, err := searcher.FindBestMatch(c.Data)
		if err != nil {
			// Literal fallback
			entries = append(entries, format.ChunkEntry{
				CodebookID:   format.LiteralCodebookID,
				DeltaSize:    uint32(len(c.Data)),
				OriginalSize: uint32(c.Size),
				DeltaOffset:  uint64(len(packedDeltas)),
			})
			packedDeltas = append(packedDeltas, c.Data...)
			continue
		}

		residual := delta.XOR(c.Data, match.Pattern)
		entries = append(entries, format.ChunkEntry{
			CodebookID:   match.CodebookID,
			DeltaSize:    uint32(len(residual)),
			OriginalSize: uint32(c.Size),
			DeltaOffset:  uint64(len(packedDeltas)),
		})
		packedDeltas = append(packedDeltas, residual...)
	}

	// 5. Compress delta pool
	compressedPool, err := delta.CompressPool(packedDeltas)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: compress pool: %v", err))
	}

	// 6. Build result as simple JSON-like report (binary output would need more JS bridge work)
	ratio := float64(len(compressedPool)) / float64(inputLen) * 100.0
	result := map[string]interface{}{
		"originalSize":   inputLen,
		"compressedSize": len(compressedPool),
		"chunks":         len(entries),
		"ratio":          fmt.Sprintf("%.1f%%", ratio),
	}

	// Return a JS object
	jsResult := js.Global().Get("Object").New()
	for k, v := range result {
		jsResult.Set(k, js.ValueOf(v))
	}

	// Also attach the raw compressed bytes
	jsBytes := js.Global().Get("Uint8Array").New(len(compressedPool))
	js.CopyBytesToJS(jsBytes, compressedPool)
	jsResult.Set("data", jsBytes)

	return jsResult
}
