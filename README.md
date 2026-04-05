<p align="center">
  <h1 align="center">🧬 Crompressor</h1>
  <p align="center"><strong>Semantic Compression Engine for Go</strong></p>
  <p align="center">
    <a href="https://pkg.go.dev/github.com/MrJc01/crompressor"><img src="https://pkg.go.dev/badge/github.com/MrJc01/crompressor.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/MrJc01/crompressor"><img src="https://goreportcard.com/badge/github.com/MrJc01/crompressor" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
  </p>
</p>

---

**Crompressor** is a high-performance, lossless compression library written in Go. It combines semantic extraction via LSH B-Tree indexing, cosine-similarity search (HNSW), and a trainable codebook to achieve aggressive compression ratios — especially on structured data like source code, logs, JSON, config files, and AI model tensors.

> **PT-BR:** O Crompressor é um motor de compressão semântica lossless escrito em Go. Ele combina extração semântica via B-Tree LSH, busca cosenoidal HNSW e um codebook treinável para atingir razões de compressão agressivas em dados estruturados.

## Features

- 🧠 **Trainable Codebook** — Build domain-specific dictionaries from your own data
- ⚡ **LSH B-Tree O(1) Lookup** — Near-constant-time pattern matching
- 🔒 **Lossless Integrity** — Bit-perfect reconstruction with Merkle tree verification
- 📦 **`.crom` Format** — Compact, streamable binary format with built-in metadata
- 🖥️ **VFS / FUSE Support** — Mount compressed archives as virtual filesystems
- 🌐 **P2P Sync** — Kademlia/LibP2P mesh for distributed codebook sharing
- 🔐 **Post-Quantum Crypto** — ChaCha20-Poly1305 + Dilithium-inspired signatures
- 🏗️ **WASM Build** — Run the compressor in the browser

## Installation

```bash
go get github.com/MrJc01/crompressor@latest
```

### Building from source

```bash
git clone https://github.com/MrJc01/crompressor.git
cd crompressor
make build
```

The binary will be placed at `./bin/crompressor`.

**Requirements:** Go 1.22+ and Make.

## Quick Start

### CLI Usage

```bash
# Train a codebook from your data
./bin/crompressor train --input ./my-data/ --output codebook.cromdb --size 8192

# Compress a file
./bin/crompressor pack --input data.bin --output data.crom --codebook codebook.cromdb

# Decompress
./bin/crompressor unpack --input data.crom --output restored.bin --codebook codebook.cromdb

# Verify bit-perfect integrity
./bin/crompressor verify --original data.bin --restored restored.bin
```

### Go API

```go
package main

import (
    "fmt"
    "github.com/MrJc01/crompressor/pkg/sdk"
)

func main() {
    c := sdk.NewCompressor()

    // Pack
    err := c.Pack("input.bin", "output.crom", "codebook.cromdb")
    if err != nil {
        panic(err)
    }

    // Unpack
    err = c.Unpack("output.crom", "restored.bin", "codebook.cromdb")
    if err != nil {
        panic(err)
    }

    fmt.Println("Done — lossless compression verified.")
}
```

## Project Structure

```
crompressor/
├── cmd/crompressor/     # CLI binary (pack, unpack, verify, train, daemon)
├── pkg/                 # Public API
│   ├── cromdb/          # Codebook database engine
│   ├── cromlib/         # Core compiler & unpacker
│   ├── format/          # .crom binary format (reader/writer)
│   ├── sdk/             # High-level SDK (vault, compressor, crypto)
│   ├── sync/            # Manifest-based sync
│   └── wasm/            # WebAssembly entry point
├── internal/            # Internal packages
│   ├── chunker/         # Content-defined chunking (CDC)
│   ├── codebook/        # Codebook builder & LSH indexing
│   ├── entropy/         # Shannon entropy analysis & bypass
│   ├── fractal/         # Fractal pattern generator
│   ├── merkle/          # Merkle tree integrity
│   ├── search/          # HNSW cosine similarity engine
│   ├── vfs/             # Virtual filesystem & FUSE mount
│   └── ...              # delta, crypto, metrics, network, etc.
├── docs/                # Technical documentation (10 chapters)
├── examples/            # Usage examples
├── scripts/             # Codebook generation helpers
├── go.mod
└── LICENSE              # MIT
```

