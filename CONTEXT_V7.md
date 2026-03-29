# 🧠 Contexto: Arquitetura Crompressor (Versão 7)

Este documento centraliza o conhecimento atual ("Brain State") da Motor V7, que integra Chunking Dinâmico (FastCDC), Telemetria de SRE, e Roteamento baseado em Entropia.

## O que é o Crompressor
Compressor soberano escrito em **Go** que funciona como um- **Content-Defined Chunking (FastCDC)**: Quebra de arquivos baseada em Gear-Hash. Extremamente cobiçada para uso P2P, garantindo imunidade de shift.
- **Mixture of Experts / Shannon Entropy**: O motor afasta chunks incompressíveis avaliando entropia termodinâmica no fluxo, salvando CPU.
- **SRE Stack**: Monitoramento real-time via Prometheus + Grafana mapeado em Docker no `/monitoring`.
- **Façade SDK**: Compilado nativo exposto em `pkg/sdk` (`Compressor`, `PackCommand`, `UnpackCommand`).
- **Codebook P2P Handshake**: Sincronização inteligente com `CodebookHash`. Identifica dissonâncias antes da falha fatal.
- **Stream Pack/Unpack**: Compressão dinâmica ligada a Unix pipes `os.Stdin` via `PackStream`.
- **Merkle Tree V5**: Árvore hash atômica armazenada sobre blocos com tolerância a corrupção localizada e verificação `strict=true|false`.
- **Zero-Trust Encryption**: Suporte nativo ao AES-256-GCM.

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

## Fluxo de Compressão SDK (compiler.go)
```
Input File (ou Stream) → FastCDC Chunker (GearHash) → [chunk₁, chunk₂, ...] 
  → Para cada chunk (goroutine):
      Eval: Entropy > 7.8? → Literal (Salva CPU)
      Senão: LSH Search no Codebook → XOR(chunk, padrão) → residual
  → Agrupar kTable + DeltaPool → .crom
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

## Resultados Reais da Pesquisa V7

| Pesquisa | Dataset | Resultado |
|:---------|:--------|:----------|
| 01 — Logs JSON | 26.2MB, 200k linhas | **81.17%** economia vs Gzip (74%) |
| 02 — CDC Shift | 100KB mutado (byte int) | **99.85%** de blocos idênticos mantidos |
| 03 — VFS Mount | logs.crom | TTFB < 10ms |
| 04 — SRE | Grafana Dashboard / P2P | Métricas ativas |
| 05 — TCO | Projeção 1PB | $18.6K/mês economia |
| 06 — Imagens | 7 formatos, ~1700 testes | **1681/1783 PASS (94.2%)** |

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

## Problemas Conhecidos e Resoluções Históricas
1. **✅ Codebook não generalizava em imagens** → Data Augmentation no Treinamento (Resolvido).
2. **✅ Universal perdia para especialista em entropia densa** → Shannon Router + Passthrough bloqueou gargalo de CPU (Resolvido).
3. **✅ P2P EOF Race Condition** → Modulo Autenticador adaptado para ignorar loops mDNS no Node Sync Manual (Resolvido).
4. **✅ CDC Byte-Shifting Invalidação** → Implementado Gear-Hash FastCDC na V7 em vez do fixo de 128 bytes (Resolvido).

## Dependências Go
- `github.com/spf13/cobra` (CLI)
- `github.com/prometheus/client_golang` (Metrics)
- `github.com/schollz/progressbar/v3` (UI)
- Zstd via `github.com/klauspost/compress`

## Testes
```bash
make clean build && go test ./...  # Todos passam (exit 0)
```
