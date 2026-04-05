# 📚 Crompressor — Documentação Técnica

Artigos detalhados sobre arquitetura, benchmarks e análise de performance do Crompressor.

## Índice

| # | Artigo | Descrição |
|:--|:-------|:----------|
| 1 | [Arquitetura do Sistema](01-arquitetura.md) | Como o Crompressor funciona internamente |
| 2 | [Benchmark de Compressão](02-benchmark-compressao.md) | Resultados detalhados de compressão em diferentes tipos de arquivo |
| 3 | [Benchmark de Rede — Two Brains](03-benchmark-rede.md) | Análise completa de economia de banda com codebooks compartilhados |

## Como Reproduzir

```bash
# Benchmark de Compressão
crompressor train -i <dados> -o codebook.cromdb -s 16384
crompressor pack -i <arquivo> -o <arquivo>.crom -c codebook.cromdb
crompressor verify --original <arquivo> --restored <restaurado>

# Benchmark de Rede
./scripts/network_benchmark.sh
```
