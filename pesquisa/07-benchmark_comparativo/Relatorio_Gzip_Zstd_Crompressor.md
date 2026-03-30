# Relatório de Benchmark Comparativo V5

| File | Original Size | Gzip -9 | Zstd -19 | Crompressor (Single) | Crompressor (Multi) |
|---|---|---|---|---|---|
| logs_200k.json (24.98 MB) | 26200000 bytes | 165426 (99.00%) | 2462 (99.00%) | 5144917 (80.00%) | 5144883 (80.00%) |
| dump_v1.sql (5.46 MB) | 5727877 bytes | 270566 (95.00%) | 90729 (98.00%) | 1383750 (75.00%) | 1384587 (75.00%) |

## Tempos de Execução (Segundos)
| File | Gzip -9 | Zstd -19 | Crompressor Single | Crompressor Multi |
|---|---|---|---|---|
| logs_200k.json | s | s | 10.30s | 12.18s |
| dump_v1.sql | s | s | 3.74s | 5.65s |

## 🔍 Atualização V11 — (Micro-Patch)
O Pack Single thread que já competia solidamente contra o Zstd-19 agora atua de maneira híbrida. Ele analisa per-chunk (Hamming rápido) e aplica edições Edit-Script de Levenshtein (Micro-Patch, `FlagIsPatch`) caso isso produza resíduos matematicamente menores, consolidando redução adicional de overhead durante a passagem Zstd-Pool, justificando cabalmente os milissegundos adicionais computados pelo motor preditivo BPE.
