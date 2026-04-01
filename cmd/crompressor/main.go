// CROM — Compressão de Realidade e Objetos Mapeados
//
// crom is the unified CLI for the CROM compression system.
// It provides subcommands for packing (compressing), unpacking (decompressing),
// and verifying files using a Codebook-based compression scheme.
//
// Usage:
//
//	crompressor pack   --input FILE --output FILE --codebook FILE
//	crompressor unpack --input FILE --output FILE --codebook FILE
//	crompressor verify --original FILE --restored FILE
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/MrJc01/crompressor/internal/autobrain"
	"github.com/MrJc01/crompressor/internal/entropy"

	"github.com/MrJc01/crompressor/internal/trainer"

	"github.com/MrJc01/crompressor/pkg/cromlib"

	"github.com/MrJc01/crompressor/pkg/format"
	cromsync "github.com/MrJc01/crompressor/pkg/sync"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var version = "24.0.0-Singularity"

func main() {
	rootCmd := &cobra.Command{
		Use:   "crompressor",
		Short: "Crompressor — Compilador de Realidade",
		Long: `CROM (Compressão de Realidade e Objetos Mapeados)

Um sistema de compressão lossless de nova geração baseado em
um Codebook Universal de padrões. Transforma dados brutos em
mapas de referências determinísticos com fidelidade bit-a-bit.

"Não comprimimos dados. Compilamos realidade."`,
		Version: version,
	}

	rootCmd.AddCommand(packCmd())
	rootCmd.AddCommand(unpackCmd())
	rootCmd.AddCommand(trainCmd())
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(benchmarkCmd())
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(grepCmd())

	// Comandos sensíveis a OS injetados via build tags
	addSystemCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Initialize Prometheus Metrics

// Initialize CromNode

// Node identity banner

// Setup Protocol

// Start DHT

// Background task to announce local files

// wait for pubsub to settle

// Simplistic announce of existing files

// For testing purpose: if there's a file called 'sync_test.txt' we can initiate a sync to a peer
// A better CLI would have `crompressor daemon sync <peer_multiaddr> <filename>` but we're keeping it running

// HTTP API for CLI integration (crompressor share, crom info --network)

// To avoid unused var error for syncProto for now

func trainCmd() *cobra.Command {
	var inputDir, outputPath, updatePath, basePath string
	var maxCodewords, concurrency, chunkSize int
	var augmentTrain, useBPE bool

	cmd := &cobra.Command{
		Use:   "train",
		Short: "Treina um Codebook a partir de dados em um diretório",
		Long:  `Rastreia arquivos em lote, extrai padrões frequentes de 128 bytes e seleciona uma elite com LSH diversity para formar um CROMDB universal.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputDir == "" || outputPath == "" {
				return fmt.Errorf("flags --input e --output são obrigatórias")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║          CROM TRAIN (Treinador)           ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input Dir: %-29s ║\n", inputDir)
			fmt.Printf("║  Output:    %-29s ║\n", outputPath)
			fmt.Printf("║  Target:    %-29d ║\n", maxCodewords)
			if updatePath != "" {
				fmt.Printf("║  Mode:      %-29s ║\n", "Incremental Update")
				fmt.Printf("║  Base CB:   %-29s ║\n", updatePath)
			} else if basePath != "" {
				fmt.Printf("║  Mode:      %-29s ║\n", "Transfer Learning")
				fmt.Printf("║  Base CB:   %-29s ║\n", basePath)
			} else {
				fmt.Printf("║  Mode:      %-29s ║\n", "Standard")
			}
			if useBPE {
				fmt.Printf("║  Engine:    %-29s ║\n", "BPE (Neural Tokenizer)")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			bar := progressbar.DefaultBytes(
				-1,
				"Extraindo Padrões",
			)

			opts := trainer.DefaultTrainOptions()
			opts.InputDir = inputDir
			opts.OutputPath = outputPath
			if maxCodewords > 0 {
				opts.MaxCodewords = maxCodewords
			}
			if concurrency > 0 {
				opts.Concurrency = concurrency
			}
			if chunkSize > 0 {
				opts.ChunkSize = chunkSize
			}
			opts.UpdatePath = updatePath
			opts.BasePath = basePath
			opts.DataAugmentation = augmentTrain
			opts.UseBPE = useBPE
			opts.OnProgress = func(n int) {
				bar.Add(n)
			}

			res, err := trainer.Train(opts)
			if err != nil {
				return fmt.Errorf("erro no treinamento: %v", err)
			}

			fmt.Printf("\n✔ Training completed in %v\n", res.Duration)
			fmt.Printf("  Files Parsed:    %d\n", res.TotalFiles)
			fmt.Printf("  Total Bytes:     %d\n", res.TotalBytes)
			fmt.Printf("  Unique Patterns: %d\n", res.UniquePatterns)
			fmt.Printf("  Elite Selected:  %d (Codebook Gerado)\n", res.SelectedElite)
			if res.MergedPatterns > 0 {
				fmt.Printf("  Merged Patterns: %d\n", res.MergedPatterns)
				fmt.Printf("  Replaced Slots:  %d\n", res.ReplacedSlots)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputDir, "input", "i", "", "Diretório com os dados de treinamento")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Caminho do .cromdb gerado")
	cmd.Flags().IntVarP(&maxCodewords, "size", "s", 8192, "Número máximo de padrões no codebook (Target)")
	cmd.Flags().IntVarP(&chunkSize, "chunk-size", "k", 0, "Tamanho base dos chunks (0 = auto)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Número de goroutines para processamento paralelo")
	cmd.Flags().StringVar(&updatePath, "update", "", "Caminho para .cromdb existente (atualização incremental)")
	cmd.Flags().StringVar(&basePath, "base", "", "Caminho (.cromdb) de base (Transfer Learning)")
	cmd.Flags().BoolVar(&augmentTrain, "augment", false, "Aplica shift de bits estocástico para diversificar o conjunto elite (combate overfitting)")
	cmd.Flags().BoolVar(&useBPE, "use-bpe", false, "Usa motor iterativo neural BPE em vez da Frequência Absoluta Bruta")

	return cmd
}

func packCmd() *cobra.Command {
	var input, output, codebookPath string
	var concurrency, chunkSize int
	var useCDC bool
	var encryptionKey string
	var autoBrain bool
	var multiPass bool
	var streamMode bool
	var brainDir string

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Compila (comprime) um arquivo usando o Codebook",
		Long:  `Divide o arquivo em chunks, busca padrões no Codebook e gera um arquivo .crom compacto.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || output == "" {
				return fmt.Errorf("flags --input e --output são obrigatórias")
			}

			var det *autobrain.DetectionResult
			var useAutoTrain bool
			if autoBrain {
				if codebookPath != "" {
					return fmt.Errorf("--auto-brain e --codebook não podem ser usados juntos")
				}
				router, err := autobrain.NewBrainRouter(brainDir)
				if err != nil {
					return fmt.Errorf("falha ao inicializar auto-brain: %w", err)
				}
				cb, result, err := router.SelectBrain(input)
				if err != nil {
					// No suitable brain found — fall back to Auto-Training
					fmt.Printf("⚠ Nenhum Brain adequado encontrado para '%s'. Ativando Auto-Training...\n", result.Category)
					useAutoTrain = true
					det = result
				} else {
					codebookPath = cb
					det = result
				}
			} else if codebookPath == "" {
				// Zero-config mode: no codebook provided at all → Auto-Training
				fmt.Println("🧠 Modo Zero-Config: Nenhum codebook fornecido. Ativando Auto-Training...")
				useAutoTrain = true
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║            CROM PACK (Compilador)         ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			fmt.Printf("║  Output:   %-30s ║\n", output)
			if autoBrain {
				fmt.Printf("║  AutoBrain: %-29s ║\n", det.Category)
				fmt.Printf("║   ↳ Codebook: %-27s ║\n", filepath.Base(codebookPath))
			} else {
				fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			}
			if encryptionKey != "" {
				fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			info, err := os.Stat(input)
			if err != nil {
				return err
			}

			bar := progressbar.DefaultBytes(
				info.Size(),
				"Compilando",
			)

			opts := cromlib.DefaultPackOptions()
			if concurrency > 0 {
				opts.Concurrency = concurrency
			}
			if chunkSize > 0 {
				opts.ChunkSize = chunkSize
			}
			opts.UseCDC = useCDC
			opts.MultiPass = multiPass
			if encryptionKey != "" {
				opts.EncryptionKey = encryptionKey
			}
			opts.OnProgress = func(n int) {
				bar.Add(n)
			}

			if streamMode {
				// Stream mode: read from file or stdin without Seek
				var reader io.Reader
				if input == "-" {
					reader = os.Stdin
				} else {
					f, err := os.Open(input)
					if err != nil {
						return err
					}
					defer f.Close()
					reader = f
				}

				metrics, err := cromlib.PackStream(reader, output, codebookPath, opts)
				if err != nil {
					return fmt.Errorf("erro no stream pack: %v", err)
				}

				fmt.Printf("\n✔ Stream Pack completed\n")
				fmt.Printf("  Original Size: %d bytes\n", metrics.OriginalSize)
				fmt.Printf("  Packed Size:   %d bytes (%.2f%% ratio)\n",
					metrics.PackedSize,
					float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100)
				fmt.Printf("  Hit Rate:      %.2f%%\n", metrics.HitRate)
				return nil
			}

			var metrics *cromlib.Metrics
			if useAutoTrain {
				metrics, err = cromlib.AutoPack(input, output, opts)
			} else {
				metrics, err = cromlib.Pack(input, output, codebookPath, opts)
			}
			if err != nil {
				return fmt.Errorf("erro no empacotamento: %v", err)
			}

			fmt.Printf("\n✔ Pack completed in %v\n", metrics.Duration)
			fmt.Printf("  Original Size: %d bytes\n", metrics.OriginalSize)
			fmt.Printf("  Packed Size:   %d bytes (%.2f%% ratio)\n",
				metrics.PackedSize,
				float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100)
			fmt.Printf("  Hit Rate:      %.2f%% dos chunks no Radar\n", metrics.HitRate)
			fmt.Printf("  Data Entropy:  %.2f bits/byte\n", metrics.Entropy)

			var litPct float64
			if metrics.TotalChunks > 0 {
				litPct = float64(metrics.LiteralChunks) / float64(metrics.TotalChunks) * 100
			}
			fmt.Printf("  Literal Chunks: %d/%d (%.2f%%)\n", metrics.LiteralChunks, metrics.TotalChunks, litPct)
			fmt.Printf("  Avg Similarity: %.2f%%\n", metrics.AvgSimilarity*100)

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Caminho do arquivo de entrada")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Caminho do arquivo .crom de saída")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().BoolVar(&autoBrain, "auto-brain", false, "Seleciona o codebook automaticamente baseado no conteúdo do arquivo")
	cmd.Flags().StringVar(&brainDir, "brain-dir", filepath.Join(os.Getenv("HOME"), ".crompressor", "brains"), "Diretório contendo codebooks para Auto-Brain")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Número de goroutines para processamento paralelo")
	cmd.Flags().IntVarP(&chunkSize, "chunk-size", "k", 0, "Tamanho base dos chunks (0 = auto)")
	cmd.Flags().BoolVar(&useCDC, "cdc", false, "Habilitar Content-Defined Chunking")
	cmd.Flags().BoolVar(&multiPass, "multi-pass", false, "Habilitar compressão LSH Top-K em Duas Passagens (Otimiza delta)")
	cmd.Flags().BoolVar(&streamMode, "stream", false, "Modo streaming — comprime pipes/stdin sem Seek (ex: tail -f | crompressor pack --stream)")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Chave/Senha para criptografia AES-256-GCM")

	return cmd
}

func unpackCmd() *cobra.Command {
	var input, output, codebookPath string
	var fuzziness float64
	var encryptionKey string
	var strict bool

	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "Decompila (descomprime) um arquivo .crom",
		Long:  `Lê o arquivo .crom, busca os padrões no Codebook e reconstrói o arquivo original.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || output == "" || codebookPath == "" {
				return fmt.Errorf("flags --input, --output e --codebook são obrigatórias")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║          CROM UNPACK (Decompilador)       ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			fmt.Printf("║  Output:   %-30s ║\n", output)
			fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			if fuzziness > 0 {
				fmt.Printf("║  Fuzziness: %-29.2f ║\n", fuzziness)
			}
			if encryptionKey != "" {
				fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			opts := cromlib.DefaultUnpackOptions()
			opts.Fuzziness = fuzziness
			opts.Strict = strict
			if encryptionKey != "" {
				opts.EncryptionKey = encryptionKey
			}
			if err := cromlib.Unpack(input, output, codebookPath, opts); err != nil {
				return fmt.Errorf("erro no desempacotamento: %v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Caminho do arquivo .crom")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Caminho do arquivo de saída restaurado")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().Float64Var(&fuzziness, "fuzziness", 0.0, "Variação na reconstrução (0 = Lossless, ex: 0.1 = 10% Fuzziness)")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Chave/Senha para descriptografia")
	cmd.Flags().BoolVar(&strict, "strict", false, "Abortar desempacotamento ao encontrar qualquer bloco corrompido")

	return cmd
}

// Check if mount point exists and is a directory

func verifyCmd() *cobra.Command {
	var original, restored string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verifica integridade bit-a-bit entre dois arquivos",
		Long:  `Compara SHA-256 de dois arquivos para confirmar fidelidade lossless.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if original == "" || restored == "" {
				return fmt.Errorf("flags --original e --restored são obrigatórias")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║          CROM VERIFY (Verificador)        ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Original: %-30s ║\n", original)
			fmt.Printf("║  Restored: %-30s ║\n", restored)
			fmt.Println("╚═══════════════════════════════════════════╝")

			origBytes, err := os.ReadFile(original)
			if err != nil {
				return fmt.Errorf("erro ao ler %s: %w", original, err)
			}
			restBytes, err := os.ReadFile(restored)
			if err != nil {
				return fmt.Errorf("erro ao ler %s: %w", restored, err)
			}

			origHash := sha256.Sum256(origBytes)
			restHash := sha256.Sum256(restBytes)

			if origHash != restHash {
				return fmt.Errorf("FALHA: SHA-256 divergem!\n  Original: %x\n  Restored: %x", origHash[:8], restHash[:8])
			}

			fmt.Println("✔ INTEGRIDADE CONFIRMADA: SHA-256 100% idênticos (fidelidade bit-a-bit)")
			return nil
		},
	}

	cmd.Flags().StringVar(&original, "original", "", "Caminho do arquivo original")
	cmd.Flags().StringVar(&restored, "restored", "", "Caminho do arquivo restaurado")

	return cmd
}

func infoCmd() *cobra.Command {
	var input, codebookPath, encryptionKey string
	var exportManifest bool
	var networkMode bool

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Exibe estatísticas detalhadas de um arquivo .crom (Format V2)",
		Long:  `Analisa o arquivo .crom e exibe header, block table, fragmentação, entropia dos blocos e distribuição de CodebookIDs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkMode {
				resp, err := http.Get("http://127.0.0.1:9099/info")
				if err != nil {
					return fmt.Errorf("não foi possível conectar ao daemon local. Ele está rodando? Erro: %v", err)
				}
				defer resp.Body.Close()
				bodyBytes, _ := io.ReadAll(resp.Body)
				fmt.Printf("\n═══ Status do Enxame (P2P) ═══\n")
				fmt.Printf("Resposta do Daemon: %s\n", string(bodyBytes))
				return nil
			}

			if input == "" {
				return fmt.Errorf("flag --input é obrigatória")
			}

			fmt.Println("╔═══════════════════════════════════════════╗")
			fmt.Println("║          CROM INFO (Analisador)           ║")
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Input:    %-30s ║\n", input)
			if encryptionKey != "" {
				fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			// Open and parse
			f, err := os.Open(input)
			if err != nil {
				return fmt.Errorf("erro ao abrir %s: %w", input, err)
			}
			defer f.Close()

			fileStat, err := f.Stat()
			if err != nil {
				return err
			}

			reader := format.NewReader(f)
			header, blockTable, entries, rStream, err := reader.ReadStream(encryptionKey)
			if err != nil {
				return fmt.Errorf("erro ao parsear formato: %w", err)
			}
			compDeltaPool, _ := io.ReadAll(rStream)

			// === Header ===
			fmt.Println("\n═══ Header ═══")
			fmt.Printf("  Version:       %d\n", header.Version)
			fmt.Printf("  Encrypted:     %v\n", header.IsEncrypted)
			fmt.Printf("  Original Size: %d bytes\n", header.OriginalSize)
			fmt.Printf("  Original Hash: %x\n", header.OriginalHash[:16])
			fmt.Printf("  Chunk Count:   %d\n", header.ChunkCount)
			fmt.Printf("  File Size:     %d bytes\n", fileStat.Size())

			// === Block Table ===
			if header.Version == format.Version2 && len(blockTable) > 0 {
				fmt.Printf("\n═══ Block Table (%d blocos) ═══\n", len(blockTable))

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

				// Fragmentation Ratio
				fragmentationRatio := float64(fileStat.Size()) / float64(header.OriginalSize)
				fmt.Printf("  Fragmentation:    %.4f (packed/original)\n", fragmentationRatio)
			}

			// === Shannon Entropy of compressed delta pool ===
			if len(compDeltaPool) > 0 {
				entropy := shannonEntropy(compDeltaPool)
				fmt.Printf("\n═══ Entropia ═══\n")
				fmt.Printf("  Shannon Entropy (Delta Pool): %.4f bits/byte\n", entropy)
				fmt.Printf("  Max Possible:                 8.0000 bits/byte\n")
				fmt.Printf("  Randomness:                   %.2f%%\n", entropy/8.0*100)
			}

			// === CodebookID Distribution (Top-10) ===
			if len(entries) > 0 {
				fmt.Printf("\n═══ Distribuição de CodebookIDs (Top-10) ═══\n")

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
				fmt.Printf("  ... %d CodebookIDs únicos no total\n", len(freq))
			}

			// === Manifest Export ===
			if exportManifest {
				if codebookPath == "" {
					return fmt.Errorf("--codebook é obrigatório para exportar manifesto")
				}

				manifest, err := cromsync.GenerateManifest(input, codebookPath, encryptionKey)
				if err != nil {
					return fmt.Errorf("erro ao gerar manifesto: %w", err)
				}

				jsonData, err := manifest.ToJSON()
				if err != nil {
					return fmt.Errorf("erro ao serializar manifesto: %w", err)
				}

				fmt.Printf("\n═══ ChunkManifest (JSON) ═══\n")
				fmt.Println(string(jsonData))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Caminho do arquivo .crom")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb) — necessário para --manifest")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Chave/Senha para descriptografia")
	cmd.Flags().BoolVar(&exportManifest, "export", false, "Exporta o ChunkManifest para o stdout no formato JSON (útil via pipe)")
	cmd.Flags().BoolVar(&networkMode, "network", false, "Exibe informações do nó na rede soberana")

	return cmd
}

func benchmarkCmd() *cobra.Command {
	var input, codebookPath, outputJson string
	var runs int

	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Executa benchmark completo de compressão e descompressão",
		Long:  `Executa N ciclos de Pack + Unpack + Verify usando os mesmos dados, emitindo métricas em JSON.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || codebookPath == "" {
				return fmt.Errorf("flags --input e --codebook são obrigatórias")
			}

			inputBytes, err := os.ReadFile(input)
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			inputEntropy, _, _ := entropy.Analyze(bytes.NewReader(inputBytes), len(inputBytes))

			type Output struct {
				InputFile     string  `json:"input_file"`
				InputSize     uint64  `json:"input_size"`
				InputEntropy  float64 `json:"input_entropy"`
				PackedSize    uint64  `json:"packed_size"`
				Ratio         float64 `json:"ratio"`
				HitRate       float64 `json:"hit_rate"`
				LiteralChunks int     `json:"literal_chunks"`
				TotalChunks   int     `json:"total_chunks"`
				AvgSimilarity float64 `json:"avg_similarity"`
				PackMs        int64   `json:"pack_ms"`
				UnpackMs      int64   `json:"unpack_ms"`
				Verify        string  `json:"verify"`
				Runs          int     `json:"runs"`
				Engine        string  `json:"engine"`
			}

			out := Output{
				InputFile:    input,
				InputSize:    uint64(len(inputBytes)),
				InputEntropy: inputEntropy,
				Runs:         runs,
				Engine:       "V4",
			}

			var totalPack, totalUnpack time.Duration
			var lastMetrics *cromlib.Metrics

			for i := 0; i < runs; i++ {
				cromFile, _ := os.CreateTemp("", "bench_crom_*")
				restoredFile, _ := os.CreateTemp("", "bench_restored_*")
				cromPath := cromFile.Name()
				restoredPath := restoredFile.Name()
				cromFile.Close()
				restoredFile.Close()

				metrics, err := cromlib.Pack(input, cromPath, codebookPath, cromlib.DefaultPackOptions())
				if err != nil {
					return fmt.Errorf("pack failed run %d: %v", i+1, err)
				}
				totalPack += metrics.Duration
				lastMetrics = metrics

				startUnpack := time.Now()
				err = cromlib.Unpack(cromPath, restoredPath, codebookPath, cromlib.DefaultUnpackOptions())
				if err != nil {
					return fmt.Errorf("unpack failed run %d: %v", i+1, err)
				}
				totalUnpack += time.Since(startUnpack)

				restoredBytes, _ := os.ReadFile(restoredPath)
				if bytes.Equal(inputBytes, restoredBytes) {
					out.Verify = "PASS"
				} else {
					out.Verify = "FAIL"
				}

				os.Remove(cromPath)
				os.Remove(restoredPath)
			}

			if lastMetrics != nil {
				out.PackedSize = lastMetrics.PackedSize
				out.Ratio = float64(out.PackedSize) / float64(out.InputSize) * 100
				out.HitRate = lastMetrics.HitRate
				out.LiteralChunks = lastMetrics.LiteralChunks
				out.TotalChunks = lastMetrics.TotalChunks
				out.AvgSimilarity = lastMetrics.AvgSimilarity
			}
			out.PackMs = totalPack.Milliseconds() / int64(runs)
			out.UnpackMs = totalUnpack.Milliseconds() / int64(runs)

			if out.Verify != "PASS" {
				return fmt.Errorf("verificação de integridade falhou")
			}

			jsonData, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}

			if outputJson != "" {
				if err := os.WriteFile(outputJson, jsonData, 0644); err != nil {
					return err
				}
				fmt.Printf("Benchmark salvo em %s\n", outputJson)
			} else {
				fmt.Println(string(jsonData))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Caminho do arquivo de entrada")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().StringVarP(&outputJson, "output-json", "o", "", "Caminho para salvar o JSON (opcional)")
	cmd.Flags().IntVarP(&runs, "runs", "r", 1, "Número de iterações do benchmark")

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

func grepCmd() *cobra.Command {
	var inputPath, codebookPath string

	cmd := &cobra.Command{
		Use:   "grep <query>",
		Short: "Grep O(1) diretamente nos Super-Tokens do arquivo .crom (Neural Search)",
		Long:  `Traduz uma query para seu ID BPE Numérico e varre a Matrix de Ocorrências ignorando a descompressão local. Pesquisa instantânea em qualquer volume de dados.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputPath == "" || codebookPath == "" {
				return fmt.Errorf("flags --input e --codebook são obrigatórias na busca Neural")
			}
			target := args[0]

			isCloud := len(inputPath) > 7 && (inputPath[:7] == "http://" || inputPath[:8] == "https://")

			fmt.Println("╔═══════════════════════════════════════════╗")
			if isCloud {
				fmt.Println("║   CROM SEARCH (Remote Neural Grep S3)    ║")
			} else {
				fmt.Println("║       CROM SEARCH (Grep Transparente)     ║")
			}
			fmt.Println("╠═══════════════════════════════════════════╣")
			fmt.Printf("║  Target:   %-30s ║\n", target)
			fmt.Printf("║  Input:    %-30s ║\n", inputPath)
			fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
			if isCloud {
				fmt.Printf("║  Mode:     HTTP Range (Zero-Download)     ║\n")
			}
			fmt.Println("╚═══════════════════════════════════════════╝")

			return cromlib.Grep(target, inputPath, codebookPath)
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Caminho do arquivo .crom alvo (aceita URLs HTTP/HTTPS para busca remota S3/CDN)")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (Dicionário de Referência)")

	return cmd
}
