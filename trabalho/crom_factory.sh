#!/bin/bash
# ==============================================================
# CROM FACTORY: Gerador de Arquivos .crom para os Laboratórios
# Uso correto da API: train -i DIRETÓRIO | pack -i ARQUIVO -c CROMDB
# ==============================================================

BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TRABALHO_DIR="$BASE_DIR/trabalho"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ ! -f "$CROM_BIN" ]; then
    echo -e "${RED}Erro: Binário não encontrado em $CROM_BIN${NC}"
    exit 1
fi

cd "$TRABALHO_DIR"

for dir in app_*; do
    echo -e "\n${YELLOW}🏗️  Processando: $dir${NC}"
    cd "$dir"

    # 1. Criar diretório com dados de treino
    rm -rf _train_data
    mkdir -p _train_data

    # Gerar conteúdo sintético relevante ao tema
    echo "{\"app\": \"$dir\", \"version\": \"1.0\", \"created\": \"$(date -Iseconds)\"}" > _train_data/manifest.json
    echo "IDENTIDADE SOBERANA: $dir — Motor Crompressor V24+" > _train_data/identity.txt
    dd if=/dev/urandom of=_train_data/sample.bin bs=4K count=8 status=none
    # Duplicar para dar padrões ao codebook
    cp _train_data/sample.bin _train_data/sample_copy.bin

    # 2. Train: gera o .cromdb a partir do DIRETÓRIO (ou SQUASH)
    echo "  📦 [Camada 1] Escrevendo SquashFS Dummy base..."
    mksquashfs _train_data _train_data.sqsh -noI -noD -noX -noF -no-xattrs >/dev/null 2>&1

    echo "  📚 Treinando codebook..."
    "$CROM_BIN" train -i _train_data.sqsh -o ./${dir}.cromdb -s 256 >/dev/null 2>&1 || {
        echo -e "${RED}  ❌ Train falhou para $dir${NC}"
        cd ..
        continue
    }

    # 3. Pack: empacota o SQUASH na malha FUSE CASCADING
    echo "  📦 Empacotando Tri-Camada FUSE..."
    "$CROM_BIN" pack -i _train_data.sqsh -o ./${dir}.crom -c ./${dir}.cromdb >/dev/null 2>&1 || {
        echo -e "${RED}  ❌ Pack falhou para $dir${NC}"
        cd ..
        continue
    }

    # Limpeza dos temporários
    rm -rf _train_data _train_data.sqsh

    echo -e "${GREEN}  ✅ ${dir}.crom + ${dir}.cromdb criados${NC}"
    cd ..
done

echo -e "\n${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🎯 Fábrica finalizada. Verifique com:${NC}"
echo "   find trabalho -name '*.crom' -type f"
