package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"github.com/MrJc01/crompressor/pkg/format"
	"github.com/spf13/cobra"
)

func infoCmd() *cobra.Command {
	var input, codebookPath, encryptionKey string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display detailed statistics of a .crom file",
		Long:  `Analyzes the .crom file and displays header, block table, fragmentation, entropy, and CodebookID distribution.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("flag --input is required")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║             CROMPRESSOR INFO              ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			fmt.Println("╚═══════════════════════════════════════════╝")

			f, err := os.Open(input)
			if err != nil {
				return fmt.Errorf("error opening %s: %w", input, err)
			}
			defer f.Close()

			fileStat, err := f.Stat()
			if err != nil {
				return err
			}

			reader := format.NewReader(f)
			header, blockTable, entries, rStream, err := reader.ReadStream(encryptionKey)
			if err != nil {
				return fmt.Errorf("format parse error: %w", err)
			}
			compDeltaPool, _ := io.ReadAll(rStream)

			// Header
			fmt.Println("\n═══ Header ═══")
			fmt.Printf("  Version:       %d\n", header.Version)
			fmt.Printf("  Encrypted:     %v\n", header.IsEncrypted)
			fmt.Printf("  Original Size: %d bytes\n", header.OriginalSize)
			fmt.Printf("  Original Hash: %x\n", header.OriginalHash[:16])
			fmt.Printf("  Chunk Count:   %d\n", header.ChunkCount)
			fmt.Printf("  File Size:     %d bytes\n", fileStat.Size())

			// Block Table
			if header.Version == format.Version2 && len(blockTable) > 0 {
				fmt.Printf("\n═══ Block Table (%d blocks) ═══\n", len(blockTable))

				var totalCompressed uint64
				minBlock := uint32(^uint32(0))
				maxBlock := uint32(0)

				for _, size := range blockTable {
					totalCompressed += uint64(size)
					if size < minBlock {
						minBlock = size
					}
					if size > maxBlock {
						maxBlock = size
					}
				}

				avgBlock := totalCompressed / uint64(len(blockTable))
				fmt.Printf("  Total Compressed: %d bytes\n", totalCompressed)
				fmt.Printf("  Average Block:    %d bytes\n", avgBlock)
				fmt.Printf("  Min Block:        %d bytes\n", minBlock)
				fmt.Printf("  Max Block:        %d bytes\n", maxBlock)

				fragmentationRatio := float64(fileStat.Size()) / float64(header.OriginalSize)
				fmt.Printf("  Fragmentation:    %.4f (packed/original)\n", fragmentationRatio)
			}

			// Shannon Entropy
			if len(compDeltaPool) > 0 {
				ent := shannonEntropy(compDeltaPool)
				fmt.Printf("\n═══ Entropy ═══\n")
				fmt.Printf("  Shannon Entropy (Delta Pool): %.4f bits/byte\n", ent)
				fmt.Printf("  Max Possible:                 8.0000 bits/byte\n")
				fmt.Printf("  Randomness:                   %.2f%%\n", ent/8.0*100)
			}

			// CodebookID Distribution (Top-10)
			if len(entries) > 0 {
				fmt.Printf("\n═══ CodebookID Distribution (Top-10) ═══\n")

				freq := make(map[uint64]int)
				for _, e := range entries {
					freq[e.CodebookID]++
				}

				type idCount struct {
					ID    uint64
					Count int
				}
				sorted := make([]idCount, 0, len(freq))
				for id, c := range freq {
					sorted = append(sorted, idCount{id, c})
				}
				sort.Slice(sorted, func(i, j int) bool {
					return sorted[i].Count > sorted[j].Count
				})

				limit := 10
				if len(sorted) < limit {
					limit = len(sorted)
				}
				for i := 0; i < limit; i++ {
					pct := float64(sorted[i].Count) / float64(len(entries)) * 100
					fmt.Printf("  #%02d  CodebookID: %-10d  Count: %-6d  (%.2f%%)\n",
						i+1, sorted[i].ID, sorted[i].Count, pct)
				}
				fmt.Printf("  ... %d unique CodebookIDs total\n", len(freq))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input .crom file path")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Codebook path (.cromdb)")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Passphrase for decryption")

	return cmd
}

// shannonEntropy calculates the Shannon Entropy (bits per byte) of a byte slice.
func shannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	var freq [256]float64
	for _, b := range data {
		freq[b]++
	}
	total := float64(len(data))
	entropy := 0.0
	for _, f := range freq {
		if f > 0 {
			p := f / total
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}
