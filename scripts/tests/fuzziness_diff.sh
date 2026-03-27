#!/bin/bash
# fuzziness_diff.sh: O Analista de Clones
# Gera arquivos empacotados usando diferentes níveis de fuzziness para provar LSH.

set -e

WORKSPACE="/tmp/crom_fuzz_$$"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

CROM_BIN="/home/j/Área de trabalho/crompessor/crom"
CB_PATH="fuzz.cromdb"

echo "[*] Criando imagem/binário de testes..."
dd if=/dev/urandom of=img.raw bs=1M count=1 2>/dev/null

echo "[*] Treinando Codebook Fuzzing..."
"$CROM_BIN" train --input img.raw --output "$CB_PATH" 2>/dev/null || true

echo "[*] Empacotando arquivo original..."
"$CROM_BIN" pack --input img.raw --codebook "$CB_PATH" --output img.crom 2>/dev/null || true

echo "[*] Fuzziness 0.00 (Exato)"
"$CROM_BIN" unpack --input img.crom --codebook "$CB_PATH" --output img.00.raw --fuzziness 0.00 2>/dev/null || true

echo "[*] Fuzziness 0.05"
"$CROM_BIN" unpack --input img.crom --codebook "$CB_PATH" --output img.05.raw --fuzziness 0.05 2>/dev/null || true

echo "[*] Fuzziness 0.10"
"$CROM_BIN" unpack --input img.crom --codebook "$CB_PATH" --output img.10.raw --fuzziness 0.10 2>/dev/null || true

echo "RESULTADO_FUZZINESS"
for restored in img.00.raw img.05.raw img.10.raw; do
    if [ -f "$restored" ]; then
        sz=$(stat -c%s "$restored")
        echo "$restored - Tamanho: $sz bytes"
    else
        echo "$restored - ERROR"
    fi
done

# As the files are compressed differently because chunks get merged or replaced with nearest neighbors, sizes should vary slightly

# Cleanup
rm -rf "$WORKSPACE"
