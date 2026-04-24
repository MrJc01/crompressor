# Benchmarks

## Metodologia

Todos os benchmarks são executados com:
- `go test -bench=. -benchmem -benchtime=10s`
- Single core, sem paralelismo artificial
- Dados gerados deterministicamente para reprodutibilidade

## Resultados — Compressão por Tipo de Dado

| Tipo de Dado | Entropia (bits/byte) | Modo Edge | Modo Archive |
|-------------|---------------------|-----------|-------------|
| Pesos LLM (Gaussiano) | ~4.2 | 8.0x | 1.59x |
| Código-fonte Go | ~4.8 | 3.2x | 2.1x |
| Logs de servidor | ~5.1 | 2.8x | 1.8x |
| Config JSON/YAML | ~4.0 | 4.5x | 2.5x |
| Binários (headers) | ~6.2 | 1.8x | 1.3x |
| Ruído aleatório (urandom) | ~8.0 | 5.1x* | 1.0x** |

\* Edge em ruído: alta compressão mas alta distorção (MSE elevado)
\** Archive em ruído: resíduo XOR consome todo o espaço (limite de Shannon)

## Throughput

| Operação | Velocidade |
|----------|-----------|
| Pack (Edge) | ~45 MB/s |
| Pack (Archive) | ~35 MB/s |
| Unpack | ~50 MB/s |
| Codebook lookup (LSH) | O(1), ~21 ns/weight |

## Como Reproduzir

```bash
# Benchmarks unitários
make bench

# Teste de round-trip completo
make demo

# Stress test (100 MB)
make stress
```

## Referência

Análise completa de rate-distortion e provas formais:
→ [crompressor-matematica](https://github.com/MrJc01/crompressor-matematica)
