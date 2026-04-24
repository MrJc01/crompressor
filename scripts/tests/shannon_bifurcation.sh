#!/bin/bash
set -e

echo "╔═══════════════════════════════════════════════╗"
echo "║  TESTE 5: BIFURCAÇÃO DE SHANNON (Lossy/Lossless)║"
echo "╚═══════════════════════════════════════════════╝"

CROM="./bin/crompressor"
INPUT="testdata/real_world/random.bin"
VAULT_OUT="testdata/real_world/random_vault.crom"
EDGE_OUT="testdata/real_world/random_edge.crom"

if [ ! -f "$INPUT" ]; then
    echo "Gerando arquivo ruidoso..."
    mkdir -p testdata/real_world
    dd if=/dev/urandom of="$INPUT" bs=1M count=2 2>/dev/null
fi

echo "→ Treinando Codebook Rápido..."
$CROM train --input testdata/real_world --output testdata/real_world/bifurcation.cromdb --size 256 > /dev/null

echo "→ Compressão Modo VAULT (Lossless E2E)"
$CROM pack --mode vault --input "$INPUT" --output "$VAULT_OUT" --codebook testdata/real_world/bifurcation.cromdb > /dev/null
VAULT_SIZE=$(stat -c%s "$VAULT_OUT")

echo "→ Compressão Modo EDGE (Lossy)"
$CROM pack --mode edge --input "$INPUT" --output "$EDGE_OUT" --codebook testdata/real_world/bifurcation.cromdb > /dev/null
EDGE_SIZE=$(stat -c%s "$EDGE_OUT")

echo "====================================="
echo "Tamanho Original: $(stat -c%s "$INPUT") bytes"
echo "Tamanho VAULT:    $VAULT_SIZE bytes"
echo "Tamanho EDGE:     $EDGE_SIZE bytes"
echo "====================================="

if [ "$EDGE_SIZE" -lt "$VAULT_SIZE" ]; then
    echo "✅ Sucesso: Modo EDGE descartou o delta e gerou arquivo menor!"
else
    echo "❌ Falha: Modo EDGE não gerou arquivo menor que VAULT."
    exit 1
fi

echo "→ Validando Reconstrução do VAULT"
$CROM unpack --input "$VAULT_OUT" --output "testdata/real_world/random_restored.bin" --codebook testdata/real_world/bifurcation.cromdb > /dev/null
$CROM verify --original "$INPUT" --restored "testdata/real_world/random_restored.bin" > /dev/null

echo "✅ Teste da Bifurcação de Shannon Finalizado com Sucesso!"
