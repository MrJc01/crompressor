# 03 — Estrutura do Dicionário (Codebook Universal)

> *"O Codebook não armazena conhecimento. Ele armazena a realidade fragmentada e indexada."*

---

## O Que é o Codebook?

O Codebook Universal (`.cromdb`) é o componente central do CROM. É um **banco de dados binário estático** de 50GB+ que contém bilhões de fragmentos de padrões binários (chamados **Codewords**, ou "palavras-código") extraídos de datasets massivos.

Ele funciona como uma **tabela de consulta universal** onde cada padrão binário tem um **ID único** (índice numérico). O compilador `crompressor-pack` busca padrões similares nesta tabela; o decompilador `crompressor-unpack` faz lookups diretos por ID.

---

## Anatomia do Arquivo `.cromdb`

```
┌────────────────────────────────────────────────────┐
│                 CROMDB FORMAT v1                    │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  HEADER (512 bytes)                          │   │
│  │  ├── Magic: "CROMDB" (6 bytes)               │   │
│  │  ├── Version: uint16                         │   │
│  │  ├── Codeword Size: uint16 (ex: 64, 128)     │   │
│  │  ├── Codeword Count: uint64                  │   │
│  │  ├── Embedding Dim: uint16                   │   │
│  │  ├── Index Offset: uint64                    │   │
│  │  ├── Data Offset: uint64                     │   │
│  │  ├── Build Hash: SHA-256                     │   │
│  │  └── Reserved: padding to 512               │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  HNSW INDEX (~5-10GB)                        │   │
│  │  ├── Graph Layers (L0..Lmax)                 │   │
│  │  ├── Node Embeddings (float32 vectors)       │   │
│  │  └── Navigation Tables                       │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  CODEWORD DATA (~40GB)                       │   │
│  │  ├── [ID=0]  64-512 bytes (raw pattern)      │   │
│  │  ├── [ID=1]  64-512 bytes (raw pattern)      │   │
│  │  ├── [ID=2]  64-512 bytes (raw pattern)      │   │
│  │  ├── ...                                     │   │
│  │  └── [ID=N]  64-512 bytes (raw pattern)      │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  METADATA TABLE                              │   │
│  │  ├── Frequency counts per codeword           │   │
│  │  ├── Category tags (image/text/binary/code)  │   │
│  │  └── Collision statistics                    │   │
│  └─────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────┘
```

### Dimensionamento

| Parâmetro | Valor (MVP) | Valor (Produção) |
|---|---|---|
| **Codeword Size** | 64 bytes | 128-512 bytes |
| **Codeword Count** | ~15 milhões | ~500 milhões |
| **Embedding Dim** | 32 floats | 64-128 floats |
| **HNSW Index Size** | ~500MB | ~5-10GB |
| **Codeword Data** | ~1GB | ~40-50GB |
| **Total** | ~1.5GB | ~50GB |

---

## Memory Mapping (mmap) em Go

O Codebook de 50GB **não é carregado na RAM**. Utilizamos `mmap` (memory-mapped files) para permitir que o sistema operacional carregue apenas as páginas de memória que estão sendo acessadas.

### Como Funciona

```go
package codebook

import (
    "os"
    "syscall"
    "unsafe"
)

// CodebookReader lê o Codebook via mmap sem carregar tudo na RAM.
type CodebookReader struct {
    file     *os.File
    data     []byte          // mmap'd region
    header   *CodebookHeader
    dataOff  uint64          // offset where codeword data begins
    cwSize   uint16          // codeword size in bytes
}

// Open abre o Codebook via memory mapping.
func Open(path string) (*CodebookReader, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }

    info, _ := f.Stat()
    size := info.Size()

    // mmap: O OS mapeia o arquivo no espaço de endereçamento virtual.
    // Páginas de 4KB são carregadas sob demanda (page fault → disk read).
    data, err := syscall.Mmap(
        int(f.Fd()),
        0,
        int(size),
        syscall.PROT_READ,
        syscall.MAP_SHARED,
    )
    if err != nil {
        f.Close()
        return nil, err
    }

    reader := &CodebookReader{
        file: f,
        data: data,
    }
    reader.parseHeader()
    return reader, nil
}

// Lookup busca um codeword por ID — O(1), acesso direto.
func (cr *CodebookReader) Lookup(id uint64) []byte {
    offset := cr.dataOff + (id * uint64(cr.cwSize))
    return cr.data[offset : offset+uint64(cr.cwSize)]
}

// Close libera o mmap e fecha o arquivo.
func (cr *CodebookReader) Close() error {
    syscall.Munmap(cr.data)
    return cr.file.Close()
}
```

### Vantagens do mmap para o CROM

| Aspecto | Sem mmap | Com mmap |
|---|---|---|
| **RAM necessária** | 50GB (tudo na RAM) | ~200MB (hot pages apenas) |
| **Tempo de inicialização** | 30-60s (carregar tudo) | <1ms (mapeamento virtual) |
| **Concorrência** | Múltiplas cópias na RAM | Compartilhado pelo OS (CoW) |
| **Paginação** | Manual e complexa | Automática pelo kernel |

