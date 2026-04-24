#!/bin/bash
# ==============================================================
# Script de Preparação: TLauncher Client via Crompressor VFS
# ==============================================================
set -e

# Pegando o caminho absoluto do diretório de trabalho (evitando problemas com espaços)
BASE_DIR="$(pwd)"
WORK_DIR="$BASE_DIR/trabalho/minecraft_client/tlauncher"
CROM_BIN="$BASE_DIR/crompressor-novo"

mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

echo "📥 [FASE 1] Baixando TLauncher oficial para Linux..."
if [ ! -f "TLauncher.zip" ]; then
    wget -q --show-progress -O TLauncher.zip https://tlauncher.org/download/linux
fi

echo "📦 [FASE 1] Extraindo executável..."
unzip -o TLauncher.zip
mv *.jar tlauncher.jar 2>/dev/null || true

echo "--------------------------------------------------------"
echo "⚠️  [ATENÇÃO - BOOT INICIAL]"
echo "Agora você deve abrir o TLauncher UMA VEZ manualmente."
echo "1. Escolha seu nome de usuário."
echo "2. Selecione a versão 1.20.4."
echo "3. Clique em 'Instalar' e espere baixar os assets (skins, sons, chunks)."
echo ""
echo "COMANDO PARA ABRIR:"
echo "java -jar '$WORK_DIR/tlauncher.jar'"
echo ""
echo "Após o download terminar e o jogo abrir, FECHE O JOGO."
echo "Então, me avise para eu rodar a [FASE 3] de Consolidação CROM VFS."
echo "--------------------------------------------------------"
