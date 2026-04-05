# 10 — Estratégia de Lançamento MVP

> *"4 semanas. Do zero ao primeiro arquivo descomprimido com fidelidade absoluta."*

---

## Cronograma de 4 Semanas

### Semana 1: Fundação (Codebook + Chunking)

| Dia | Tarefa | Critério de Sucesso |
|---|---|---|
| 1-2 | Estrutura do projeto Go (`go mod init`, `cmd/`, `internal/`) | `go build ./...` compila sem erro |
| 2-3 | Implementar `FixedChunker` (chunks de 128 bytes) | Teste: arquivo → chunks → reconstrução = idêntico |
| 3-4 | Criar formato `.cromdb` básico (header + data section) | Consegue gravar e ler codewords de um arquivo binário |
| 4-5 | Implementar `codebook.Lookup(id)` com mmap | Lookup de 1M IDs em <1s |
| 5 | Criar mini-Codebook de teste (1MB, ~8K codewords) | `go test ./internal/codebook/ -v` PASS |

### Semana 2: Motor de Busca + Delta

| Dia | Tarefa | Critério de Sucesso |
|---|---|---|
| 1-2 | Implementar embedding (SimHash) para chunks | Chunks similares → embeddings com distância pequena |
| 2-3 | Integrar HNSW (hnswlib via CGO ou lib Go nativa) | Busca em 8K codewords retorna vizinho correto >95% |
| 3-4 | Implementar `delta.XOR()` e `delta.Apply()` | `Apply(pattern, XOR(original, pattern)) == original` |
| 4-5 | Comprimir Delta Pool com Zstd | Pool de deltas comprime >10x quando PMR é alto |
| 5 | Integrar busca + delta em pipeline end-to-end | Chunk → search → delta → reconstrução = lossless |

### Semana 3: Compilador + Formato .crom

| Dia | Tarefa | Critério de Sucesso |
|---|---|---|
| 1-2 | Definir e implementar formato `.crom` (header + chunk table + delta pool) | Serialização e deserialização roundtrip perfeito |
| 2-3 | Criar `crompressor-pack` CLI (cobra) | `crompressor pack -i file -o out.crom -c mini.cromdb` funciona |
| 3-4 | Criar `crompressor-unpack` CLI | `crompressor unpack -i out.crom -o restored -c mini.cromdb` funciona |
| 4-5 | Implementar verificação SHA-256 | `sha256sum original == sha256sum restored` ✅ |
| 5 | Paralelizar compilação com goroutines | Speedup >3x em CPU de 4+ cores |

### Semana 4: Benchmark + Polish

| Dia | Tarefa | Critério de Sucesso |
|---|---|---|
| 1-2 | Criar suite de benchmarks (`go test -bench`) | Métricas: MB/s compile, MB/s unpack, PMR |
| 2-3 | Testar com dataset real (1GB de arquivos mistos) | Compressão >5:1 com mini-Codebook |
| 3-4 | Comparar com `gzip -9` e `zstd -19` | Documentar resultados em tabela |
| 4 | Criar comando `crompressor analyze` (cobertura do Codebook) | Relatório de PMR e CR estimado |
| 5 | Finalizar documentação e README | Projeto compilável e testável por terceiros |

---

## Checklist Técnico de Prioridades

### 🔴 P0 — Crítico (Semana 1-2)

- [ ] `go mod init github.com/crom-project/crom`
- [ ] `internal/chunker/fixed.go` — Chunking de tamanho fixo
- [ ] `internal/codebook/mmap.go` — Leitura do Codebook via mmap
- [ ] `internal/codebook/lookup.go` — Lookup por ID (O(1))
- [ ] `internal/delta/xor.go` — XOR e Apply
- [ ] `internal/delta/compress.go` — Zstd do Delta Pool
- [ ] Testes unitários para cada componente acima

### 🟡 P1 — Importante (Semana 2-3)

- [ ] `internal/hnsw/search.go` — Busca de vizinhos mais próximos
- [ ] `internal/codebook/embed.go` — SimHash embedding
- [ ] `internal/format/writer.go` — Serialização do .crom
- [ ] `internal/format/reader.go` — Deserialização do .crom
- [ ] `internal/verify/sha256.go` — Validação de integridade
- [ ] `cmd/crompressorpressor-pack/main.go` — CLI do compilador
- [ ] `cmd/crompressorpressor-unpack/main.go` — CLI do decompilador

### 🟢 P2 — Desejável (Semana 3-4)

- [ ] Pipeline paralelo (goroutines + semáforo)
- [ ] Progress bar (schollz/progressbar)
- [ ] Relatório de compilação (estatísticas detalhadas)
- [ ] `cmd/crompressorpressor-verify/main.go` — CLI de verificação
- [ ] `cmd/crompressorpressor-analyze/main.go` — Análise de cobertura

