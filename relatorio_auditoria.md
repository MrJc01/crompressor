# Relatório de Auditoria CROM

**Data da Execução:** 2026-03-29 23:01:48

## 1. Bench Ratio (Eficiência de Compressão)

| Arquivo | Tamanho Original | Tamanho CROM | 
|---------|------------------|--------------|
| ARQUIVO | TAMANHO_ORIG(bytes) | TAMANHO_CROM(bytes) |
| random.bin | 5242880 | 5625685 |
| text.txt | 3638894 | 782724 |
| code.go | 608894 | 135851 |

*O Hit Rate pode ser calculado comparando as colunas.*

## 2. CDC Resilience (Shift de 1 Byte)

```text
Tamanho Original (.crom): 647098 bytes
Tamanho Shifted  (.crom): 647108 bytes
Diferença Absoluta: 10 bytes
```

## 3. Sovereignty Kill (Auto-Unmount FUSE)

❌ **Falhou:**
```
[*] Treinando Codebook e empacotando...
╔═══════════════════════════════════════════╗
║          CROM TRAIN (Treinador)           ║
╠═══════════════════════════════════════════╣
║  Input Dir: /tmp/crom_sov_419007/secret.txt ║
║  Output:    /tmp/crom_sov_419007/sov.cromdb ║
║  Target:    8192                          ║
╚═══════════════════════════════════════════╝

✔ Training completed in 66.182644ms
  Files Parsed:    1
  Total Bytes:     1048603
  Unique Patterns: 8192
  Elite Selected:  8192 (Codebook Gerado)
╔═══════════════════════════════════════════╗
║            CROM PACK (Compilador)         ║
╠═══════════════════════════════════════════╣
║  Input:    /tmp/crom_sov_419007/secret.txt ║
║  Output:   /tmp/crom_sov_419007/vault.crom ║
║  Codebook: /tmp/crom_sov_419007/sov.cromdb ║
╚═══════════════════════════════════════════╝

✔ Pack completed in 45.088391ms
  Original Size: 1048603 bytes
  Packed Size:   196780 bytes (18.77% ratio)
  Hit Rate:      99.99% dos chunks no Radar
[*] Montando vault VFS via FUSE
╔═══════════════════════════════════════════╗
║       CROM VFS (Virtual Filesystem)       ║
╠═══════════════════════════════════════════╣
║  Input:    /tmp/crom_sov_419007/vault.crom ║
║  Mount:    /tmp/crom_sov_419007/mnt       ║
║  Codebook: /tmp/crom_sov_419007/sov.cromdb ║
╚═══════════════════════════════════════════╝
✔ CROM Virtual Filesystem montado com sucesso!
  Arquivo:  /tmp/crom_sov_419007/vault.crom
  Ponto:    /tmp/crom_sov_419007/mnt
  Codebook: /tmp/crom_sov_419007/sov.cromdb
  Soberania: Watcher ativo (codebook + signals)
Pressione Ctrl+C para desmontar...
vfs: read error at off=0 len=131072: vfs: lookup codeword 12094627905536: codebook: lookup out of bounds: id=12094627905536, count=8192
vfs: read error at off=0 len=4096: vfs: lookup codeword 12094627905536: codebook: lookup out of bounds: id=12094627905536, count=8192
cat: /tmp/crom_sov_419007/mnt/vaultvfs: read error at off=131072 len=131072: vfs: lookup codeword 562949953421312: codebook: lookup out of bounds: id=562949953421312, count=8192
: Erro de entrada/saída
[FAIL] Não foi possível ler o arquivo do FUSE Mount!

⚡ Sinal recebido (terminated). Desmontando VFS...

```

## 4. Fuzziness Diff (LSH Clones)

```text
img.00.raw - Tamanho: 1048576 bytes
img.05.raw - Tamanho: 1048576 bytes
img.10.raw - Tamanho: 1048576 bytes
```

