# 🧠 Crompressor V5 — Contexto Completo para Continuação

## O que é o Crompressor
Compressor soberano escrito em **Go** que funciona como um **compilador de padrões**:
1. `train` — Treina um codebook (`.cromdb`) com padrões extraídos de um dataset
2. `pack` — Comprime um arquivo usando XOR contra padrões do codebook → gera `.crom`
3. `unpack` — Descomprime restaurando o original bit-a-bit (lossless SHA-256)
4. `mount` — Monta o `.crom` como filesystem virtual (FUSE) com acesso aleatório O(1)
5. `daemon` — Nó P2P mesh que sincroniza `.crom` entre peers

## Arquitetura de Diretórios
crompressor/
├── cmd/crompressor/main.go    # CLI + Daemon P2P (cobra)
├── internal/
│   ├── autobrain/             # V5: Detector de formato + Router de codebooks
│   │   ├── detector.go        #   Magic Bytes + Entropia Shannon → categoria
│   │   └── router.go          #   Categoria → melhor .cromdb disponível
│   ├── chunker/               # Content-Defined Chunking (CDC)
│   ├── codebook/              # Leitor de .cromdb (hash table de padrões)
│   ├── crypto/                # AES-256-GCM encrypt/decrypt
│   ├── delta/                 # XOR residual + Zstd compress/decompress
│   ├── entropy/               # Análise de Entropia Shannon
│   ├── merkle/                # V5: Merkle Tree (BuildFromHashes, Diff)
│   ├── metrics/               # V5: Prometheus counters + histograms
│   ├── network/               # LibP2P node + mDNS + P2P Delta Sync + Codebook Sharing
│   │   ├── protocol.go        #   V6: MsgCodebookHash, MsgCodebookReq, Delta Diff
│   │   └── bitswap.go         #   V6: ReceiveChunks com merge local+remoto
│   ├── search/                # LSH + Linear search + Restrict() (Multi-Pass)
│   ├── trainer/               # Engine de treinamento + Data Augmentation (V5.3)
│   │   ├── engine.go          #   Pipeline com AugmentPatterns integrado
│   │   └── augmentation.go    #   V5.3: Bit shifts e rotações circulares
│   └── vfs/                   # FUSE mount + RandomReader com BlockCache
├── pkg/
│   ├── cromlib/               # Pack() e Unpack() — motor principal
│   │   ├── compiler.go        #   Loop de compressão (goroutines paralelas + Multi-Pass)
│   │   ├── compiler_stream.go #   V6.2: PackStream() — compressão via io.Reader (pipes)
│   │   ├── unpacker.go        #   Loop de descompressão (tolerant mode)
│   │   └── compiler_test.go   #   Suite completa de roundtrip tests
│   ├── format/                # Header V1-V5 + ChunkTable serialization
│   ├── sdk/                   # SDK público (thin wrapper)
│   └── sync/                  # Sync primitives + ChunkManifest + Diff
├── pesquisa/                  # 7 pesquisas com datasets e scripts
│   ├── Manifesto.md           # Checklist de 14 pontos de auditoria V5
│   ├── scripts/run_benchmarks.sh
│   ├── 06-image_format_analysis/  # 1223 testes de imagem
│   └── 07-benchmark_comparativo/  # V5.2: Gzip vs Zstd vs Crompressor
└── Makefile
```

## Header V5 (112 bytes)
```
Offset  Size  Campo
0       4     Magic ("CROM")
4       2     Version (5)
6       1     IsEncrypted
7       1     IsPassthrough
8       16    Salt
24      32    OriginalHash (SHA-256)
56      8     OriginalSize
64      4     ChunkCount
68      4     ChunkSize
72      8     CodebookHash
80      32    MerkleRoot ← NOVO V5
```

## Fluxo de Compressão (compiler.go)
```
Input File → CDC Chunker → [chunk₁, chunk₂, ...] 
  → [Multi-Pass?] Pass 1: Tally CodebookIDs → Restrict(Top-256)
  → Para cada chunk (goroutine):
      LSH Search no Codebook → melhor padrão
      XOR(chunk, padrão) → residual (delta)
  → Agrupar deltas em blocos de 131072 chunks
  → Zstd compress cada bloco
  → SHA-256 de cada bloco → Merkle Tree
  → Header V5 + BlockTable + ChunkTable + DeltaPool → .crom
```

## Fluxo de Streaming (compiler_stream.go)
```
io.Reader (stdin/pipe) → CDC Chunker → [chunks]
  → LSH Search + XOR → deltas
  → Compress blocos → Buffer /tmp/
  → EOF → Assemble Header + Tables + Pool → .crom
```

## Fluxo P2P Delta Sync (protocol.go)
```
Node B → Envia CodebookHash → Node A responde Hash
  → Se Mismatch: B baixa .cromdb de A
  → B envia SyncReq(filename)
  → A responde com Manifest
  → B gera LocalManifest (se arquivo parcial existe)
  → B executa Diff(local, remote) → missingIndices
  → B envia DiffReq(missingIndices)
  → A envia apenas os chunks faltantes
  → B mescla chunks locais + remotos → .crom final
```

## Resultados Reais da Pesquisa V5

| Pesquisa | Dataset | Resultado |
|:---------|:--------|:----------|
| 01 — Logs JSON | 26.2MB, 200k linhas | **81.17%** economia, 1089ms |
| 02 — Delta SQL | 5.7MB dump | **76.4%** economia |
| 03 — VFS Mount | logs.crom | TTFB < 10ms |
| 04 — P2P | Node mesh | Identidade validada |
| 05 — TCO | Projeção 1PB | $18.6K/mês economia |
| 06 — Imagens | 7 formatos, 1503 testes | **1401/1503 PASS (93.2%)** |

### Economia por Formato de Imagem (same-brain)
| Formato | Economia | Observação |
|:--------|:---------|:-----------|
| SVG | 34.73% | Texto XML — melhor caso |
| BMP | 21.82% | Raw pixels — bom |
| TIFF | 20.32% | Raw pixels — bom |
| JPG | 8.28% | DCT lossy — marginal |
| PNG | 0.00% | Deflate — passthrough |
| WebP | 0.00% | VP8L — passthrough |
| GIF | -0.45% | LZW — passthrough |

## Problemas Conhecidos e Resoluções
1. **✅ Codebook não generaliza** — degradacao de 32.71% em dados novos → **RESOLVIDO**: Data Augmentation (Sprint 5.3)
2. **Universal perde para especialista SEMPRE** — valida arquitetura Mixture of Experts
3. **Dados pré-comprimidos (PNG/WebP/GIF) são imunes** — XOR não funciona em alta entropia
4. **Overhead Merkle é desprezível** (+0.23% tamanho) mas custo CPU real (+45% tempo)
5. **✅ P2P não sincronizava com Codebooks diferentes** → **RESOLVIDO**: Codebook Sharing Protocol (Sprint 6.3)
6. **✅ Pack não suportava pipes/stdin** → **RESOLVIDO**: PackStream (Sprint 6.2)

## Dependências Go
- `github.com/spf13/cobra` (CLI)
- `github.com/prometheus/client_golang` (Metrics)
- `github.com/schollz/progressbar/v3` (UI)
- Zstd via `github.com/klauspost/compress`

## Testes
```bash
make clean build && go test ./...  # Todos passam (exit 0)
```
