package cromlib

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/fractal"
	"github.com/MrJc01/crompressor/internal/metrics"
	"github.com/MrJc01/crompressor/pkg/format"
)

// UnpackOptions defines settings for decompression.
type UnpackOptions struct {
	Fuzziness     float64 // 0.0 = lossless, > 0 = variational clone
	EncryptionKey string  // Passphrase for AES-256-GCM. If empty, uses no encryption.
	Strict        bool    // If true, aborts on decompression errors; if false, skips corrupted blocks.
}

// DefaultUnpackOptions returns sensible defaults (lossless).
func DefaultUnpackOptions() UnpackOptions {
	return UnpackOptions{
		Fuzziness: 0.0,
		Strict:    false,
	}
}

// Unpack reads a .crom file, extracts the deltas, looks up the codewords,
// rebuilds the original file directly to disk via streaming to prevent memory overflow.
func Unpack(inputPath, outputPath, codebookPath string, opts UnpackOptions) error {
	start := time.Now()

	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return fmt.Errorf("unpack: failed to open codebook: %w", err)
	}
	defer cb.Close()

	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("unpack: open .crom file: %w", err)
	}
	defer inFile.Close()

	reader := format.NewReader(inFile)
	header, blockTable, entries, rStream, err := reader.ReadStream(opts.EncryptionKey)
	if err != nil {
		return fmt.Errorf("unpack: parse format: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("unpack: open output file: %w", err)
	}
	defer outFile.Close()

	hasher := sha256.New()
	var writeOut func([]byte) error
	if opts.Fuzziness > 0.0 {
		// In variational mode, don't check hash
		writeOut = func(b []byte) error { _, e := outFile.Write(b); return e }
	} else {
		// Standard lossless check
		writeOut = func(b []byte) error {
			_, e1 := outFile.Write(b)
			_, e2 := hasher.Write(b)
			if e1 != nil { return e1 }
			return e2
		}
	}

	corruptBlocks := 0
	// PASSTHROUGH LOGIC (V3 / V16 Smart Passthrough)
	if header.IsPassthrough || header.ChunkCount == 0 {
		// The rest of the file is just the raw data or encrypted raw data
		if header.IsEncrypted {
			derivedKey := crypto.DeriveKey([]byte(opts.EncryptionKey), header.Salt[:])
			encData, _ := io.ReadAll(rStream)
			dec, err := crypto.Decrypt(derivedKey, encData)
			if err != nil {
				return fmt.Errorf("unpack: decrypt passthrough: %w", err)
			}
			if err := writeOut(dec); err != nil {
				return err
			}
		} else {
			if _, err := io.Copy(io.MultiWriter(outFile, hasher), rStream); err != nil {
				return err
			}
		}
		
		reconstructedHash := hasher.Sum(nil)
		if opts.Fuzziness == 0.0 {
			var h [32]byte
			copy(h[:], reconstructedHash)
			if h != header.OriginalHash {
				return fmt.Errorf("unpack: SECURITY/INTEGRITY FAILURE: reconstructed SHA-256 does not match original")
			}
		}
		fmt.Printf("✔ Unpack (passthrough) completed in %v\n", time.Since(start))
		return nil
	}

	maxID := cb.CodewordCount() - 1

	if header.Version >= format.Version2 {
		var derivedKey []byte
		if header.IsEncrypted {
			derivedKey = crypto.DeriveKey([]byte(opts.EncryptionKey), header.Salt[:])
		}

		currentGlobalOffset := uint64(0)
		entryIdx := 0

		// Stream block by block
		for i, blockSize := range blockTable {
			blockData := make([]byte, blockSize)
			if _, err := io.ReadFull(rStream, blockData); err != nil {
				return fmt.Errorf("unpack: unexpected end of delta pool reading block %d: %w", i, err)
			}

			if header.IsEncrypted {
				dec, err := crypto.Decrypt(derivedKey, blockData)
				if err != nil {
					return fmt.Errorf("unpack: decrypt block %d: %w", i, err)
				}
				blockData = dec
			}

			var uncompressedBlock []byte
			var decompressErr error
			for retry := 0; retry < 3; retry++ {
				uncompressedBlock, decompressErr = delta.DecompressPool(blockData)
				if decompressErr == nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			if decompressErr != nil {
				if opts.Strict {
					return fmt.Errorf("unpack: decompress block %d: %w", i, decompressErr)
				}
				fmt.Printf("[Warning] Failed to decompress block %d, skipping (tolerant mode): %v\n", i, decompressErr)
				corruptBlocks++
				
				// Advance state to the next block
				nextBlockIdx := (i + 1) * format.ChunksPerBlock
				if nextBlockIdx < len(entries) {
					currentGlobalOffset = entries[nextBlockIdx].DeltaOffset
				}
				
				// Skip all entries in this corrupted block by writing zero-filled chunks to maintain output alignment
				for entryIdx < len(entries) {
					if entryIdx >= nextBlockIdx {
						break
					}
					entry := entries[entryIdx]
					emptyChunk := make([]byte, entry.OriginalSize)
					outFile.Write(emptyChunk)
					entryIdx++
				}
				continue
			}

			blockEndOffset := currentGlobalOffset + uint64(len(uncompressedBlock))
			
			// Process the exact constant number of chunks that belong to this block
			chunksInThisBlock := format.ChunksPerBlock
			if i == len(blockTable)-1 {
				// Special case for last block which might be smaller
				chunksInThisBlock = int(header.ChunkCount) - (i * format.ChunksPerBlock)
			}
			
			for count := 0; count < chunksInThisBlock && entryIdx < len(entries); count++ {
				entry := entries[entryIdx]
				
				localOffset := entry.DeltaOffset - currentGlobalOffset
				endLocal := localOffset + uint64(entry.DeltaSize)
				if endLocal > uint64(len(uncompressedBlock)) {
					if opts.Strict {
						return fmt.Errorf("unpack: delta offset bounds error for chunk in block %d", i)
					}
					// If not strict, zero-fill current and skip
					fmt.Printf("[Warning] Invalid chunk boundaries in block %d, zero-filling\n", i)
					outFile.Write(make([]byte, entry.OriginalSize))
					corruptBlocks++
					entryIdx++
					continue
				}
				
				res := uncompressedBlock[localOffset:endLocal]
				
				targetID := entry.CodebookID
				var reconstructedChunk []byte

				if targetID == format.LiteralCodebookID {
					reconstructedChunk = res
				} else if entry.CodebookIndex == format.FractalCodebookIndex {
					// V26 Fractal Generative Engine -> O(1) Reconstruction bypasses dict
					seed := int64(targetID)
					reconstructedChunk = fractal.GeneratePolynomial(seed, int(entry.OriginalSize))
				} else {
					if opts.Fuzziness > 0.0 {
						spread := int(opts.Fuzziness * 100)
						if spread < 1 { spread = 1 }
						offset := uint64(rand.Intn(spread*2) - spread)
						if targetID+offset <= maxID {
							targetID += offset
						}
					}

					isPatch := (targetID & format.FlagIsPatch) != 0
					// Mask out Tier bits and Patch flag (clear upper 4 bits)
					cleanID := targetID & 0x0FFFFFFFFFFFFFFF
					pattern, err := cb.Lookup(cleanID)
					if err != nil {
						return fmt.Errorf("unpack: lookup codeword %d: %w", cleanID, err)
					}

					usablePattern := pattern
					if uint32(len(usablePattern)) > entry.OriginalSize {
						usablePattern = usablePattern[:entry.OriginalSize]
					}
					
					if isPatch {
						// Apply Edit Script
						reconstructedChunk, err = delta.ApplyPatch(usablePattern, res)
						if err != nil {
							return fmt.Errorf("unpack: failed to apply patch on chunk %d (block %d): %w", entryIdx, i, err)
						}
					} else {
						// Apply XOR
						if uint32(len(res)) > entry.OriginalSize {
							res = res[:entry.OriginalSize]
						}
						reconstructedChunk = delta.Apply(usablePattern, res)
					}
				}

				if err := writeOut(reconstructedChunk); err != nil {
					return fmt.Errorf("unpack: write chunk: %w", err)
				}
				
				entryIdx++
			}
			
			currentGlobalOffset = blockEndOffset
		}
	} else {
		// V1 Legacy Support (Load full pool)
		compDeltaPool, err := io.ReadAll(rStream)
		if err != nil { return err }
		uncompressedPool, err := delta.DecompressPool(compDeltaPool)
		if err != nil { return err }
		
		for _, entry := range entries {
			targetID := entry.CodebookID
			endOffset := entry.DeltaOffset + uint64(entry.DeltaSize)
			res := uncompressedPool[entry.DeltaOffset:endOffset]

			var reconstructedChunk []byte
			if targetID == format.LiteralCodebookID {
				reconstructedChunk = res
			} else if entry.CodebookIndex == format.FractalCodebookIndex {
				seed := int64(targetID)
				reconstructedChunk = fractal.GeneratePolynomial(seed, int(entry.OriginalSize))
			} else {
				isPatch := (targetID & format.FlagIsPatch) != 0
				cleanID := targetID & 0x0FFFFFFFFFFFFFFF
				pattern, err := cb.Lookup(cleanID)
				if err != nil { return err }

				usablePattern := pattern
				if uint32(len(usablePattern)) > entry.OriginalSize { usablePattern = usablePattern[:entry.OriginalSize] }
				
				if isPatch {
					reconstructedChunk, err = delta.ApplyPatch(usablePattern, res)
					if err != nil {
						return fmt.Errorf("unpack legacy: failed to apply patch: %w", err)
					}
				} else {
					if uint32(len(res)) > entry.OriginalSize {
						res = res[:entry.OriginalSize]
					}
					reconstructedChunk = delta.Apply(usablePattern, res)
				}
			}
			if err := writeOut(reconstructedChunk); err != nil { return err }
		}
	}

	reconstructedHash := hasher.Sum(nil)
	if opts.Fuzziness == 0.0 {
		var h [32]byte
		copy(h[:], reconstructedHash)
		if h != header.OriginalHash {
			if !opts.Strict && corruptBlocks > 0 {
				fmt.Printf("⚠ INTEGRITY MISMATCH: %d corrupted blocks were skipped.\n", corruptBlocks)
			} else {
				return fmt.Errorf("unpack: SECURITY/INTEGRITY FAILURE: reconstructed SHA-256 does not match original")
			}
		}
	} else {
		fmt.Printf("⚠ VARIATIONAL MODE ACTIVE (Fuzziness: %.2f)\n", opts.Fuzziness)
	}

	metrics.RecordUnpack(corruptBlocks)

	fmt.Printf("✔ Unpack completed in %v\n", time.Since(start))
	if opts.Fuzziness == 0.0 && corruptBlocks == 0 {
		fmt.Printf("  Integrity verified: SHA-256 match perfectly.\n")
	}
	return nil
}