### 🔵 P3 — Futuro (Pós-MVP)

- [ ] Content-Defined Chunking (Rabin)
- [ ] Codebook Builder (geração automatizada)
- [ ] Codebooks especializados (medical, geo, code)
- [ ] Criptografia AES-256-GCM do .crom
- [ ] Otimização SIMD para XOR
- [ ] API HTTP para compressão remota
- [ ] GUI desktop (Wails ou Fyne)

---

## Estrutura de Diretórios do Repositório

```
crompressor/
│
├── cmd/                            # Binários executáveis
│   ├── crompressor-pack/
│   │   └── main.go                 # CLI: compilador
│   ├── crompressor-unpack/
│   │   └── main.go                 # CLI: decompilador
│   ├── crompressor-verify/
│   │   └── main.go                 # CLI: verificador
│   └── crompressor-analyze/
│       └── main.go                 # CLI: analisador de cobertura
│
├── internal/                       # Pacotes internos (não exportados)
│   ├── chunker/
│   │   ├── chunker.go              # Interface Chunker
│   │   ├── fixed.go                # FixedChunker
│   │   ├── rabin.go                # RabinChunker (P3)
│   │   └── chunker_test.go
│   ├── codebook/
│   │   ├── codebook.go             # Interface CodebookReader
│   │   ├── mmap.go                 # Implementação com mmap
│   │   ├── lookup.go               # Lookup direto por ID
│   │   ├── search.go               # Busca HNSW
│   │   ├── embed.go                # SimHash embedding
│   │   └── codebook_test.go
│   ├── delta/
│   │   ├── xor.go                  # XOR e Apply
│   │   ├── compress.go             # Zstd do Delta Pool
│   │   └── delta_test.go
│   ├── format/
│   │   ├── header.go               # Struct do header .crom
│   │   ├── chunktable.go           # Chunk Table
│   │   ├── writer.go               # Serialização .crom
│   │   ├── reader.go               # Deserialização .crom
│   │   └── format_test.go
│   ├── hnsw/
│   │   ├── hnsw.go                 # Interface HNSWIndex
│   │   ├── cgo_wrapper.go          # Binding CGO (hnswlib)
│   │   └── hnsw_test.go
│   └── verify/
│       ├── sha256.go               # Validação SHA-256
│       └── verify_test.go
│
├── pkg/
│   └── cromlib/
│       ├── compiler.go             # API pública: Compile()
│       ├── unpacker.go             # API pública: Unpack()
│       └── cromlib.go              # Tipos exportados
│
├── scripts/
│   ├── build.sh                    # Build multi-plataforma
│   ├── benchmark.sh                # Runner de benchmarks
│   └── gen_mini_codebook.go        # Gerador de Codebook de teste
│
├── testdata/
│   ├── mini.cromdb                 # Mini Codebook (1MB) para testes
│   ├── sample_1mb.bin              # Arquivo de teste
│   └── expected/                   # Outputs esperados
│
├── docs/                           # Esta documentação
│   ├── 01-CONCEITO_E_VISAO.md
│   ├── 02-ARQUITETURA_DO_SISTEMA.md
│   ├── ...
│   └── 10-ESTRATEGIA_MVP.md
│
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

---

## Makefile

```makefile
.PHONY: build test bench clean

build:
	go build -o bin/crompressor-pack   ./cmd/crompressorpressor-pack
	go build -o bin/crompressor-unpack ./cmd/crompressorpressor-unpack
	go build -o bin/crompressor-verify ./cmd/crompressorpressor-verify

test:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem -benchtime=10s ./...

clean:
	rm -rf bin/

lint:
	go vet ./...
	golangci-lint run

demo: build
	@echo "=== Compilando arquivo de teste ==="
	./bin/crompressor-pack -i testdata/sample_1mb.bin -o /tmp/test.crom -c testdata/mini.cromdb
	@echo "=== Descompilando ==="
	./bin/crompressor-unpack -i /tmp/test.crom -o /tmp/restored.bin -c testdata/mini.cromdb
	@echo "=== Verificando integridade ==="
	sha256sum testdata/sample_1mb.bin /tmp/restored.bin
```

---

## Critério de Sucesso do MVP

| Critério | Meta |
|---|---|
| **Compressão funcional** | Arquivo de 1MB comprime e descomprime sem perda |
| **Fidelidade lossless** | SHA-256 original = SHA-256 restaurado (100%) |
| **Taxa de compressão** | >5:1 com mini-Codebook de 1MB |
| **Velocidade** | >10 MB/s de compilação em CPU moderna |
| **Testes** | >80% de cobertura em pacotes `internal/` |
| **Documentação** | README completo + docs técnicos (✅ este documento) |

---

> *"Não comprimimos dados. Compilamos realidade."*
