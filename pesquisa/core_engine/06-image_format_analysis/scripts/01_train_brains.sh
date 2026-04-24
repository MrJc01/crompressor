#!/bin/bash
# ==============================================================================
# 🧠 Pesquisa 06 — Script 01: Treinamento de Cérebros (Codebooks)
# Treina 7 cérebros específicos + 1 universal
# ==============================================================================

set -e
source "$(dirname "$0")/utils.sh"

log_phase "FASE 2: TREINAMENTO DE CÉREBROS (CODEBOOKS)"

TRAIN_LOG="$BRAINS/training_metrics.csv"
csv_init "$TRAIN_LOG" "brain,format,size_patterns,time_ms,cromdb_bytes,num_files_trained"

train_brain() {
    local name=$1
    local input_dir=$2
    local output="$BRAINS/${name}.cromdb"
    local fmt=$3
    
    local num_files=$(find "$input_dir" -type f 2>/dev/null | wc -l)
    
    if [ "$num_files" -eq 0 ]; then
        log_warn "Sem arquivos em $input_dir — pulando $name"
        return
    fi
    
    log_info "Treinando $name ($num_files arquivos de $input_dir)..."
    
    local start=$(timer_start)
    $BIN train -i "$input_dir" -o "$output" --size 8192 --concurrency 4 2>&1 | tail -5
    local elapsed=$(timer_elapsed_ms "$start")
    
    local db_size=$(stat -c%s "$output" 2>/dev/null || echo 0)
    
    csv_append "$TRAIN_LOG" "$name,$fmt,8192,$elapsed,$db_size,$num_files"
    log_ok "$name treinado em ${elapsed}ms ($(fmt_bytes $db_size))"
}

# --------------------------------------------------------------------------
# Treinar 7 cérebros específicos (apenas com dados de TREINO)
# --------------------------------------------------------------------------
for fmt in "${FORMATS[@]}"; do
    train_brain "brain_${fmt}" "$DATASETS/${fmt}/train" "$fmt"
done

# --------------------------------------------------------------------------
# Treinar cérebro UNIVERSAL (todos os formatos juntos)
# --------------------------------------------------------------------------
log_info "Preparando diretório de treino universal..."
UNIVERSAL_TRAIN="/tmp/crom_universal_train"
rm -rf "$UNIVERSAL_TRAIN"
mkdir -p "$UNIVERSAL_TRAIN"

for fmt in "${FORMATS[@]}"; do
    if [ -d "$DATASETS/${fmt}/train" ] && [ "$(ls -A "$DATASETS/${fmt}/train" 2>/dev/null)" ]; then
        for f in "$DATASETS/${fmt}/train"/*; do
            cp "$f" "$UNIVERSAL_TRAIN/$(basename "$f")"
        done
    fi
done

train_brain "brain_universal" "$UNIVERSAL_TRAIN" "universal"
rm -rf "$UNIVERSAL_TRAIN"

# --------------------------------------------------------------------------
# Resumo
# --------------------------------------------------------------------------
log_phase "RESUMO DE CÉREBROS TREINADOS"
echo ""
printf "  %-20s │ %-12s │ %-10s\n" "BRAIN" "TAMANHO" "TEMPO"
printf "  %-20s─┼─%-12s─┼─%-10s\n" "────────────────────" "────────────" "──────────"
for f in "$BRAINS"/*.cromdb; do
    NAME=$(basename "$f" .cromdb)
    SIZE=$(stat -c%s "$f")
    printf "  %-20s │ %-12s │ ver CSV\n" "$NAME" "$(fmt_bytes $SIZE)"
done
echo ""

log_ok "Todos os cérebros treinados com sucesso!"
echo ""
cat "$TRAIN_LOG"
