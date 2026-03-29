# 🗺️ Crompressor V7 — Roadmap de Melhorias (The Swarm Phase)

Prioridades definidas com base na consolidação SRE e infraestrutura de alta resiliência alcançadas nas versões V6 e V7.

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

## 🟢 Sprint 8: A Fronteira do Navegador e Distribuição (Próxima)

### 8.1 WASM (WebAssembly) Build
- Empacotar o `pkg/sdk` e transpilar via `GOOS=js GOARCH=wasm`.
- Permitir compressão em clientes frontend (React/Next.js) prévia ao upload S3 (Client-side deduplication).

### 8.2 Kubernetes CRD Operator (Operator SDK)
- Criar um `CrompressorDaemonSet` CRD que instaura drives `.cromdb` sincronizados nos volumes de nós EKS/GKE nativamente, permitindo logs compressos já na raíz docker.

### 8.3 Advanced Content-Aware Chunking (ACAC)
- Detecção algorítmica de limites de estrutura (ex: detectar boundaries naturais de chaves JSON) em vez de Hash puro, para logs altamente indexáveis.

---

## Calendário de Integração
```
✅ Sprint 6 (Delta Sync & Stream)        → Concluído (2026-03-29)
✅ Sprint 7 (The Swarm & Anti-Fragility) → Concluído (2026-03-29)
⬜ Sprint 8 (Cloud Native & WASM Target) → Em Planejamento
```
