# Arquitetura do Crompressor

## Visão Geral

O Crompressor é um motor de compressão baseado em dicionário que combina três primitivas compostas:

```
CROM(X) = Σᵢ C[q(chunkᵢ(X))] ⊕ Δᵢ
```

### 1. CDC — Content-Defined Chunking

Utiliza Rabin Fingerprint para dividir a entrada em blocos semânticos de tamanho variável.
Isso permite deduplicação entre arquivos estruturalmente similares, ao contrário de chunking fixo
que é sensível a inserções/deleções.

**Implementação:** `internal/chunker/` (fixo, CDC, FastCDC)

### 2. VQ — Vector Quantization

Mapeia cada chunk para a entrada mais próxima em um codebook pré-treinado:

```
q(x) = argmin_k ‖x - eₖ‖²
```

O lookup é O(1) após o treinamento via LSH (Locality-Sensitive Hashing) com B-Tree indexada.

**Implementação:** `internal/search/` (LSH, Linear, Multi-strategy)

### 3. XOR Delta

Armazena o resíduo entre o chunk original e o padrão reconstruído do codebook:

```
Δᵢ = original XOR reconstructed
```

- **Modo Edge:** `Δᵢ` é descartado → compressão lossy
- **Modo Archive:** `Δᵢ` é armazenado → compressão lossless

**Implementação:** `internal/delta/` (XOR, Diff, Patch)

---

## Pipeline de Compressão

```
Input File
    │
    ▼
┌─────────────┐
│ Entropy     │ ← Análise de Shannon (64KB amostra)
│ Analysis    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Chunking    │ ← CDC (Rabin) ou Fixed-size
│             │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Codebook    │ ← LSH lookup O(1) → índice do padrão mais similar
│ Matching    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Delta       │ ← XOR(chunk, pattern) → resíduo
│ Encoding    │   Edge: descarta | Archive: armazena
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Format      │ ← Serialização .crom v8
│ Writer      │   Header + Chunk Table + Delta Blob
└─────────────┘
```

---

## Formato do Arquivo .crom (v8)

| Seção | Tamanho | Descrição |
|-------|---------|-----------|
| Magic | 4 bytes | `CROM` |
| Version | 1 byte | `0x08` |
| Header | 136 bytes | Metadata, SHA-256 original, flags |
| Chunk Table | variável | Índices do codebook + delta sizes |
| Delta Blob | variável | Resíduos XOR concatenados |

**Implementação:** `pkg/format/` (reader, writer, format)

---

## Codebook (.cromdb)

O codebook é um dicionário treinado por K-Means sobre o corpus de dados-alvo.
Cada entrada (codeword) é um vetor de bytes representando um padrão frequente.

- **Treinamento:** `internal/trainer/` (extração de frequência, seleção elite, BPE)
- **Serialização:** `internal/codebook/` (leitura, mmap, lookup)
- **Tamanho típico:** 256 a 16384 entradas

---

## Referências

- [Estudo Matemático Completo](https://github.com/MrJc01/crompressor-matematica)
- [Edge vs Archive: quando usar cada modo](modes.md)
- [Benchmarks e metodologia](benchmarks.md)
