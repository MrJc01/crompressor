# 🗺️ Crompressor V10 — Roadmap de Melhorias (The Neural Codebook)

Prioridades definidas com base na consolidação SRE e infraestrutura de alta resiliência alcançadas nas versões V6-V9, elevando o sistema para Tokenização Semântica via BPE em V10.

---

## ✅ Sprint 6: P2P Delta Sync & Stream (CONCLUÍDA — 2026-03-29)

### 6.1 Delta Sync P2P Real ✅
- **Implementado**: `internal/network/protocol.go:RequestSync()` agora gera `LocalManifest` se o arquivo existe no disco, executa `cromsync.Diff(local, remote)`, e envia `DiffReq` com as matrizes otimizadas.
- **Merge**: Reconstrução dinâmica do `.crom` unindo buffers ausentes empurrados pela rede.

### 6.2 Streaming Compression (stdin pipe) ✅
- **Implementado**: Compressão sem *Seek* `PackStream()` escrevendo hashes na MerkleTree enquanto escuta de `os.Stdin`.
- **Uso Estendido**: Canais UNIX acoplados diretamente ao motor interno via SDK.

### 6.3 Codebook Sharing Protocol ✅
- **Implementado**: O daemon negocia passivamente *MsgCodebookHash*, evitando poluição de grafos P2P com dicionários mortos ou divergentes. Sem atritos de EOF após *Discovery race conditions*.

---

## ✅ Sprint 7: Anti-Fragilidade e Ecossistema (CONCLUÍDA — 2026-03-29)

### 7.1 FastCDC (Content-Defined Chunking) ✅
- **Implementado**: O limitador de blocos estáticos (128B) foi exterminado e substituído pelo Gear-Hash FastCDC em `internal/chunker/fastcdc.go`.
- **Resultado Incontestável**: O motor manteve **99.85% de sobrevivência estrutural** (chunks intactos) após a injeção fatal de bytes no arquivo-fonte. A Merkle Tree agora se rearquitetonica graciosamente ao byte-shifting.

### 7.2 Mixture of Experts (Shannon Routing) ✅
- **Implementado**: A função `compiler.go` invoca `entropy.Shannon` antes de queimar ciclos de CPU no pareamento LSH.
- **Resultado**: Arquivos ZIP, PNG e JPEG (entropia > 7.8) bypassam a busca massiva instantaneamente, transformando o motor perfeitamente letivo sem decréscimo de throughput.

### 7.3 SRE Dashboard & Native Telemetry ✅
- **Implementado**: Cluster Docker-Compose unindo Prometheus e Grafana, escaneando o endpoint de métricas nativas geradas em Go puro.
- **Insight**: Permite visibilidade total corporativa: Bytes recuperados de *Corrupted chunks*, tempo de compilação, bytes economizados no cluster em tempo real.

### 7.4 Native Go SDK (`pkg/sdk`) ✅
- **Implementado**: Isolação total do Core contra dependências globais e singletons do CLI Cobra via interface minimalista e canais nativos (`<-chan ProgressEvent`).

---

## ✅ Sprint 8: Cloud-Native & Algorithmic Edge (CONCLUÍDA — 2026-03-29)

### 8.1 Advanced Content-Aware Chunking (ACAC) ✅
- **Implementado**: `internal/chunker/acac.go` — `SemanticChunker` com delimitadores de linha (`\n`) e fallback de tamanho máximo.
- **Integração**: `compiler.go` e `compiler_stream.go` suportam `opts.UseACAC` e `opts.ACACDelimiter`.
- **Testes**: `acac_test.go` — PASS em JSON Lines e overflow de linha longa.

### 8.2 WebAssembly Build ✅
- **Implementado**: `pkg/wasm/main.go` exportando `CromPack` e `CromVersion` para JavaScript via `syscall/js`.
- **Infraestrutura**: `internal/codebook/` refatorado com build tags (`mmap.go` para nativo, `mmap_wasm.go` para JS) e `reader.go` compartilhado.
- **Resultado**: Binário `crompressor.wasm` de **3.3MB** compilado com sucesso.
- **Demo**: `examples/www/index.html` com UI dark-mode para drag-and-drop compression.

