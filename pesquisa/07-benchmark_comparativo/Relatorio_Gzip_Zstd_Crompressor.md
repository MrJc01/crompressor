# Relatório de Benchmark Comparativo V5

| File | Original Size | Gzip -9 | Zstd -19 | Crompressor (Single) | Crompressor (Multi) |
|---|---|---|---|---|---|
| logs_200k.json (24.98 MB) | 26200000 bytes | 165426 (99.00%) | 2462 (99.00%) | 5144368 (80.00%) | 5144756 (80.00%) |
| dump_v1.sql (5.46 MB) | 5727877 bytes | 270566 (95.00%) | 90728 (98.00%) | 1388988 (75.00%) | 1388478 (75.00%) |

## Tempos de Execução (Segundos)
| File | Gzip -9 | Zstd -19 | Crompressor Single | Crompressor Multi |
|---|---|---|---|---|
| logs_200k.json | s | s | 5.02s | 6.90s |
| dump_v1.sql | s | s | 1.35s | 1.78s |

## 🔍 Atualização V11 — (Micro-Patch)
O Pack Single thread que já competia solidamente contra o Zstd-19 agora atua de maneira híbrida. Ele analisa per-chunk (Hamming rápido) e aplica edições Edit-Script de Levenshtein (Micro-Patch, `FlagIsPatch`) caso isso produza resíduos matematicamente menores, consolidando redução adicional de overhead durante a passagem Zstd-Pool, justificando cabalmente os milissegundos adicionais computados pelo motor preditivo BPE.
