# 🗺️ Crompressor V6 — Roadmap de Melhorias

Prioridades definidas com base nas pesquisas reais V5 (1223 testes de imagem + benchmarks 01-05).

---

## ✅ Sprint 5: Performance Core (CONCLUÍDA — 2026-03-29)

### 5.1 Multi-Pass Compression ✅
- **Implementado**: Flag `--multi-pass` no `crompressor pack`. Two-pass Tallying restringe CodebookIDs aos Top-256.
- **Onde**: `pkg/cromlib/compiler.go` + `internal/search/lsh.go:Restrict()` + `internal/search/linear.go:Restrict()`
- **Resultado**: -4KB a -10KB adicionais por arquivo versus single-pass.

### 5.2 Benchmark Comparativo vs Gzip/Zstd ✅
- **Implementado**: Script `pesquisa/07-benchmark_comparativo/run_benchmark.sh`
- **Resultado Real**: Crompressor mantém 81% saving em logs JSON (26MB), 76% em SQL dumps (5.7MB). Gzip/Zstd vencem em ratio puro (95-99%), mas não oferecem VFS Mount, P2P Sync ou Merkle Tree.

### 5.3 Data Augmentation no Treino ✅
- **Implementado**: `internal/trainer/augmentation.go` + flag `--augment` no `crompressor train`.
- **Mecânica**: Bit shifts (Left/Right) e Rotações Circulares nos padrões elites para combater overfitting.

---

## ✅ Sprint 6: P2P Delta Sync & Stream (CONCLUÍDA — 2026-03-29)

### 6.1 Delta Sync P2P Real ✅
- **Implementado**: `internal/network/protocol.go:RequestSync()` agora gera `LocalManifest` se o arquivo existe no disco, executa `cromsync.Diff(local, remote)`, e envia `DiffReq` apenas com os indices faltantes.
- **Merge**: `internal/network/bitswap.go:ReceiveChunks()` reconstrói o `.crom` final mesclando chunks locais + remotos.

### 6.2 Streaming Compression (stdin pipe) ✅
- **Implementado**: `pkg/cromlib/compiler_stream.go:PackStream()` — lê de `io.Reader` sem Seek. Buffering via `/tmp`.
- **CLI**: `crompressor pack --stream -i - -c dict.cromdb -o out.crom`
- **Uso**: `tail -f /var/log/syslog | crompressor pack --stream -c dict.cromdb -o live.crom`

### 6.3 Codebook Sharing Protocol ✅
- **Implementado**: Novos message types `MsgCodebookHash (0x05)`, `MsgCodebookReq (0x06)`, `MsgCodebookData (0x07)`.
- **Mecânica**: O `RequestSync` agora inicia com Handshake de Hash, e se divergir, baixa silenciosamente o `.cromdb` remoto antes de continuar.

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

## Ordem de Execução
```
✅ Sprint 5.2 (Benchmark vs Gzip/Zstd) → Concluído (2026-03-29)
✅ Sprint 5.1 (Multi-Pass Compression)  → Concluído (2026-03-29)
✅ Sprint 5.3 (Data Augmentation)       → Concluído (2026-03-29)
✅ Sprint 6.1 (Delta Sync P2P Real)     → Concluído (2026-03-29)
✅ Sprint 6.2 (Streaming Compression)   → Concluído (2026-03-29)
✅ Sprint 6.3 (Codebook Sharing)        → Concluído (2026-03-29)
⬜ Sprint 7   (Ecossistema)             → Próxima Sprint
```
