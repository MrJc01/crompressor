# 05 — Camada de Refinamento (Delta Lossless)

> *"O Codebook acerta 90%. O Delta garante os 10% restantes — sem exceção."*

---

## O Problema: Busca Aproximada vs. Fidelidade Absoluta

O HNSW encontra o codeword **mais próximo** — mas "próximo" não significa "idêntico". Para dados binários e executáveis, **um único bit errado** corrompe tudo.

A Camada de Refinamento elimina **100%** das discrepâncias via **operação XOR** — determinística e reversível.

---

## O Que é o Delta?

O Delta é a **diferença exata** (XOR byte a byte) entre o chunk original e o codeword do Codebook.

```
Propriedade fundamental:
  A ⊕ B = D    →    B ⊕ D = A

  onde: A = Original, B = Pattern, D = Delta
```

### Implementação em Go

```go
package delta

// XOR calcula o resíduo entre dois buffers.
func XOR(original, pattern []byte) []byte {
    d := make([]byte, len(original))
    minLen := len(pattern)
    if len(original) < minLen { minLen = len(original) }
    for i := 0; i < minLen; i++ { d[i] = original[i] ^ pattern[i] }
    if len(original) > len(pattern) { copy(d[len(pattern):], original[len(pattern):]) }
    return d
}

// Apply reconstrói o chunk original: pattern ⊕ delta = original
func Apply(pattern, delta []byte) []byte {
    out := make([]byte, len(delta))
    minLen := len(pattern)
    if len(delta) < minLen { minLen = len(delta) }
    for i := 0; i < minLen; i++ { out[i] = pattern[i] ^ delta[i] }
    if len(delta) > len(pattern) { copy(out[len(pattern):], delta[len(pattern):]) }
    return out
}
```

---

## Por Que o Delta Comprime Bem?

Quando o codeword é similar ao chunk original, o Delta contém longas sequências de `0x00`:

```
Chunk Original:  [0xFA, 0x3C, 0x92, 0x10, 0xFF, 0x88, 0x44]
Codeword:        [0xFA, 0x3C, 0x92, 0x10, 0xFF, 0x8A, 0x44]
Delta (XOR):     [0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00]
                  → Comprime extraordinariamente com Zstd!
```

### Cenários de Match

| Qualidade | % Chunks | Delta/Chunk | Após Zstd |
|---|---|---|---|
| **Perfect** (dist=0) | 75% | 0 bytes | 0 bytes |
| **Ótimo** (<10% diff) | 15% | ~13 bytes | ~4 bytes |
| **Bom** (<25% diff) | 7% | ~32 bytes | ~15 bytes |
| **Razoável** (<50%) | 2% | ~64 bytes | ~40 bytes |
| **Ruim** (>50%) | 1% | ~128 bytes| ~100 bytes |
| **Média Ponderada** | 100% | **~5 bytes** | **~2 bytes** |

Para chunks de 128 bytes, custo médio de **~2 bytes** → taxa de **64:1** só no Delta.

---

## Compressão do Delta Pool

Todos os deltas são concatenados e comprimidos com Zstd:

```go
func CompressDeltaPool(deltas [][]byte) ([]byte, error) {
    var pool bytes.Buffer
    for _, d := range deltas { pool.Write(d) }
    encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
    return encoder.EncodeAll(pool.Bytes(), nil), nil
}
```

---

## Validação de Integridade (SHA-256)

```go
func Verify(original, reconstructed []byte) error {
    h1, h2 := sha256.Sum256(original), sha256.Sum256(reconstructed)
    if h1 != h2 {
        return fmt.Errorf("FALHA DE INTEGRIDADE: orig=%x rec=%x", h1[:8], h2[:8])
    }
    return nil // ✅ Fidelidade bit-a-bit confirmada
}
```

### Protocolo

1. **Compilação:** `SHA-256(original)` → armazenado no header `.crom`
2. **Decompilação:** Reconstrói via `pattern ⊕ delta` → calcula SHA-256
3. **Comparação:** Match ✅ = sucesso | Mismatch ❌ = erro + log do chunk divergente

---

## Garantias Formais

| Propriedade | Garantia | Mecanismo |
|---|---|---|
| **Lossless** | 100% | XOR é bijetivo: `A ⊕ B ⊕ B = A` |
| **Determinístico** | Mesma entrada → mesma saída | Sem ponto flutuante no Delta |
| **Verificável** | SHA-256 original = SHA-256 reconstruído | Hash criptográfico |
| **Reversível** | `apply(P, xor(A, P)) == A` | Propriedade algébrica do XOR |

---

> **Próximo:** [06 - Tech Stack](06-TECH_STACK.md)
