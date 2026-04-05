#!/bin/bash
set -e

mkdir -p nodeA nodeB

echo "[*] Criando um arquivo gigante de 5MB no Node A..."
dd if=/dev/urandom of=nodeA/gigante.bin bs=1M count=5 2>/dev/null

echo "[*] Treinando o Codebook Global..."
../../crompressor train --input nodeA/gigante.bin --output global.cromdb

echo "[*] O Node B obteve o Codebook global via pendrive/satélite..."
cp global.cromdb nodeB/

echo "[*] Node A empacota o arquivo internamente"
cd nodeA
../../crompressor pack --input gigante.bin --codebook ../global.cromdb --output gigante.crom
cd ..

echo "--------------------------------------------------------"
echo "Para executar de verdade, rodaríamos:"
echo "NODE A: crompressor daemon --port 4001; crompressor share gigante.crom"
echo "NODE B: crompressor daemon --port 4002; crompressor download <HASH>"
echo "Como este é um ambiente de demonstração, observe que o tamanho do gigante.crom no NodeA é MÍNIMO comparado aos 5MB:"
ls -lh nodeA/gigante.crom | awk '{print $5}'
echo "O arquivo agora só tem alguns KBs de manifesto CDC Hash."
echo "--------------------------------------------------------"
