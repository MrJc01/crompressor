#!/bin/bash
# ==============================================================================
# 🖼️ Pesquisa 06 — Script 00: Geração de Dataset Controlado (INTERNET REAL)
# Baixa 50 imagens exclusivas da web e converte para múltiplos formatos.
# ==============================================================================

source "$(dirname "$0")/utils.sh"

log_phase "FASE 0: DATASET (50 FOTOS VIA PICSUM.PHOTOS)"

SOURCE_DIR="$DATASETS/source"
TOTAL_IMAGES=50
RESOLUCAO="600/600" # 600x600 evita Delta Pool Overflow em raw BMP (fica ~1MB)

# --------------------------------------------------------------------------
# PASSO 1: Baixar imagens-fonte JPEG
# --------------------------------------------------------------------------
log_info "Baixando $TOTAL_IMAGES imagens sementes (${RESOLUCAO}px)..."

for i in $(seq 1 $TOTAL_IMAGES); do
    IMG_IDX=$(printf "%02d" $i)
    SRC_FILE="$SOURCE_DIR/real_$IMG_IDX.jpg"
    
    # Download se não existir
    if [ ! -f "$SRC_FILE" ]; then
        curl -s -L -o "$SRC_FILE" "https://picsum.photos/seed/crompressor_$i/$RESOLUCAO"
    fi
done

TOTAL_SOURCES=$(ls "$SOURCE_DIR"/*.jpg 2>/dev/null | wc -l)
log_ok "Total de imagens-fonte reais baixadas: $TOTAL_SOURCES / $TOTAL_IMAGES"

# --------------------------------------------------------------------------
# PASSO 2: Converter para todos os formatos
# --------------------------------------------------------------------------
log_info "Convertendo fotografias para formatos auditados..."

INVENTORY_CSV="$DATASETS/inventory.csv"
csv_init "$INVENTORY_CSV" "formato,arquivo,tamanho_bytes,largura,altura,canais,set"

# 40 para treino, 10 para teste
for i in $(seq 1 $TOTAL_IMAGES); do
    IMG_IDX=$(printf "%02d" $i)
    src="$SOURCE_DIR/real_$IMG_IDX.jpg"
    BASENAME=$(basename "$src" .jpg)
    
    # Índices maiores que 40 vão para teste
    if [ $i -gt 40 ]; then
        SPLIT="test"
    else
        SPLIT="train"
    fi
    
    # BMP (24-bit uncompressed)
    convert "$src" "BMP3:$DATASETS/bmp/${SPLIT}/${BASENAME}.bmp" 2>/dev/null
    
    # PNG (deflate)
    convert "$src" "$DATASETS/png/${SPLIT}/${BASENAME}.png" 2>/dev/null
    
    # JPEG (Quality 95)
    cp "$src" "$DATASETS/jpg/${SPLIT}/${BASENAME}.jpg"
    
    # WebP (lossy padrão altíssima qualidade ou lossless)
    convert "$src" -define webp:lossless=true "$DATASETS/webp/${SPLIT}/${BASENAME}.webp" 2>/dev/null
    
    # GIF (256 cores)
    convert "$src" -colors 256 "$DATASETS/gif/${SPLIT}/${BASENAME}.gif" 2>/dev/null
    
    # TIFF (uncompressed)
    convert "$src" -compress None "$DATASETS/tiff/${SPLIT}/${BASENAME}.tiff" 2>/dev/null
    
    # SVG (ImageMagick wrapper - explicitamente mantido pelo user)
    B64=$(base64 -w0 "$src")
    cat > "$DATASETS/svg/${SPLIT}/${BASENAME}.svg" <<SVGEOF
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="600" height="600">
<image width="600" height="600" xlink:href="data:image/jpeg;base64,${B64}"/>
</svg>
SVGEOF

    # Apenas loga a cada 10 arquivos para não poluir
    if [ $((i % 10)) -eq 0 ]; then
        log_info "  Progresso: Processado até $BASENAME (set: $SPLIT)"
    fi
done

# --------------------------------------------------------------------------
# PASSO 3: Resumo
# --------------------------------------------------------------------------
log_phase "RESUMO DO DATASET REAL"
echo ""
for fmt in "${FORMATS[@]}"; do
    TRAIN_COUNT=$(ls -1 "$DATASETS/$fmt/train/" 2>/dev/null | wc -l)
    TEST_COUNT=$(ls -1 "$DATASETS/$fmt/test/" 2>/dev/null | wc -l)
    TRAIN_SIZE=$(du -sb "$DATASETS/$fmt/train/" 2>/dev/null | cut -f1)
    TEST_SIZE=$(du -sb "$DATASETS/$fmt/test/" 2>/dev/null | cut -f1)
    printf "  %-6s │ Train: %2d files (%s) │ Test: %2d files (%s)\n" \
        "$fmt" "$TRAIN_COUNT" "$(fmt_bytes ${TRAIN_SIZE:-0})" "$TEST_COUNT" "$(fmt_bytes ${TEST_SIZE:-0})"
done
echo ""

log_ok "Dataset realístico baixado e estruturado com sucesso!"
