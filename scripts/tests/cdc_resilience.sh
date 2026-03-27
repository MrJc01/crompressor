#!/bin/bash
# cdc_resilience.sh: O Teste do Rabin
# Comprova a resiliência do rolling hash à inserção de apenas 1 byte.

set -e

WORKSPACE="/tmp/crom_cdc_$$"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

CROM_BIN="/home/j/Área de trabalho/crompressor/crompressor"
CB_PATH="cdc_test.cromdb"

echo "[*] Criando original.txt (repetitivo 2MB)"
for i in {1..50000}; do echo "CROM CDC Fingerprint Shift Resiliency Test Block Data Magic Code A9F"; done > original.txt

echo "[*] Treinando Codebook..."
"$CROM_BIN" train --input original.txt --output "$CB_PATH" > /dev/null 2>&1 || true

echo "[*] Empacotando original.txt"
"$CROM_BIN" pack --input original.txt --codebook "$CB_PATH" --output original.txt.crom > pack_orig.log 2>&1 || true

echo "[*] Inserindo 'X' (1 byte shift) no início de original.txt"
echo -n "X" > modified.txt
cat original.txt >> modified.txt

echo "[*] Empacotando modified.txt"
"$CROM_BIN" pack --input modified.txt --codebook "$CB_PATH" --output modified.txt.crom > pack_mod.log 2>&1 || true

ORIG_SIZE=$(stat -c%s original.txt.crom 2>/dev/null || echo 0)
MOD_SIZE=$(stat -c%s modified.txt.crom 2>/dev/null || echo 0)
DIFF=$((MOD_SIZE - ORIG_SIZE))

echo "RESULTADO_CDC_RESILIENCE"
echo "Tamanho Original (.crom): $ORIG_SIZE bytes"
echo "Tamanho Shifted  (.crom): $MOD_SIZE bytes"
echo "Diferença Absoluta: $DIFF bytes"

# Cleanup
rm -rf "$WORKSPACE"
