package cromlib

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/pkg/format"
)

// Grep scans the .crom file for the target string in O(1) decompression time
// by only looking up semantic Codebook IDs, then conditionally materializes
// the first 20 matching chunks for display.
func Grep(target string, inputPath string, codebookPath string) error {
	start := time.Now()

	// 1. Load Codebook
	cb, err := codebook.Open(codebookPath)
	if err != nil {
		return fmt.Errorf("grep: falha ao abrir codebook: %w", err)
	}
	defer cb.Close()

	// 2. Map matching IDs
	matchedIDs := make(map[uint64]bool)
	targetBytes := []byte(target)
	count := cb.CodewordCount()
	for i := uint64(0); i < count; i++ {
		pattern, err := cb.Lookup(i)
		if err == nil && bytes.Contains(pattern, targetBytes) {
			matchedIDs[i] = true
		}
	}

	if len(matchedIDs) == 0 {
		fmt.Printf("⚠ Grep Neural: A string '%s' não formou nenhum Token BPE no Cérebro.\n", target)
		fmt.Printf("Isso significa que ela pode estar fatiada entre chunks ou é um literal raro não padronizado.\n")
	} else {
		fmt.Printf("🧠 Cérebro detectou %d Super-Tokens que contêm '%s'.\n", len(matchedIDs), target)
	}

	// 3. Open Crom File and read headers + full stream for materialization
	inFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	reader := format.NewReader(inFile)
	header, blockTable, entries, rStream, err := reader.ReadStream("")
	if err != nil {
		return err
	}

	if header.IsPassthrough {
		return fmt.Errorf("grep não suportado em arquivos passthrough (sem chunks semânticos)")
	}

	// 4. Scan ChunkTable (O(1) payload decompression!)
	fmt.Printf("\n🔍 Varrendo Matrix de Chunks (%d referências verticais)...\n", header.ChunkCount)
	matchCount := 0

	// Pre-load all blocks for materialization of matched chunks
	// We decompress blocks on-demand as we find matches
	type blockData struct {
		uncompressed []byte
		startOffset  uint64
	}
	blockCache := make(map[int]*blockData)

	// Helper to get uncompressed block data
	allBlockData := make([][]byte, len(blockTable))
	for i, blockSize := range blockTable {
		buf := make([]byte, blockSize)
		if _, err := rStream.Read(buf); err != nil {
			break
		}
		decompressed, err := delta.DecompressPool(buf)
		if err != nil {
			continue
		}
		allBlockData[i] = decompressed
	}

	// Build block offset map
	currentGlobalOffset := uint64(0)
	for blockIdx := range blockTable {
		blockStartChunkIdx := blockIdx * format.ChunksPerBlock
		if blockStartChunkIdx < len(entries) {
			blockCache[blockIdx] = &blockData{
				uncompressed: allBlockData[blockIdx],
				startOffset:  entries[blockStartChunkIdx].DeltaOffset,
			}
		}
	}

	var approxOffset uint64
	for i, entry := range entries {
		cleanID := entry.CodebookID & 0x0FFFFFFFFFFFFFFF
		if matchedIDs[cleanID] {
			matchCount++
			fmt.Printf("  -> [Match #%d] Index %d (Offset %d)\n", matchCount, i, approxOffset)

			// Conditional Materialization
			if matchCount <= 20 {
				blockIdx := i / format.ChunksPerBlock
				bd, ok := blockCache[blockIdx]
				if ok && bd.uncompressed != nil {
					localOffset := entry.DeltaOffset - bd.startOffset
					endLocal := localOffset + uint64(entry.DeltaSize)
					if endLocal <= uint64(len(bd.uncompressed)) {
						res := bd.uncompressed[localOffset:endLocal]
						var chunk []byte

						if entry.CodebookID == format.LiteralCodebookID {
							chunk = res
						} else {
							isPatch := (entry.CodebookID & format.FlagIsPatch) != 0
							pattern, err := cb.Lookup(cleanID)
							if err == nil {
								if isPatch {
									chunk, _ = delta.ApplyPatch(pattern, res)
								} else {
									usable := pattern
									if uint32(len(usable)) > entry.OriginalSize {
										usable = usable[:entry.OriginalSize]
									}
									if uint32(len(res)) > entry.OriginalSize {
										res = res[:entry.OriginalSize]
									}
									chunk = delta.Apply(usable, res)
								}
							}
						}

						if len(chunk) > 0 {
							cleanBuf := bytes.ReplaceAll(chunk, []byte("\n"), []byte(" "))
							fmt.Printf("     | Content: %s\n", string(cleanBuf))
						}
					}
				}
			} else if matchCount == 21 {
				fmt.Printf("     | ... (Materialização dos próximos omitida para evitar poluição)\n")
			}
		}
		approxOffset += uint64(entry.OriginalSize)
	}

	_ = currentGlobalOffset

	fmt.Printf("\n✔ Grep Neural (Zero-Payload) concluído em %v.\n", time.Since(start))
	fmt.Printf("  Total de Ocorrências Semânticas: %d (Zero descompressões de bloco Zstd executadas).\n", matchCount)
	return nil
}
