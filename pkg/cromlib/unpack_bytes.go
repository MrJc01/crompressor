package cromlib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/fractal"
	"github.com/MrJc01/crompressor/pkg/format"
)

// UnpackBytes decompresses a .crom payload entirely in memory, without touching
// the filesystem. This is the primary entry point for WASM and embedded environments.
//
// Parameters:
//   - cromData: raw .crom bytes (e.g. output of PackBytes)
//   - codebookData: raw .cromdb bytes
//   - opts: decompression options (same as Unpack)
//
// Returns the reconstructed original bytes and any error.
func UnpackBytes(cromData []byte, codebookData []byte, opts UnpackOptions) ([]byte, error) {
	if len(cromData) == 0 {
		return nil, fmt.Errorf("unpackbytes: empty input")
	}

	// 1. Open codebook from bytes
	cb, err := codebook.OpenFromBytes(codebookData)
	if err != nil {
		return nil, fmt.Errorf("unpackbytes: open codebook: %w", err)
	}
	defer cb.Close()

	// 2. Parse the .crom format from memory
	r := bytes.NewReader(cromData)
	reader := format.NewReader(r)
	header, blockTable, entries, rStream, err := reader.ReadStream(opts.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("unpackbytes: parse format: %w", err)
	}

	var output bytes.Buffer
	output.Grow(int(header.OriginalSize))

	// 3. Handle passthrough files
	if header.IsPassthrough || header.ChunkCount == 0 {
		if header.IsEncrypted {
			derivedKey := crypto.DeriveKey([]byte(opts.EncryptionKey), header.Salt[:])
			encData, _ := io.ReadAll(rStream)
			dec, err := crypto.Decrypt(derivedKey, encData)
			if err != nil {
				return nil, fmt.Errorf("unpackbytes: decrypt passthrough: %w", err)
			}
			output.Write(dec)
		} else {
			io.Copy(&output, rStream)
		}
		return output.Bytes(), nil
	}

	maxID := cb.CodewordCount() - 1

	// 4. V2+ block-based decompression
	if header.Version >= format.Version2 {
		var derivedKey []byte
		if header.IsEncrypted {
			derivedKey = crypto.DeriveKey([]byte(opts.EncryptionKey), header.Salt[:])
		}

		currentGlobalOffset := uint64(0)
		entryIdx := 0

		for i, blockSize := range blockTable {
			blockData := make([]byte, blockSize)
			if _, err := io.ReadFull(rStream, blockData); err != nil {
				return nil, fmt.Errorf("unpackbytes: read block %d: %w", i, err)
			}

			if header.IsEncrypted {
				dec, err := crypto.Decrypt(derivedKey, blockData)
				if err != nil {
					return nil, fmt.Errorf("unpackbytes: decrypt block %d: %w", i, err)
				}
				blockData = dec
			}

			uncompressedBlock, err := delta.DecompressPool(blockData)
			if err != nil {
				return nil, fmt.Errorf("unpackbytes: decompress block %d: %w", i, err)
			}

			blockEndOffset := currentGlobalOffset + uint64(len(uncompressedBlock))

			chunksInThisBlock := format.ChunksPerBlock
			if i == len(blockTable)-1 {
				chunksInThisBlock = int(header.ChunkCount) - (i * format.ChunksPerBlock)
			}

			for count := 0; count < chunksInThisBlock && entryIdx < len(entries); count++ {
				entry := entries[entryIdx]

				localOffset := entry.DeltaOffset - currentGlobalOffset
				endLocal := localOffset + uint64(entry.DeltaSize)
				if endLocal > uint64(len(uncompressedBlock)) {
					return nil, fmt.Errorf("unpackbytes: delta offset bounds error chunk %d block %d", entryIdx, i)
				}

				res := uncompressedBlock[localOffset:endLocal]
				chunk, err := reconstructChunk(entry, res, cb, maxID, opts.Fuzziness)
				if err != nil {
					return nil, fmt.Errorf("unpackbytes: reconstruct chunk %d: %w", entryIdx, err)
				}

				output.Write(chunk)
				entryIdx++
			}

			currentGlobalOffset = blockEndOffset
		}
	} else {
		// V1 Legacy
		compDeltaPool, err := io.ReadAll(rStream)
		if err != nil {
			return nil, err
		}
		uncompressedPool, err := delta.DecompressPool(compDeltaPool)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			endOffset := entry.DeltaOffset + uint64(entry.DeltaSize)
			res := uncompressedPool[entry.DeltaOffset:endOffset]
			chunk, err := reconstructChunk(entry, res, cb, maxID, opts.Fuzziness)
			if err != nil {
				return nil, err
			}
			output.Write(chunk)
		}
	}

	// 5. Integrity check
	if opts.Fuzziness == 0.0 {
		reconstructedHash := sha256.Sum256(output.Bytes())
		if reconstructedHash != header.OriginalHash {
			return nil, fmt.Errorf("unpackbytes: SHA-256 integrity mismatch")
		}
	}

	return output.Bytes(), nil
}

// reconstructChunk rebuilds a single chunk from its codebook entry and residual.
func reconstructChunk(entry format.ChunkEntry, res []byte, cb *codebook.Reader, maxID uint64, fuzziness float64) ([]byte, error) {
	targetID := entry.CodebookID

	if targetID == format.LiteralCodebookID {
		return res, nil
	}

	if entry.CodebookIndex == format.FractalCodebookIndex {
		seed := int64(targetID)
		return fractal.GeneratePolynomial(seed, int(entry.OriginalSize)), nil
	}

	isPatch := (targetID & format.FlagIsPatch) != 0
	cleanID := targetID & 0x0FFFFFFFFFFFFFFF
	_ = maxID // fuzziness not applied in WASM for determinism

	pattern, err := cb.Lookup(cleanID)
	if err != nil {
		return nil, fmt.Errorf("lookup codeword %d: %w", cleanID, err)
	}

	usablePattern := pattern
	if uint32(len(usablePattern)) > entry.OriginalSize {
		usablePattern = usablePattern[:entry.OriginalSize]
	}

	if isPatch {
		return delta.ApplyPatch(usablePattern, res)
	}

	if uint32(len(res)) > entry.OriginalSize {
		res = res[:entry.OriginalSize]
	}
	return delta.Apply(usablePattern, res), nil
}
