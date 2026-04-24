# Relatório de Auditoria CROM

**Data da Execução:** 2026-04-24 17:27:12

## 1. Bench Ratio (Eficiência de Compressão)

| Arquivo | Tamanho Original | Tamanho CROM | 
|---------|------------------|--------------|
| ARQUIVO | TAMANHO_ORIG(bytes) | TAMANHO_CROM(bytes) |
| random.bin | 5242880 | 5242992 |
| text.txt | 3638894 | 762495 |
| code.go | 608894 | 128566 |

*O Hit Rate pode ser calculado comparando as colunas.*

## 2. CDC Resilience (Shift de 1 Byte)

```text
Tamanho Original (.crom): 674392 bytes
Tamanho Shifted  (.crom): 674393 bytes
Diferença Absoluta: 1 bytes
```

## 3. Sovereignty Kill (Auto-Unmount FUSE)

✅ Passou: Ponto de montagem desapareceu instantaneamente em contato com a morte do codebook.

## 4. Fuzziness Diff (LSH Clones)

```text
img.00.raw - Tamanho: 1048576 bytes
img.05.raw - Tamanho: 1048576 bytes
img.10.raw - Tamanho: 1048576 bytes
```

