# 🗺️ Crompressor V6 — Roadmap de Melhorias

Prioridades definidas com base nas pesquisas reais V5 (1223 testes de imagem + benchmarks 01-05).

---

## 🔴 Sprint 5: Performance Core (Prioridade Alta)

### 5.1 Multi-Pass Compression
- **O quê**: Primeiro pass coleta histograma de CodebookIDs usados. Segundo pass recodifica usando apenas Top-K padrões mais frequentes.
- **Por quê**: Menos vocabulário = menor entropia na Delta Pool = Zstd comprime melhor. Estimativa: +10-20% de economia.
- **Onde**: `pkg/cromlib/compiler.go` — novo modo `opts.MultiPass = true`
- **Teste**: Comparar ratio single-pass vs multi-pass no dataset de logs 26MB.

### 5.2 Benchmark Comparativo vs Gzip/Zstd
- **O quê**: Script automatizado que comprime os mesmos datasets com `gzip -9`, `zstd -19`, e `crompressor pack`, medindo ratio, tempo, e RAM.
- **Por quê**: Sem esse comparativo, não temos argumento quantificável de superioridade.
- **Onde**: `pesquisa/07-benchmark_comparativo/` — novo diretório com scripts bash.
- **Datasets**: logs_200k.json (26MB), dump_v1.sql (5.7MB), imagens BMP/SVG.

### 5.3 Data Augmentation no Treino
- **O quê**: Durante `train`, aplicar perturbações (bit shifts, byte rotation, padding) nos padrões antes de selecionar os elites.
- **Por quê**: O codebook atual é um "memorizer" (degradação 32% em dados novos). Com augmentation, capturaria padrões mais robustos.
- **Onde**: `internal/trainer/engine.go` — novo pipeline de augmentation pré-elite-selection.
- **Teste**: Re-executar Pesquisa 06 (inference test) e medir se degradação cai abaixo de 15%.

---

## 🟡 Sprint 6: P2P Delta Sync (Prioridade Média)

### 6.1 Protocolo de Negociação Merkle
- **O quê**: Dois peers trocam MerkleRoot → se difere, trocam hashes das folhas → `Diff()` identifica blocos divergentes → transferem apenas esses blocos.
- **Onde**: `internal/network/` — novo `merkle_sync.go` com handler gRPC/WebSocket.
- **Dependência**: MerkleRoot já está no Header V5 (implementado).

### 6.2 Streaming Compression (stdin pipe)
- **O quê**: Modo que comprime dados de stdin progressivamente, emitindo blocos sem precisar do arquivo completo.
- **Por quê**: Para logs em tempo real (`tail -f | crompressor pack --stream`).
- **Onde**: `pkg/cromlib/compiler.go` — novo `PackStream(reader io.Reader, ...)`.

### 6.3 Codebook Sharing Protocol
- **O quê**: Handshake P2P que negocia codebook comum entre peers, ou transfere o `.cromdb` junto com os dados.
- **Por quê**: Hoje dois peers com codebooks diferentes não conseguem trocar `.crom`.
- **Onde**: `internal/network/` — extensão do protocol de sync.

---

## 🟢 Sprint 7: Ecossistema (Prioridade Baixa)

### 7.1 Dashboard Grafana JSON
- Entregar um `monitoring/grafana_dashboard.json` pronto para importar.
- Panels: BytesSaved, PackOps, UnpackOps, PackDuration histogram, CorruptBlocks.

### 7.2 SDK Library Mode
- `cromlib.PackBytes(data []byte, codebook []byte) ([]byte, error)` — sem tocar em disco.
- Abre integração programática com aplicações Go.

### 7.3 WASM Build
- Compilar o motor para WebAssembly para compressão/descompressão no browser.

### 7.4 Chunk Size Adaptativo Intra-Arquivo
- Detectar regiões de baixa/alta entropia dentro do MESMO arquivo e usar chunk sizes diferentes por região.
- Melhoraria logs que contêm tokens JWT inline (alta entropia dentro de texto limpo).

---

## Ordem de Execução Recomendada
```
Sprint 5.2 (Benchmark vs Gzip/Zstd) → Baseline comparativo
Sprint 5.1 (Multi-Pass Compression)  → Melhorar ratio
Sprint 5.3 (Data Augmentation)       → Melhorar generalização
Sprint 6.1 (Delta Sync P2P Real)     → Feature killer
```
