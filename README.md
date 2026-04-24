# Crompressor

> Motor de compressão baseado em dicionário para dados estruturados, tensores e pesos de LLMs — escrito em Go.

[![Build](https://github.com/MrJc01/crompressor/actions/workflows/ci.yml/badge.svg)](https://github.com/MrJc01/crompressor/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/MrJc01/crompressor)](https://goreportcard.com/report/github.com/MrJc01/crompressor)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## O que é o Crompressor?

O Crompressor é um motor de compressão construído sobre três primitivas compostas:

- **CDC** — Content-Defined Chunking via Rabin Fingerprint. Divide a entrada em blocos semânticos de tamanho variável, permitindo deduplicação entre arquivos estruturalmente similares.
- **VQ** — Vector Quantization. Mapeia cada chunk para a entrada mais próxima em um codebook pré-treinado usando `argmin_k ‖x - eₖ‖²`. Lookup é O(1) após treinamento.
- **XOR Delta** — Armazena o resíduo `Δᵢ = original XOR reconstruído` para reconstrução lossless quando necessário.

O motor opera em dois modos distintos:

| Modo | Descrição | Compressão | Fidelidade |
|------|-----------|------------|------------|
| **Edge** | Lossy. Descarta o delta XOR. Lookup rápido, armazenamento mínimo. | ~8x em pesos de LLM | MSE ~2.55 |
| **Archive** | Lossless. Armazena o delta XOR. SHA-256 do output = SHA-256 do input. | ~1.5–2.5x em tensores densos | Bit-exact |

Esses modos são mutuamente exclusivos por design. Escolher Edge significa aceitar erro de quantização (como GPTQ/AWQ). Escolher Archive significa obedecer aos limites de entropia de Shannon sobre o resíduo.

---

## Instalação

```bash
git clone https://github.com/MrJc01/crompressor
cd crompressor
make build
# Binário em ./bin/crompressor
```

---

## Quick Start

```bash
# Treinar um codebook a partir de dados de exemplo
crompressor train -i ./training_data/ -o codebook.cromdb --size 4096

# Comprimir em modo Edge (lossy, rápido)
crompressor pack -i dados.bin -o dados.crom --codebook codebook.cromdb --mode edge

# Comprimir em modo Archive (lossless)
crompressor pack -i dados.bin -o dados.crom --codebook codebook.cromdb --mode archive

# Descomprimir
crompressor unpack -i dados.crom -o restaurado.bin --codebook codebook.cromdb

# Verificar round-trip lossless (somente modo Archive)
crompressor verify --original dados.bin --restored restaurado.bin

# Inspecionar um arquivo .crom
crompressor info -i dados.crom
```

---

## Como funciona

**A equação central:**

```
CROM(X) = Σᵢ C[q(chunkᵢ(X))] ⊕ Δᵢ
```

Onde:
- `chunkᵢ(X)` — o i-ésimo bloco semântico do CDC
- `q(·)` — quantização vetorial: entrada mais próxima no codebook
- `C[·]` — lookup O(1) no codebook
- `Δᵢ` — resíduo XOR (armazenado no Archive, descartado no Edge)
- `⊕` — composição XOR

---

## Estrutura do Repositório

```
crompressor/
├── cmd/crompressor/        ← CLI (pack, unpack, train, verify, info, grep, benchmark)
├── pkg/
│   ├── cromlib/            ← Motor de compressão (Pack, Unpack, AutoPack)
│   ├── format/             ← Serialização do formato .crom v8
│   └── cromdb/             ← Leitura de codebooks
├── internal/               ← Implementação interna (chunker, codebook, delta, entropy, etc.)
├── examples/               ← Exemplos Go (edge_mode, archive_mode, codebook_train)
├── docs/                   ← Documentação técnica (architecture, modes, benchmarks)
├── testdata/               ← Fixtures para testes
├── deployments/k8s/        ← Manifests Kubernetes
├── monitoring/             ← Prometheus + Grafana
├── Makefile
├── LICENSE                 ← MIT
└── README.md
```

---

## Makefile

```bash
make build      # Compila ./bin/crompressor
make test       # Roda todos os testes com -race
make bench      # Roda benchmarks de performance
make lint       # go vet
make clean      # Remove ./bin/
make demo       # Round-trip completo: gerar → comprimir → descomprimir → verificar
```

---

## Documentação

| Documento | Descrição |
|-----------|-----------|
| [Arquitetura](docs/architecture.md) | CDC + VQ + XOR: decisões de design |
| [Modos de Operação](docs/modes.md) | Edge vs Archive: quando usar cada um |
| [Benchmarks](docs/benchmarks.md) | Metodologia e resultados |
| [Pesquisa](docs/research/) | Links para estudo matemático |

---

## Fundamentos Matemáticos

O modelo de compressão, metodologia de benchmark e análise de rate-distortion estão em um repositório separado:

→ [MrJc01/crompressor-matematica](https://github.com/MrJc01/crompressor-matematica)

---

## Ecossistema

| Repositório | Papel |
|-------------|-------|
| [crompressor](https://github.com/MrJc01/crompressor) | Motor core (este repo) |
| [crompressor-gui](https://github.com/MrJc01/crompressor-gui) | Interface gráfica nativa |
| [crompressor-matematica](https://github.com/MrJc01/crompressor-matematica) | Estudo matemático e benchmarks |
| [crompressor-neuronio](https://github.com/MrJc01/crompressor-neuronio) | Pesquisa neural |
| [crompressor-security](https://github.com/MrJc01/crompressor-security) | Camada de segurança |
| [crompressor-sinapse](https://github.com/MrJc01/crompressor-sinapse) | Transporte P2P |
| [crompressor-video](https://github.com/MrJc01/crompressor-video) | Codec de vídeo |

---

## Licença

MIT — veja [LICENSE](LICENSE).
