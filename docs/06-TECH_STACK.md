# 06 — Tech Stack

> *"Go não é apenas a linguagem. É a filosofia: simplicidade, performance e zero dependências."*

---

## Por Que Go?

A escolha de Go, e não Python, Rust ou C++, é **deliberada** e fundamentada em 5 pilares:

### 1. Binários Estáticos (Distribuição Zero-Friction)

```bash
# Um único arquivo. Sem runtime. Sem dependências.
$ file crompressor-unpack
crompressor-unpack: ELF 64-bit LSB executable, x86-64, statically linked

# O usuário baixa, executa, descomprime. Fim.
$ ./crompressor-unpack --input dados.crom --codebook crom.cromdb --output ./
```

Go compila para **binários estáticos** — o `crompressor-unpack` pode ser distribuído como um único executável que roda em qualquer Linux/macOS/Windows sem instalar nada.

### 2. Goroutines (Paralelismo Nativo)

O compilador `crompressor-pack` processa milhões de chunks. Go oferece goroutines com custo de criação de ~2KB cada (vs. ~1MB por thread em Java/C++):

```go
// Processar 8M chunks em paralelo com pool de N workers
sem := make(chan struct{}, runtime.NumCPU())
for i, chunk := range chunks {
    go func(idx int, c Chunk) {
        sem <- struct{}{}
        defer func() { <-sem }()
        results[idx] = compiler.processChunk(c)
    }(i, chunk)
}
```

### 3. Memory Mapping (mmap) Nativo

Go tem suporte direto a `syscall.Mmap()` — essencial para acessar o Codebook de 50GB sem carregar na RAM.

### 4. Cross-Compilation Trivial

```bash
# Compilar para todas as plataformas de uma vez
GOOS=linux   GOARCH=amd64 go build -o crompressor-pack-linux   ./cmd/crompressorpressor-pack
GOOS=darwin  GOARCH=arm64 go build -o crompressor-pack-macos   ./cmd/crompressorpressor-pack
GOOS=windows GOARCH=amd64 go build -o crompressor-pack.exe     ./cmd/crompressorpressor-pack
```

### 5. Tooling Integrado

| Ferramenta | Uso no CROM |
|---|---|
| `go test -bench` | Benchmarks de chunking, busca e delta |
| `go test -race` | Detectar race conditions na compilação paralela |
| `go tool pprof` | Profiling de CPU e memória durante compressão |
| `go vet` | Análise estática de código |

---

## Uso de CGO: Quando e Por Quê

O CROM usa Go puro para 90% do código. CGO é usado **apenas** para bibliotecas C/C++ que não têm equivalente nativo em Go:

### HNSW (hnswlib)

```go
// #cgo CXXFLAGS: -std=c++14 -O2
// #cgo LDFLAGS: -lstdc++ -lm
// #include "hnswlib_wrapper.h"
import "C"
```

**Justificativa:** O `hnswlib` é a implementação mais otimizada e testada de HNSW. Implementar do zero em Go seria custoso e menos performante.

**Alternativa futura:** Avaliar implementações Go-nativas como `github.com/viterin/vek` para eliminar CGO completamente.

### Zstd (compressão do Delta Pool)

```go
// Usando klauspost/compress (Go puro — sem CGO!)
import "github.com/klauspost/compress/zstd"
```

O Zstd é usado via `klauspost/compress` — uma implementação **100% Go**, sem CGO.

---

## Gerenciamento de Memória

### Estratégia de Alocação

| Componente | Memória | Estratégia |
|---|---|---|
| **Codebook** | ~200MB ativo (50GB mapeado) | mmap — OS gerencia |
| **Chunks em processamento** | ~chunk_size × num_workers | Pool de buffers reutilizáveis |
| **Delta Pool** | ~10% do input | Streaming para disco |
| **HNSW queries** | ~embedding_dim × K × sizeof(float32) | Stack allocation |

### Pool de Buffers

```go
var chunkPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, DefaultChunkSize)
        return &buf
    },
}

// Reutilizar buffers em vez de alocar novos a cada chunk
buf := chunkPool.Get().(*[]byte)
defer chunkPool.Put(buf)
```

---

## Dependências do Projeto

```go
// go.mod
module github.com/crom-project/crom

go 1.22

require (
    github.com/klauspost/compress v1.17.0   // Zstd (Go puro)
    github.com/cespare/xxhash/v2 v2.2.0     // Hash rápido para chunks
    github.com/spf13/cobra v1.8.0            // CLI framework
    github.com/schollz/progressbar/v3 v3.14  // Progress bar
)
```

**Princípio:** Mínimo de dependências. Máximo de controle.

---

## Estrutura de Diretórios

```
crompressor/
├── cmd/
│   ├── crompressor-pack/main.go       # CLI Compilador
│   ├── crompressor-unpack/main.go     # CLI Decompilador
│   └── crompressor-verify/main.go     # CLI Verificação
├── internal/
│   ├── chunker/                # Engine de chunking
│   ├── codebook/               # Acesso ao Codebook (mmap + lookup)
│   ├── delta/                  # XOR + compressão de resíduos
│   ├── format/                 # Formato .crom (read/write)
│   ├── hnsw/                   # Wrapper CGO para hnswlib
│   └── verify/                 # Validação SHA-256
├── pkg/
│   └── cromlib/                # API pública para integração
├── docs/                       # Documentação (esta pasta)
├── testdata/                   # Fixtures de teste
├── scripts/                    # Scripts de build e benchmark
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

> **Próximo:** [07 - Segurança e Soberania](07-SEGURANCA_E_SOBERANIA.md)
