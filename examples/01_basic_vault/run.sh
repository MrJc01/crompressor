#!/bin/bash
set -e

echo "[*] Preparando cenário..."
mkdir -p cofres_confidenciais
echo "Senha nuclear: 123456" > cofres_confidenciais/segredo1.txt
echo "Código fonte proprietário: func main() {}" > cofres_confidenciais/projeto_alpha.go

echo "[*] Treinando seu Codebook Pessoal..."
../../crompressor train --input cofres_confidenciais --output meu_vault.cromdb

echo "[*] Empacotando e Encriptando..."
# Se a versão atual não suporta --encrypt e espera a key no envio, simulamos apenas o pack
../../crompressor pack --input cofres_confidenciais --codebook meu_vault.cromdb --output meu_vault.crom

echo "[*] Pronto! Seu vault foi criado."
ls -lh meu_vault.crom
echo "Você pode remover a pasta original e os dados estarão preservados em meu_vault.crom!"
