package cromlib

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/crom-project/crom/internal/codebook"
	"github.com/crom-project/crom/internal/crypto"
	"github.com/crom-project/crom/internal/delta"
	"github.com/crom-project/crom/pkg/format"
)

// UnpackOptions defines settings for decompression.
type UnpackOptions struct {
	Fuzziness     float64 // 0.0 = lossless, > 0 = variational clone
	EncryptionKey string  // Passphrase for AES-256-GCM. If empty, uses no encryption.
}

// DefaultUnpackOptions returns sensible defaults (lossless).
func DefaultUnpackOptions() UnpackOptions {
	return UnpackOptions{
		Fuzziness: 0.0,
	}
}

// Unpack reads a .crom file, extracts the deltas, looks up the codewords,
// rebuilds the original file. If Fuzziness is 0, it verifies the SHA-256 hash perfectly.
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
	header, blockTable, entries, compDeltaPool, err := reader.Read(opts.EncryptionKey)
	if err != nil {
		return fmt.Errorf("unpack: parse format: %w", err)
	}

	var uncompressedPool []byte

	if header.Version == format.Version2 {
		var derivedKey []byte
		if header.IsEncrypted {
			derivedKey = crypto.DeriveKey([]byte(opts.EncryptionKey), header.Salt[:])
		}

		offset := 0
		for i, blockSize := range blockTable {
			if offset+int(blockSize) > len(compDeltaPool) {
				return fmt.Errorf("unpack: unexpected end of delta pool reading block %d (len(compDeltaPool)=%d, requested=%d)",
					i, len(compDeltaPool), offset+int(blockSize))
			}
			blockData := compDeltaPool[offset : offset+int(blockSize)]
			offset += int(blockSize)

			if header.IsEncrypted {
				dec, err := crypto.Decrypt(derivedKey, blockData)
				if err != nil {
					return fmt.Errorf("unpack: decrypt block %d: %w", i, err)
				}
				blockData = dec
			}

			uncompressedBlock, err := delta.DecompressPool(blockData)
			if err != nil {
				return fmt.Errorf("unpack: decompress block %d: %w", i, err)
			}

			uncompressedPool = append(uncompressedPool, uncompressedBlock...)
		}
	} else {
		uncompressedPool, err = delta.DecompressPool(compDeltaPool)
		if err != nil {
			return fmt.Errorf("unpack: decompress delta pool: %w", err)
		}
	}

	reconstructed := make([]byte, 0, header.OriginalSize)
	maxID := cb.CodewordCount() - 1

	for i, entry := range entries {
		targetID := entry.CodebookID

		// Variational Clones: Introduce deliberate noise by selecting nearby codewords
		if opts.Fuzziness > 0.0 {
			spread := int(opts.Fuzziness * 100)
			if spread < 1 {
				spread = 1
			}
			offset := uint64(rand.Intn(spread*2) - spread)

			if targetID+offset <= maxID {
				targetID += offset
			}
		}

		pattern, err := cb.Lookup(targetID)
		if err != nil {
			return fmt.Errorf("unpack: lookup codeword %d for chunk %d: %w", targetID, i, err)
		}

		endOffset := entry.DeltaOffset + uint64(entry.DeltaSize)
		if endOffset > uint64(len(uncompressedPool)) {
			return fmt.Errorf("unpack: delta offset bounds error for chunk %d", i)
		}

		res := uncompressedPool[entry.DeltaOffset:endOffset]

		usablePattern := pattern
		if uint32(len(usablePattern)) > entry.OriginalSize {
			usablePattern = usablePattern[:entry.OriginalSize]
		}
		if uint32(len(res)) > entry.OriginalSize {
			res = res[:entry.OriginalSize]
		}

		reconstructedChunk := delta.Apply(usablePattern, res)
		reconstructed = append(reconstructed, reconstructedChunk...)
	}

	reconstructedHash := sha256.Sum256(reconstructed)

	if opts.Fuzziness == 0.0 {
		if reconstructedHash != header.OriginalHash {
			return fmt.Errorf("unpack: SECURITY/INTEGRITY FAILURE: reconstructed SHA-256 (%x) does not match original (%x)",
				reconstructedHash[:8], header.OriginalHash[:8])
		}
	} else {
		fmt.Printf("⚠ VARIATIONAL MODE ACTIVE (Fuzziness: %.2f)\n", opts.Fuzziness)
		fmt.Printf("  Original Hash: %x\n", header.OriginalHash[:8])
		fmt.Printf("  Clone Hash:    %x\n", reconstructedHash[:8])
	}

	if err := os.WriteFile(outputPath, reconstructed, 0644); err != nil {
		return fmt.Errorf("unpack: write output: %w", err)
	}

	fmt.Printf("✔ Unpack completed in %v\n", time.Since(start))
	if opts.Fuzziness == 0.0 {
		fmt.Printf("  Integrity verified: SHA-256 match perfectly.\n")
	} else {
		fmt.Printf("  Variational clone generated successfully.\n")
	}
	fmt.Printf("  Restored output: %s (%d bytes)\n", outputPath, len(reconstructed))

	return nil
}
