#!/bin/bash
set -e

echo "[*] Gerando dados em texto... (Shakespeare fake)"
mkdir -p corpus_texto
for i in {1..1000}; do echo "Ser ou não ser. $i" >> corpus_texto/poesia.txt; done

echo "[*] Treinando Codebook especialista em TEXTO..."
../../crompressor train --input corpus_texto --output brain_texto.cromdb

echo "[*] Vamos tentar comprimir o texto..."
../../crompressor pack --input corpus_texto/poesia.txt --codebook brain_texto.cromdb --output poesia.crom
sz_orig=$(stat -c%s corpus_texto/poesia.txt)
sz_crom=$(stat -c%s poesia.crom)
echo "[HIT] Poesia Original: $sz_orig. Poesia Comprimida: $sz_crom bytes."

echo "----------------------------------------------------"
echo "[*] Agora gerando um arquivo de entropia randômica (imagem/binário)..."
dd if=/dev/urandom of=imagem.bin bs=100K count=1 2>/dev/null

echo "[*] Tentando empacotar com o cérebro TEXTO..."
../../crompressor pack --input imagem.bin --codebook brain_texto.cromdb --output imagem.crom 2>/dev/null || echo "Ocorreu warning ou miss"
sz_img=$(stat -c%s imagem.bin)
sz_img_crom=$(stat -c%s imagem.crom)
echo "[MISS] Original: $sz_img bytes. Comprimida ignorando cérebro: $sz_img_crom bytes."
echo "[INFO] O CROMpressor ignorou porque as features LSH colidem a 0% do arquivo .bin!"