---

## Indexação HNSW (Hierarchical Navigable Small World)

O HNSW é a estrutura que permite **buscar** no Codebook em tempo sub-linear. Sem ele, encontrar o padrão mais próximo entre 500 milhões de codewords seria inviável.

### Estrutura do Grafo HNSW

```
Layer 3:    [A]───────────────────[F]          (Poucos nós, links longos)
             │                     │
Layer 2:    [A]──────[C]──────[E]──[F]         (Mais nós, links médios)
             │        │        │    │
Layer 1:    [A]──[B]──[C]──[D]──[E]──[F]──[G]  (Muitos nós, links curtos)
             │    │    │    │    │    │    │
Layer 0:    [A][B][C][D][E][F][G][H][I][J]...  (Todos os nós, links locais)
```

### Algoritmo de Busca

```
1. Entrada na camada mais alta (Layer 3)
2. Greedy search: mova para o vizinho mais próximo do vetor de consulta
3. Quando não houver vizinho mais próximo na camada atual, desça uma camada
4. Repita até Layer 0
5. Refine com busca local (beam search) na Layer 0
6. Retorne os K vizinhos mais próximos
```

### Integração em Go via CGO

```go
// #cgo LDFLAGS: -lhnswlib
// #include "hnswlib.h"
import "C"

// HNSWIndex encapsula o índice HNSW para busca no Codebook.
type HNSWIndex struct {
    index unsafe.Pointer
    dim   int
}

// Search busca os K vizinhos mais próximos de um embedding.
func (h *HNSWIndex) Search(embedding []float32, k int) []SearchResult {
    // Chamada CGO para a lib C++ do HNSW.
    // Retorna K pares (id, distância) ordenados por proximidade.
    // Tempo: O(log N) onde N = número de codewords.
    // ...
}
```

### Parâmetros HNSW para o CROM

| Parâmetro | Valor | Justificativa |
|---|---|---|
| `M` (conexões por nó) | 32 | Equilíbrio entre recall e memória |
| `efConstruction` | 200 | Alta qualidade na construção do grafo |
| `efSearch` | 100 | Recall >99% para busca em produção |
| **Recall@1** | >99.5% | Garantia de encontrar o padrão mais próximo |
| **Tempo por query** | <1ms | Viável para milhões de chunks por arquivo |

---

## Pipeline de Construção do Codebook

A construção do Codebook é um processo **offline e pesado** que ocorre uma vez (ou raramente, para atualizações):

```
┌───────────────────────────────────────────────────────────┐
│                  CODEBOOK BUILDER                          │
│                                                            │
│  1. COLETA DE DADOS                                        │
│     └─ 10TB+ de arquivos diversos (imagens, docs, code)    │
│                                                            │
│  2. CHUNKING UNIVERSAL                                     │
│     └─ Dividir tudo em blocos de 64-512 bytes              │
│     └─ Resultado: ~100 bilhões de chunks                   │
│                                                            │
│  3. FEATURE EXTRACTION (Embeddings)                        │
│     └─ Calcular hash/embedding de cada chunk               │
│     └─ Método: SimHash, MinHash ou Locality-Sensitive Hash │
│                                                            │
│  4. CLUSTERING (K-Means / Mini-Batch K-Means)              │
│     └─ Agrupar chunks similares                            │
│     └─ Centróides = Codewords candidatas                   │
│                                                            │
│  5. SELEÇÃO DE CODEBOOK                                    │
│     └─ Top 500M codewords mais frequentes/representativas  │
│     └─ Otimizar cobertura vs. tamanho                      │
│                                                            │
│  6. INDEXAÇÃO HNSW                                         │
│     └─ Construir grafo HNSW sobre os embeddings            │
│     └─ Serializar para disco                               │
│                                                            │
│  7. SERIALIZAÇÃO .cromdb                                   │
│     └─ Escrever header + index + codewords + metadata      │
│     └─ Calcular Build Hash (SHA-256)                       │
└───────────────────────────────────────────────────────────┘
```

---

## Versionamento do Codebook

Cada Codebook possui um **Build Hash** (SHA-256 de todo o conteúdo). Este hash é armazenado no header do arquivo `.crom` para garantir que a descompressão use **exatamente** o mesmo Codebook.

```
┌────────────────┐     ┌──────────────────┐
│  .crom file    │     │   Codebook       │
│                │     │                  │
│  codebook_hash │────▶│  build_hash      │
│  = 0xABCD...   │     │  = 0xABCD...     │
│                │     │                  │
│  ✅ Match!     │     │  ✅ Compatível   │
└────────────────┘     └──────────────────┘
```

Se os hashes não corresponderem, o `crompressor-unpack` **recusa** a descompressão com erro explícito.

---

> **Próximo:** [04 - Especificação do Compilador](04-ESPECIFICACAO_DO_COMPILADOR.md) — Detalhes do chunking, busca e geração do mapa de IDs.
