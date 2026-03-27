# 09 — Benchmarks e Métricas

> *"Não competimos no curto alcance. Competimos onde o conhecimento prévio esmaga a estatística."*

---

## Filosofia de Benchmark

O CROM **não** pretende ser mais rápido que Gzip para comprimir um único arquivo de 1KB. Compressores tradicionais são otimizados para esse cenário.

O CROM brilha onde **correlações de longo alcance** e **redundância estrutural** dominam:
- Datasets grandes (GB-TB)
- Coleções de arquivos similares
- Dados com padrões repetitivos (logs, imagens, documentos)

---

## Métricas de Avaliação

### 1. Taxa de Compressão (Compression Ratio)

```
CR = Tamanho Original / Tamanho Comprimido

Exemplo:
  1TB original → 50GB .crom → CR = 20:1
  (Sem contar o Codebook, que é custo fixo compartilhado)
```

### 2. Taxa de Compressão Líquida (Net CR)

```
Net CR = Tamanho Original / (Tamanho .crom + Codebook Amortizado)

Se Codebook (50GB) é usado para comprimir 10TB:
  Amortização = 50GB / 10TB = 0.5%
  Net CR ≈ CR × 0.995 (impacto desprezível)
```

### 3. Perfect Match Rate (PMR)

```
PMR = Chunks com Delta vazio / Total de Chunks × 100%

Meta MVP:  PMR > 50%
Meta Prod: PMR > 75%
```

### 4. Velocidade

```
Compile Speed  = MB/s de dados processados pelo crompressor-pack
Unpack Speed   = MB/s de dados reconstruídos pelo crompressor-unpack
```

---

## Metas de Performance vs. Concorrentes

### Cenário: Dataset de 100GB de Documentos Mistos

| Métrica | Gzip -9 | Zstd -19 | CROM (1GB CB) | CROM (50GB CB) |
|---|---|---|---|---|
| **Taxa de Compressão** | 3:1 | 5:1 | 15:1 | 50:1+ |
| **Tempo de Compressão** | 20min | 8min | 45min | 60min |
| **Tempo de Descompressão** | 2min | 1min | 30s | 30s |
| **Arquivo Resultante** | 33GB | 20GB | 6.7GB | 2GB |
| **Custo Fixo (Modelo)** | 0 | 0 | 1GB | 50GB |
| **Requer Modelo p/ Abrir** | ❌ | ❌ | ✅ | ✅ |

### Cenário: 1M de Imagens JPEG (500GB)

| Métrica | Gzip -9 | Zstd -19 | CROM (50GB CB) |
|---|---|---|---|
| **Taxa de Compressão** | 1.05:1 | 1.1:1 | 20:1+ |
| **Arquivo Resultante** | 476GB | 455GB | 25GB |

> **JPEGs já são comprimidos** — Gzip/Zstd quase não conseguem reduzir. O CROM encontra padrões estruturais **entre** imagens, não dentro delas.

### Cenário: 10TB de Logs de Servidor (90 dias)

| Métrica | Gzip -9 | Zstd --train | CROM (10GB CB) |
|---|---|---|---|
| **Taxa de Compressão** | 8:1 | 12:1 | 80:1+ |
| **Arquivo Resultante** | 1.25TB | 833GB | 125GB |

---

## Onde o CROM Perde

| Cenário | Gzip/Zstd | CROM | Por quê |
|---|---|---|---|
| Arquivo único <1MB | ✅ Melhor | ❌ Overhead do Codebook | Custo fixo > ganho |
| Dados aleatórios (random) | ~1:1 | ~1:1 | Entropia máxima, sem padrões |
| Streaming single-pass | ✅ Rápido | ❌ Precisa do Codebook | Latência de inicialização |
| Sem Codebook disponível | ✅ Funciona | ❌ Impossível | Dependência arquitetural |

---

## Framework de Benchmark

```go
// benchmark_test.go
func BenchmarkCompile(b *testing.B) {
    codebook := loadCodebook("testdata/mini.cromdb")
    compiler := NewCompiler(codebook)
    data := loadTestData("testdata/sample_100mb.bin")
    
    b.ResetTimer()
    b.SetBytes(int64(len(data)))
    
    for i := 0; i < b.N; i++ {
        compiler.Compile(data, io.Discard)
    }
    // Resultado: MB/s processados
}

func BenchmarkUnpack(b *testing.B) {
    codebook := loadCodebook("testdata/mini.cromdb")
    unpacker := NewUnpacker(codebook)
    cromData := loadTestData("testdata/sample.crom")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        unpacker.Unpack(cromData, io.Discard)
    }
}
```

### Comando de Benchmark

```bash
# Executar todos os benchmarks
go test -bench=. -benchmem -benchtime=10s ./...

# Comparar versões com benchstat
go test -bench=. -count=10 ./... > old.txt
# (fazer mudanças)
go test -bench=. -count=10 ./... > new.txt
benchstat old.txt new.txt
```

---

## Métricas de Qualidade do Codebook

```bash
# Analisar cobertura do Codebook contra um dataset
crompressor analyze --codebook crom.cromdb --input ./dataset/

# Saída esperada:
# ╔═══════════════════════════════════════════╗
# ║        CODEBOOK COVERAGE ANALYSIS         ║
# ╠═══════════════════════════════════════════╣
# ║  Dataset:         ./dataset/ (50GB)       ║
# ║  Chunks Total:    409,600,000             ║
# ║  Perfect Match:   307,200,000 (75.0%)     ║
# ║  Good Match:      81,920,000 (20.0%)      ║
# ║  Partial Match:   16,384,000 (4.0%)       ║
# ║  No Match:        4,096,000 (1.0%)        ║
# ║                                           ║
# ║  Estimated CR:    42:1                    ║
# ║  Codebook Util:   68% of codewords used   ║
# ╚═══════════════════════════════════════════╝
```

---

> **Próximo:** [10 - Estratégia MVP](10-ESTRATEGIA_MVP.md)
