# Crompressor Benchmark Suite

Suíte de benchmarks completa para demonstrar as capacidades reais do Crompressor.

## Como Rodar

```bash
# A partir da raiz do repositório:
go run ./benchmark/
```

O script automaticamente:
1. **Gera 8 datasets** determinísticos (código Go, JSON, logs, configs, binários, fractal, random)
2. **Treina um codebook** específico para cada dataset
3. **Comprime** (Pack) e **descomprime** (Unpack) cada dataset
4. **Verifica integridade** SHA-256 bit-a-bit
5. **Compara** com gzip -9 e zstd -19 (se disponíveis)
6. **Gera relatório** Markdown em `benchmark/RESULTS.md`

## Datasets

| Dataset | Descrição | Tamanho |
|---|---|---|
| `go_source` | Código Go repetitivo com variações | 10 MB |
| `json_api` | JSON estruturado com campos repetitivos | 10 MB |
| `server_logs` | Logs de servidor com timestamps | 10 MB |
| `mixed_config` | YAML configs com seções repetidas | 5 MB |
| `binary_headers` | Headers ELF + padding + structs | 10 MB |
| `polynomial` | Dados fractal (ax²+bx+c mod 256) | 1 MB |
| `high_entropy` | Dados pseudorandom (pior caso) | 10 MB |
| `real_go_repo` | Código Go do próprio crompressor | ~500 KB |

## Resultados

Veja [RESULTS.md](RESULTS.md) após execução.

## Pré-requisitos

- Go 1.22+
- (Opcional) `gzip` e `zstd` instalados para comparação externa
