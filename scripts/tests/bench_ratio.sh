#!/bin/bash
# bench_ratio.sh: O Analista de Eficiência
# Gera diferentes tipos de dados e os comprime para avaliar o Ratio e Hit Rate.

set -e

WORKSPACE="/tmp/crom_bench_$$"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

# Dados simulados
echo "[*] Gerando dados de teste..."
dd if=/dev/urandom of=random.bin bs=1M count=5 2>/dev/null
# Cria um arquivo de texto com muita redundância
for i in {1..50000}; do echo "The quick brown fox jumps over the lazy dog. CROMpressor test line $i" >> text.txt; done
# Simula código com alguma repetição
for i in {1..10000}; do echo "func TestData$i() { return \"data_chunk_repeated_pattern\" }" >> code.go; done

# Assume-se que o CLI do crom já está no PATH ou no dir pai do projeto
CROM_BIN="/home/j/Área de trabalho/crompressor/crompressor"
CB_PATH="global_bench.cromdb"

echo "[*] Treinando Codebook Global..."
"$CROM_BIN" train --input . --output "$CB_PATH" 2>/dev/null || true

echo "[*] Comprimindo Arquivos..."
# Binário
"$CROM_BIN" pack --input random.bin --codebook "$CB_PATH" --output random.bin.crom 2>/dev/null || true
# Texto
"$CROM_BIN" pack --input text.txt --codebook "$CB_PATH" --output text.txt.crom 2>/dev/null || true
# Código
"$CROM_BIN" pack --input code.go --codebook "$CB_PATH" --output code.go.crom 2>/dev/null || true

echo "ARQUIVO,TAMANHO_ORIG(bytes),TAMANHO_CROM(bytes)"

for f in random.bin text.txt code.go; do
    if [ -f "$f" ] && [ -f "${f}.crom" ]; original=$(stat -c%s "$f") && crom=$(stat -c%s "${f}.crom"); then
        echo "$f,$original,$crom"
    else
        echo "$f,ERROR,ERROR"
    fi
done

# Cleanup
rm -rf "$WORKSPACE"