## Make Targets

| Command | Description |
|---|---|
| `make build` | Build the CLI binary to `./bin/crompressor` |
| `make test` | Run all tests with race detection |
| `make bench` | Run benchmarks |
| `make lint` | Run `go vet` |
| `make clean` | Remove build artifacts |

## Documentation

Detailed technical documentation is available in the [`docs/`](docs/) directory:

1. [Concept & Vision](docs/01-CONCEITO_E_VISAO.md)
2. [System Architecture](docs/02-ARQUITETURA_DO_SISTEMA.md)
3. [Dictionary Structure](docs/03-ESTRUTURA_DO_DICIONARIO.md)
4. [Compiler Specification](docs/04-ESPECIFICACAO_DO_COMPILADOR.md)
5. [Refinement Layer](docs/05-CAMADA_DE_REFINAMENTO.md)
6. [Tech Stack](docs/06-TECH_STACK.md)
7. [Security & Sovereignty](docs/07-SEGURANCA_E_SOBERANIA.md)
8. [Advanced Use Cases](docs/08-CASOS_DE_USO_AVANCADOS.md)
9. [Benchmarks & Metrics](docs/09-BENCHMARKS_E_METRICAS.md)
10. [MVP Strategy](docs/10-ESTRATEGIA_MVP.md)

## Benchmark Results

Real benchmark results from the automated test suite (`go run ./benchmark/` on the [`benchmark`](https://github.com/MrJc01/crompressor/tree/benchmark) branch):

| Dataset | Type | Original | Packed | Ratio | Pack Speed | Unpack Speed | Integrity |
|---|---|---|---|---|---|---|---|
| go_source | Repetitive Go code | 10.0 MB | 2.2 MB | **4.62x** | 14.0 MB/s | 32.9 MB/s | ✅ |
| json_api | Structured JSON | 10.0 MB | 3.2 MB | **3.14x** | 7.1 MB/s | 30.4 MB/s | ✅ |
| server_logs | Server log lines | 10.0 MB | 3.4 MB | **2.91x** | 5.1 MB/s | 26.3 MB/s | ✅ |
| mixed_config | YAML/TOML configs | 5.0 MB | 1.3 MB | **3.87x** | 10.8 MB/s | 37.6 MB/s | ✅ |
| binary_headers | ELF headers + padding | 10.0 MB | 2.4 MB | **4.25x** | 4.5 MB/s | 35.2 MB/s | ✅ |
| high_entropy | Pseudorandom (worst case) | 10.0 MB | 10.0 MB | 1.00x | 81.0 MB/s | 85.4 MB/s | ✅ |
| real_go_repo | This repo's Go source | 443 KB | 196 KB | **2.26x** | 3.6 MB/s | 23.9 MB/s | ✅ |

> **Lossless guarantee:** All datasets pass SHA-256 roundtrip verification. High-entropy data is automatically detected and passed through without expansion.

To run the full benchmark suite yourself:
```bash
git checkout benchmark
go run ./benchmark/
```

## Branches

| Branch | Purpose | Command |
|---|---|---|
| [`main`](https://github.com/MrJc01/crompressor) | Public library — clean, documented, `go get`-able | `git checkout main` |
| [`dev`](https://github.com/MrJc01/crompressor/tree/dev) | Research lab — CROM-IA, SRE audits, experiments, UI | `git checkout dev` |
| [`benchmark`](https://github.com/MrJc01/crompressor/tree/benchmark) | Benchmark suite — real performance data | `git checkout benchmark` |

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

Development happens on the [`dev`](https://github.com/MrJc01/crompressor/tree/dev) branch, which contains the full research lab, SRE audits, and experimental features.

## License

[MIT](LICENSE) © 2026 MrJc01

---

<p align="center">
  <em>"We don't compress data. We index the universe."</em>
</p>
