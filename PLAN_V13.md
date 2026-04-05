# 🚀 Roadmap Estratégico Crompressor V13: "Neural Routing & Decentralized Caching"

**Objetivo Central**: Desidratar dados usando dicionários universais (Cérebros) imutáveis compartilhados via DHT, garantindo total soberania (Zero-Trust) e armazenamento frio ultra-barato sem penalizar performance de CPU em payloads compressados nativamente.

## 🧠 Fases de Engenharia e Checklist (V13)

### Fase 1: O Engine Adaptativo (Multi-Brain & Bypass)
- `[x]` **Tarefa 1.1 (Format V7 Multi-Brain)**: Refatorar `pkg/format/format.go` (Header). Alterar amarração de 1 único cérebro `[32]byte` no cabeçalho por um *Array de Codebooks* e criar métricas backwards-compatíveis com o V6 para rotear via `CodebookIndex uint8` na Chunk Table.
- `[x]` **Tarefa 1.2 (Heurística Zero-Overhead)**: Em `pkg/cromlib/entropy.go`, adicionar o `DetectHeuristicBypass()` via Scanner de Magic Numbers (JPEG, GZIP, ELF) em conjunto com Entropia de Shannon (V12). Ignorar LSH/BPE instantaneamente se o MimeType for opaco, encapando diretamente em Zstd bruto para evitar inflação inútil.
- `[x]` **Tarefa 1.3 (Redirecionamento de Fluxos)**: Adaptar o loop do `compiler.go` para usar o cérebro A para `txt`, pular para *Bypass* no bloco de `jpeg`, otimizando a montagem e escrevendo o `CodebookIndex` correto na *Chunk Entry*. Teste end-to-end com múltiplos Codebooks.

### Fase 2: Egress Cloud Optimizer (VFS-Cache LRU)
- `[x]` **Tarefa 2.1 (LRU Mount Cache)**: Integrar LRU RAM-Cache (ex: `hashicorp/golang-lru`) no `internal/remote/cloud.go`. O FUSE HTTP Range consumirá de volta blocos adjacentes num raio restrito (ex: limite de mem max `~64MB`), matando qualquer chamadas HTTP faturáveis redundantes da Nuvem.
- `[x]` **Tarefa 2.2 (Async Prefetcher)**: O VFS identifica padrão de leitura linear e recarrega os próximos offsets via Get Range Assíncrono para suavizar a UX da ponte de rede.

### Fase 3: P2P Local Mesh Data (Zero Cloud Fetch)
- `[x]` **Tarefa 3.1 (Gossip Block Available)**: Evolutir GossipSub P2P. Um nó compartilha o Manifesto (`Codebook` hash e File Hash) + CIDs decodificados no seu disco RAM/SSD. Ao montar um VFS FUSE, Nodes pedem primeiro o bloco na LAN interna do cluster; se o colega ter os bytes cacheados, executa Bitswap P2P sem nunca tocar no servidor AWS remotamente. Em caso de *Miss*, faz fallback para o `CloudReader`. 

---

### Principais Riscos Levantados
1. **Regressão de Arquitetura V6**: Adicionar IDs indexadas quebra nativamente Header V6. A lógica deve herdar as Flags V7.
2. **LRU Cache Trashing**: Se não for alocado via Hard Limit (em MiB configurável, default 32MiB), Mounts extensivos em Data Lakes causarão `OOM Killing` do container FUSE P2P.
3. **P2P Broadcast Storm**: Advertências excessivas de "Chucks disponíveis em cache" destruirão roteamento. Propagação estrita requerida.