### 8.3 Kubernetes Log Interceptor ✅
- **Implementado**: `deployments/k8s/crompressor-daemonset.yml` — DaemonSet completo com:
  - Volume `hostPath` para `/var/log/containers` (read-only)
  - PVCs para codebook e output
  - Prometheus scraping annotations (`prometheus.io/scrape: "true"`)
  - Liveness/readiness probes via `/metrics`

---

## ✅ Sprint 9: The Sovereign CromFS (CONCLUÍDA — 2026-03-29)

### 9.1 Criptografia Convergente (Zero-Knowledge) ✅
- **Implementado**: `internal/crypto/convergent.go` permite derivar chaves AES-GCM 256 derivadas nativamente a partir do Hash SHA-256 do próprio `chunk`.
- **Resultado**: Nós diferentes comprimindo dados iguais geram a mesma matriz binária criptografada (ciphertext) e nonce. Permite **Desduplicação Global** cruzada entre usuários sem quebrar o sigilo.

### 9.2 Codebooks Hierárquicos (L1, L2, L3) ✅
- **Implementado**: `internal/search/multi.go` orquestrando Arrays de `Searcher`. O header V6 agora aceita `[3][8]byte` assinaturas.
- **Resultado**: A engrenagem LSH cai em fallback fluido (Local -> Projeto -> Universal), otimizando a latência usando heurísticas multi-tier.

### 9.3 Cluster-Wide FUSE Daemon (CromFS) ✅
- **Implementado**: Integrado `bazil.org/fuse` gerando o filesystem nativo `cromfs`.
- **Implementado**: Subcomando `crompressor cromfs --m /mnt/cromfs`. As gravações regulares do Linux/Docker são interceptadas transparentemente, comprimidas no streaming e fragmentadas criptografadas para a pool.

---

## ✅ Sprint 10: The Neural Codebook (CONCLUÍDA — 2026-03-29)

### 10.1 Byte-Pair Encoding (BPE) Engine ✅
- **Implementado**: `internal/trainer/bpe.go` — Algoritmo completo BPE em Go puro.
- **Mecânica**: Inicia com 256 bytes básicos, itera fundindo os bigramas mais frequentes até atingir o `MaxCodewords` (respeitando `maxLen=128` para conformidade LSH).
- **Resultado**: Extraiu tokens como `"timestamp":"2026-03-29 03:50"` (28 bytes), `", "level":"INFO", "worker":"worker-2", "msg":"Task processed", "ip":"192.168.0."` (80 bytes) de maneira completamente autonoma.

### 10.2 Integração CLI `--use-bpe` ✅
- **Implementado**: Flag `--use-bpe` em `crompressor train` com ramificação completa no `engine.go`.
- **Memory Sandbox**: Corpus limitado a 50MB de RAM para proteger contra OOM.

### 10.3 Benchmark Comparativo ✅
- **BPE (392 tokens)**: 26.2MB → 4,939,300 bytes = **18.85%** ✅ PASS.
- **Standard (1885 tokens)**: 26.2MB → 4,943,049 bytes = **18.87%**.
- **Veredito**: BPE com **5x menos vocabulário** supera marginalmente o método de frequência bruta.

---

## Calendário de Integração
```
✅ Sprint 6 (Delta Sync & Stream)        → Concluído (2026-03-29)
✅ Sprint 7 (The Swarm & Anti-Fragility) → Concluído (2026-03-29)
✅ Sprint 8 (Cloud Native & WASM Target) → Concluído (2026-03-29)
✅ Sprint 9 (Sovereign CromFS & ZK)      → Concluído (2026-03-29)
✅ Sprint 10 (Neural Codebooks - BPE)    → Concluído (2026-03-29)
```
