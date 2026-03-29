#!/bin/bash
# ==============================================================================
# 🖼️ Pesquisa 06 — Script 00: Geração de Dataset Controlado
# Gera imagens-fonte via ImageMagick e converte para todos os 7 formatos.
# ==============================================================================

set -e
source "$(dirname "$0")/utils.sh"

log_phase "FASE 0: GERAÇÃO DE DATASET CONTROLADO"

SOURCE_DIR="$DATASETS/source"

# --------------------------------------------------------------------------
# PASSO 1: Gerar 5 imagens-fonte (256x144) com padrões distintos
# --------------------------------------------------------------------------
log_info "Gerando 5 imagens-fonte controladas (256x144)..."

# 1. photo: gradiente complexo simulando fotografia (fundo natural)
convert -size 256x144 \
    \( xc: +noise Random -blur 0x2 \) \
    \( -size 256x144 gradient:#2E86AB-#A23B72 \) \
    -compose Overlay -composite \
    -modulate 120,130,100 \
    "$SOURCE_DIR/photo_01.png"
log_ok "photo_01.png — Gradiente naturalístico com noise"

# 2. screenshot: blocos sólidos de cores simulando UI
convert -size 256x144 xc:#1a1a2e \
    -fill '#16213e' -draw "rectangle 0,0 300,1080" \
    -fill '#0f3460' -draw "rectangle 0,0 1920,60" \
    -fill '#e94560' -draw "rectangle 320,80 920,120" \
    -fill '#ffffff' -draw "rectangle 320,140 800,160" \
    -fill '#333333' -draw "rectangle 320,180 1200,600" \
    -fill '#444444' -draw "rectangle 340,200 1180,580" \
    -fill '#e94560' -draw "circle 1800,40 1820,40" \
    -fill '#53a653' -draw "circle 1760,40 1780,40" \
    -fill '#ffffff' -pointsize 28 -annotate +340+240 "Dashboard Analytics v2.1" \
    -fill '#aaaaaa' -pointsize 14 -annotate +340+280 "Sistema de Monitoramento | Crompressor Research Lab" \
    "$SOURCE_DIR/screenshot_01.png"
log_ok "screenshot_01.png — UI Dashboard simulado"

# 3. artwork: padrão fractal/artístico
convert -size 256x144 plasma:magenta-cyan \
    -blur 0x1 -sharpen 0x3 \
    -modulate 100,150,100 \
    "$SOURCE_DIR/artwork_01.png"
log_ok "artwork_01.png — Plasma fractal artístico"

# 4. medical: padrão de alto contraste simulando imagem técnica
convert -size 256x144 xc:black \
    -fill white -draw "circle 960,540 960,200" \
    -fill '#808080' -draw "circle 960,540 960,340" \
    -fill '#404040' -draw "circle 800,400 800,350" \
    -fill '#c0c0c0' -draw "circle 1100,600 1100,560" \
    -blur 0x4 \
    -fill white -pointsize 16 -annotate +20+20 "SCAN-001 | 2026-03-29" \
    "$SOURCE_DIR/medical_01.png"
log_ok "medical_01.png — Simulação de scan médico"

# 5. texture: padrão de textura repetitiva (game tile)
convert -size 256x144 \
    \( -size 64x64 xc: +noise Uniform -normalize \) \
    -virtual-pixel tile -filter Cubic -resize 256x144! \
    -modulate 80,60,100 \
    -fill '#8B4513' -colorize 30% \
    "$SOURCE_DIR/texture_01.png"
log_ok "texture_01.png — Textura tile repetitiva"

# 6. Gerar mais 3 variantes para ter massa suficiente para train/test split
for variant in 02 03; do
    convert "$SOURCE_DIR/photo_01.png" -rotate $((RANDOM % 10 - 5)) -modulate $((90 + RANDOM % 20)),$((90 + RANDOM % 20)),100 "$SOURCE_DIR/photo_${variant}.png"
    convert "$SOURCE_DIR/screenshot_01.png" -negate -modulate $((80 + RANDOM % 40)),100,100 "$SOURCE_DIR/screenshot_${variant}.png"
    convert "$SOURCE_DIR/artwork_01.png" -swirl $((RANDOM % 60)) -modulate 100,$((80 + RANDOM % 40)),100 "$SOURCE_DIR/artwork_${variant}.png"
    convert "$SOURCE_DIR/medical_01.png" -blur 0x$((1 + RANDOM % 3)) -brightness-contrast $((RANDOM % 20 - 10))x0 "$SOURCE_DIR/medical_${variant}.png"
    convert "$SOURCE_DIR/texture_01.png" -rotate $((90 * (RANDOM % 4))) -modulate 100,100,$((80 + RANDOM % 40)) "$SOURCE_DIR/texture_${variant}.png"
done

TOTAL_SOURCES=$(ls "$SOURCE_DIR"/*.png 2>/dev/null | wc -l)
log_ok "Total de imagens-fonte geradas: $TOTAL_SOURCES"

# --------------------------------------------------------------------------
# PASSO 2: Converter para todos os 7 formatos
# --------------------------------------------------------------------------
log_info "Convertendo para 7 formatos..."

INVENTORY_CSV="$DATASETS/inventory.csv"
csv_init "$INVENTORY_CSV" "formato,arquivo,tamanho_bytes,largura,altura,canais,set"

for src in "$SOURCE_DIR"/*.png; do
    BASENAME=$(basename "$src" .png)
    
    # Determinar train ou test (últimos 20% = variantes 03 vão para test)
    if [[ "$BASENAME" == *"_03" ]]; then
        SPLIT="test"
    else
        SPLIT="train"
    fi
    
    # BMP (24-bit uncompressed)
    convert "$src" "BMP3:$DATASETS/bmp/${SPLIT}/${BASENAME}.bmp"
    
    # PNG (compressão deflate padrão)
    cp "$src" "$DATASETS/png/${SPLIT}/${BASENAME}.png"
    
    # JPEG (Quality 95 — alta fidelidade)
    convert "$src" -quality 95 "$DATASETS/jpg/${SPLIT}/${BASENAME}.jpg"
    
    # WebP (lossless)
    convert "$src" -define webp:lossless=true "$DATASETS/webp/${SPLIT}/${BASENAME}.webp"
    
    # GIF (256 cores — limitação do formato)
    convert "$src" -colors 256 "$DATASETS/gif/${SPLIT}/${BASENAME}.gif"
    
    # TIFF (uncompressed)
    convert "$src" -compress None "$DATASETS/tiff/${SPLIT}/${BASENAME}.tiff"
    
    # SVG (text-based trace via ImageMagick — básico mas funcional)
    convert "$src" -resize 256x144 "$DATASETS/svg/${SPLIT}/${BASENAME}.svg" 2>/dev/null || \
    convert "$src" -resize 256x144 "pnm:-" | convert "pnm:-" "$DATASETS/svg/${SPLIT}/${BASENAME}.svg" 2>/dev/null || \
    # Fallback: gerar SVG manual com base64 embutido
    {
        B64=$(base64 -w0 "$src")
        cat > "$DATASETS/svg/${SPLIT}/${BASENAME}.svg" <<SVGEOF
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="1920" height="1080">
<image width="1920" height="1080" xlink:href="data:image/png;base64,${B64}"/>
</svg>
SVGEOF
    }
    
    log_info "  Convertido: $BASENAME → 7 formatos (set: $SPLIT)"
done

# --------------------------------------------------------------------------
# PASSO 3: Gerar inventário com métricas
# --------------------------------------------------------------------------
log_info "Gerando inventário de dataset..."

for fmt in "${FORMATS[@]}"; do
    for split in train test; do
        dir="$DATASETS/$fmt/$split"
        if [ -d "$dir" ] && [ "$(ls -A "$dir" 2>/dev/null)" ]; then
            for f in "$dir"/*; do
                SIZE=$(stat -c%s "$f")
                # Obter dimensões (se possível)
                DIMS=$(identify -format "%wx%h" "$f" 2>/dev/null || echo "N/A")
                W=$(echo "$DIMS" | cut -dx -f1)
                H=$(echo "$DIMS" | cut -dx -f2)
                csv_append "$INVENTORY_CSV" "$fmt,$(basename "$f"),$SIZE,$W,$H,RGB,$split"
            done
        fi
    done
done

log_ok "Inventário salvo em: $INVENTORY_CSV"

# --------------------------------------------------------------------------
# PASSO 4: Resumo
# --------------------------------------------------------------------------
log_phase "RESUMO DO DATASET"
echo ""
for fmt in "${FORMATS[@]}"; do
    TRAIN_COUNT=$(ls "$DATASETS/$fmt/train/" 2>/dev/null | wc -l)
    TEST_COUNT=$(ls "$DATASETS/$fmt/test/" 2>/dev/null | wc -l)
    TRAIN_SIZE=$(du -sb "$DATASETS/$fmt/train/" 2>/dev/null | cut -f1)
    TEST_SIZE=$(du -sb "$DATASETS/$fmt/test/" 2>/dev/null | cut -f1)
    printf "  %-6s │ Train: %2d files (%s) │ Test: %2d files (%s)\n" \
        "$fmt" "$TRAIN_COUNT" "$(fmt_bytes ${TRAIN_SIZE:-0})" "$TEST_COUNT" "$(fmt_bytes ${TEST_SIZE:-0})"
done
echo ""

log_ok "Dataset gerado com sucesso!"
