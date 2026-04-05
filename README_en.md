<p align="center">
  <h1 align="center">🧬 Crompressor</h1>
  <p align="center"><strong>Semantic Compression Engine for Go</strong></p>
  <p align="center">
    <a href="https://pkg.go.dev/github.com/MrJc01/crompressor"><img src="https://pkg.go.dev/badge/github.com/MrJc01/crompressor.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/MrJc01/crompressor"><img src="https://goreportcard.com/badge/github.com/MrJc01/crompressor" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
  </p>
</p>

<p align="center">
  🇺🇸 <strong>English</strong> | <a href="README.md">🇧🇷 Ler em Português</a>
</p>

---

**Crompressor** is a high-performance, lossless compression library written in Go. It combines semantic extraction via LSH B-Tree indexing, cosine-similarity search (HNSW), and a trainable codebook to achieve aggressive compression ratios — especially on structured data like source code, logs, JSON, config files, and AI model tensors.

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

### Understanding the Architecture (File Extensions)

Before running the commands, it's crucial to understand what each file represents in the Crompressor ecosystem:

- `data.bin` or `./my-data/`: Your **original, raw data** (JSON, logs, code, etc.).
- `codebook.cromdb`: The **Semantic Dictionary (The Brain)**. Generated during the `train` step, it stores the patterns extracted from your data. Without it, you cannot decompress the file later. It can be reused across thousands of similar files.
- `data.crom`: The final **Super-Compressed File**. It only contains the "coordinates" (deltas) pointing to the patterns stored inside the Codebook.

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

Real results from the automated test suite ([`benchmark`](https://github.com/MrJc01/crompressor/tree/benchmark) branch). Run: `git checkout benchmark && go run ./benchmark/`

### Compression Ratio

| Dataset | Type | Original | Packed | Ratio | Pack | Unpack | SHA-256 |
|---|---|---|---|---|---|---|---|
| go_source | Repetitive Go code | 10 MB | 2.2 MB | **4.62x** | 7.3 MB/s | 12.8 MB/s | ✅ |
| json_api | Structured JSON | 10 MB | 3.2 MB | **3.14x** | 3.2 MB/s | 13.7 MB/s | ✅ |
| binary_headers | ELF headers + padding | 10 MB | 2.4 MB | **4.25x** | 4.3 MB/s | 31.1 MB/s | ✅ |
| mixed_config | YAML/TOML configs | 5 MB | 1.3 MB | **3.87x** | 6.9 MB/s | 16.9 MB/s | ✅ |
| server_logs | Server log lines | 10 MB | 3.4 MB | **2.91x** | 3.9 MB/s | 16.8 MB/s | ✅ |
| high_entropy | Pseudorandom (worst) | 10 MB | 10 MB | 1.00x | 64 MB/s | 57 MB/s | ✅ |

### Scaling (1MB → 500MB)

| Size | Ratio | Pack Speed | Unpack Speed |
|---|---|---|---|
| 1 MB | **2.81x** | 3.0 MB/s | 13.4 MB/s |
| 10 MB | **2.91x** | 6.6 MB/s | 33.3 MB/s |
| 100 MB | **2.92x** | 9.0 MB/s | 37.2 MB/s |
| 500 MB | **2.93x** | 7.6 MB/s | 31.9 MB/s |

### Chunker Comparison (best per dataset)

| Dataset | Fixed-128B | FastCDC | ACAC |
|---|---|---|---|
| json_api | 3.13x | **4.10x** 🏆 | 1.74x |
| server_logs | 2.88x | **3.66x** 🏆 | 2.50x |
| go_source | **4.60x** 🏆 | 4.30x | 1.86x |

### VFS FUSE Mount

| Metric | Value |
|---|---|
| VFS Sequential Read | **84.5 MB/s** |
| Direct Disk Read | 319.4 MB/s |
| First-Byte Latency | 197ms |
| Integrity via VFS | ✅ SHA-256 MATCH |

### Docker FUSE Cascade

✅ **SUCCESS** — Docker built and ran a container from a 3-layer FUSE cascade:
`.crom` → CROM VFS Mount → OverlayFS → `docker build` → `docker run`

> **Lossless guarantee:** All tests pass SHA-256 roundtrip verification. High-entropy data is automatically detected and passed through without expansion.

### 🔄 Comparison vs Standard Tools

Crompressor is a semantic, dictionary-based compiler. It is not designed to beat `gzip` or `zstd` in pure byte-compression of random files, but rather to enable zero-copy VFS streaming of highly structured data.

| Dataset | Crompressor | gzip -9 | zstd -19 | Best Raw Ratio |
|---|---|---|---|---|
| go_source | 4.62x | 38.26x | 69.09x | zstd |
| json_api | 3.14x | 8.09x | 9.96x | zstd |
| binary_headers | 4.25x | 24.70x | 29.04x | zstd |
| server_logs | 2.91x | 6.48x | 8.45x | zstd |
| high_entropy | 1.00x (Bypass) | 1.00x | 1.00x | 🏆 CROM |

*Note: `gzip` and `zstd` achieve higher raw ratios by using generalized entropy coding (LZ77, FSE) in C. Crompressor sacrifices raw ratio to allow O(1) random access reads (mounting via FUSE) without loading the entire archive into memory.*

## 🎯 Use Cases

### ✅ When Crompressor Excels
- **Database / Logs VFS:** When you want a PostgreSQL database or JSON server logs compressed, but actively readable in real-time.
- **Out-of-Core Execution:** Running heavy software (like Docker layers, VM images, or Minecraft) straight out of a compressed volume over FUSE.
- **P2P Syncing:** The Codebook-Delta architecture allows massive semantic deduplication across distributed network nodes (Mesh/GossipSub).
- **Structured Data:** JSON APIs, XML, YAML, CSV, and repetitive Go/Python code repositories.

### ❌ When NOT to use Crompressor
- **Pre-compressed Data:** Images (JPG, PNG), Videos (MP4), or Archives (.zip, .tar.gz). The entropy shield will automatically detect these and pass them through to save CPU, meaning no compression will occur.
- **Cold Storage Archiving:** If you just want to zip a folder for offline backup and don't care about real-time VFS reading, standard `zstd -19` or `7z` will give you astronomically better file sizes.
- **Small Generic Files:** If you are compressing 50KB of random text, zipping it is better than training a semantic model for it.

## ⚙️ How it Works

1. **Train:** Creates a semantic dictionary (Codebook) from a representative dataset using Neural BPE or LSH.
2. **Pack:** Splits your data into context-aware chunks (via FastCDC or Semantic Delimiters), finds the closest codebook pattern via Cosine Similarity, and stores only the XOR delta.
3. **Mount:** The `.crom` file and `.cromdb` codebook are mounted directly into the Linux Kernel via FUSE.
4. **Unpack (On-the-fly):** When Docker or an App requests a byte, the motor reconstructs the exact original chunk in microseconds without unpacking the rest of the 10GB file.

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
