package cromlib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/remote"
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

	// 3. Open Crom File and read headers
	var file io.ReaderAt
	var fileCloser io.Closer
	var fileSize int64

	isCloud := len(inputPath) > 7 && (inputPath[:7] == "http://" || inputPath[:8] == "https://")

	if isCloud {
		cr, err := remote.NewCloudReader(inputPath)
		if err != nil {
			return fmt.Errorf("grep remote: %w", err)
		}
		file = cr
		fileSize = cr.Size()
		fileCloser = io.NopCloser(nil)
	} else {
		localFile, err := os.Open(inputPath)
		if err != nil {
			return err
		}
		info, err := localFile.Stat()
		if err != nil {
			localFile.Close()
			return err
		}
		file = localFile
		fileSize = info.Size()
		fileCloser = localFile
	}
	defer fileCloser.Close()

	_ = fileSize

	readerInterface, ok := file.(io.Reader)
	if !ok {
		return fmt.Errorf("grep: file interface does not implement io.Reader")
	}

	reader := format.NewReader(readerInterface)
	// We read metadata via the internal offset sequence of CloudReader or localFile.
	header, blockTable, entries, err := reader.ReadMetadata("")
	if err != nil {
		return err
	}

	if header.IsPassthrough {
		return fmt.Errorf("grep não suportado em arquivos passthrough (sem chunks semânticos)")
	}

	// Calculate absolute offsets for physical blocks inside the .crom file (exactly like VFS does)
	tableSize := int(header.ChunkCount) * int(format.GetEntrySize(header.Version))
	hSize := format.HeaderSizeV2
	if header.Version == format.Version4 { hSize = format.HeaderSizeV4 }
	if header.Version == format.Version5 { hSize = format.HeaderSizeV5 }
	if header.Version >= format.Version6 { hSize = format.HeaderSizeV6 }

	baseOffset := int64(hSize + len(blockTable)*4 + tableSize)
	blockOffsets := make([]int64, len(blockTable))
	currentPhysical := baseOffset
	for i, size := range blockTable {
		blockOffsets[i] = currentPhysical
		currentPhysical += int64(size)
	}

	// 4. Scan ChunkTable (O(1) payload decompression!)
	fmt.Printf("\n🔍 Varrendo Matrix de Chunks (%d referências verticais)...\n", header.ChunkCount)
	matchCount := 0

	// Cache de blocos descompactados SOB DEMANDA
	blockCache := make(map[uint32][]byte)

	var approxOffset uint64
	for i, entry := range entries {
		cleanID := entry.CodebookID & 0x0FFFFFFFFFFFFFFF
		if matchedIDs[cleanID] {
			matchCount++
			fmt.Printf("  -> [Match #%d] Index %d (Offset %d)\n", matchCount, i, approxOffset)

			// Conditional Materialization
			if matchCount <= 20 {
				blockIdx := uint32(i / format.ChunksPerBlock)
				
				// Lazy Load Block ONLY if it is not cached
				pool, cached := blockCache[blockIdx]
				if !cached && blockIdx < uint32(len(blockTable)) {
					blockSize := blockTable[blockIdx]
					fileOff := blockOffsets[blockIdx]
					
					buf := make([]byte, blockSize)
					if _, err := file.ReadAt(buf, fileOff); err == nil || err == io.EOF {
						if decompressed, err := delta.DecompressPool(buf); err == nil {
							pool = decompressed
							blockCache[blockIdx] = pool
						}
					}
				}

				if pool != nil {
					blockStartChunkIdx := int64(blockIdx) * int64(format.ChunksPerBlock)
					blockStartGlobalOffset := entries[blockStartChunkIdx].DeltaOffset
					localOffset := entry.DeltaOffset - blockStartGlobalOffset

					endLocal := localOffset + uint64(entry.DeltaSize)
					if endLocal <= uint64(len(pool)) {
						res := pool[localOffset:endLocal]
						var chunk []byte

						if entry.CodebookID == format.LiteralCodebookID {
							chunk = res
						} else {
							isPatch := (entry.CodebookID & format.FlagIsPatch) != 0
							pattern, err := cb.Lookup(cleanID)
							if err == nil {
								// Truncate usable pattern exactly like unpacker
								usable := pattern
								if uint32(len(usable)) > entry.OriginalSize {
									usable = usable[:entry.OriginalSize]
								}
								if isPatch {
									chunk, _ = delta.ApplyPatch(usable, res)
								} else {
									if uint32(len(res)) > entry.OriginalSize {
										res = res[:entry.OriginalSize]
									}
									chunk = delta.Apply(usable, res)
								}
							}
						}

						// Truncate final reconstruct
						if uint32(len(chunk)) > entry.OriginalSize {
							chunk = chunk[:entry.OriginalSize]
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

	fmt.Printf("\n✔ Grep Neural (Zero-Payload) concluído em %v.\n", time.Since(start))
	fmt.Printf("  Total de Ocorrências Semânticas: %d (Zero descompressões de bloco desnecessárias executadas).\n", matchCount)
	return nil
}
