#!/bin/bash
# ==============================================================================
# 🔀 Pesquisa 06 — Script 03: Experimento B — Matriz Cruzada (Cross Brain)
# Cada formato testado contra TODOS os 7 cérebros = 49 combinações.
# ==============================================================================

source "$(dirname "$0")/utils.sh"

log_phase "EXPERIMENTO B: MATRIZ CRUZADA (7×7 = 49 COMBINAÇÕES)"

OUT_DIR="$RESULTS/cross_brain"
METRICS="$OUT_DIR/matrix.csv"
csv_init "$METRICS" "formato,brain,orig_bytes,crom_bytes,ratio_pct,saving_pct,time_ms,verify"

TOTAL_COMBOS=0
PASS_COUNT=0

for fmt in "${FORMATS[@]}"; do
    for brain_fmt in "${FORMATS[@]}"; do
        BRAIN="$BRAINS/brain_${brain_fmt}.cromdb"
        [ -f "$BRAIN" ] || continue
        
        # Usar primeiro arquivo de treino como representante
        SRC=$(ls "$DATASETS/${fmt}/train"/* 2>/dev/null | head -1)
        [ -f "$SRC" ] || continue
        
        CROM_OUT="$OUT_DIR/${fmt}_via_${brain_fmt}.crom"
        RESTORED="/tmp/crom_cross_$$_${fmt}_${brain_fmt}"
        
        # Pack
        local_start=$(timer_start)
        $BIN pack -i "$SRC" -c "$BRAIN" -o "$CROM_OUT" > /dev/null 2>&1
        PACK_EXIT=$?
        ELAPSED=$(timer_elapsed_ms "$local_start")
        
        if [ $PACK_EXIT -ne 0 ] || [ ! -f "$CROM_OUT" ]; then
            csv_append "$METRICS" "$fmt,$brain_fmt,$(stat -c%s "$SRC"),0,0,0,$ELAPSED,PACK_FAIL"
            TOTAL_COMBOS=$((TOTAL_COMBOS + 1))
            continue
        fi
        
        ORIG_SIZE=$(stat -c%s "$SRC")
        CROM_SIZE=$(stat -c%s "$CROM_OUT")
        RATIO=$(echo "scale=2; $CROM_SIZE / $ORIG_SIZE * 100" | bc)
        SAVING=$(echo "scale=2; 100 - $RATIO" | bc)
        
        # Unpack + Verify
        $BIN unpack -i "$CROM_OUT" -c "$BRAIN" -o "$RESTORED" > /dev/null 2>&1
        UNPACK_EXIT=$?
        
        VERIFY_RESULT="UNPACK_FAIL"
        if [ $UNPACK_EXIT -eq 0 ] && [ -f "$RESTORED" ]; then
            HASH_ORIG=$(sha256sum "$SRC" | cut -d' ' -f1)
            HASH_REST=$(sha256sum "$RESTORED" | cut -d' ' -f1)
            if [ "$HASH_ORIG" == "$HASH_REST" ]; then
                VERIFY_RESULT="PASS"
                PASS_COUNT=$((PASS_COUNT + 1))
            else
                VERIFY_RESULT="HASH_MISMATCH"
            fi
        fi
        rm -f "$RESTORED"
        
        csv_append "$METRICS" "$fmt,$brain_fmt,$ORIG_SIZE,$CROM_SIZE,$RATIO,$SAVING,$ELAPSED,$VERIFY_RESULT"
        TOTAL_COMBOS=$((TOTAL_COMBOS + 1))
        
        # Indicador visual
        if [ "$fmt" == "$brain_fmt" ]; then
            MARKER="🟢"
        elif [ "$VERIFY_RESULT" == "PASS" ]; then
            MARKER="🟡"
        else
            MARKER="🔴"
        fi
        
        log_info "  $MARKER $fmt → brain_$brain_fmt: ratio=${RATIO}% saving=${SAVING}% [$VERIFY_RESULT]"
    done
done

# --------------------------------------------------------------------------
# Gerar Heatmap textual
# --------------------------------------------------------------------------
log_phase "HEATMAP DA MATRIZ CRUZADA (SAVING %)"
echo ""
printf "  %-6s │" "FMT↓/BR→"
for brain_fmt in "${FORMATS[@]}"; do
    printf " %-7s│" "$brain_fmt"
done
echo ""
printf "  %-6s─┼" "──────"
for brain_fmt in "${FORMATS[@]}"; do
    printf "─%-7s┼" "───────"
done
echo ""

for fmt in "${FORMATS[@]}"; do
    printf "  %-6s │" "$fmt"
    for brain_fmt in "${FORMATS[@]}"; do
        SAVING=$(grep "^$fmt,$brain_fmt," "$METRICS" | head -1 | awk -F',' '{print $6}')
        VERIFY=$(grep "^$fmt,$brain_fmt," "$METRICS" | head -1 | awk -F',' '{print $8}')
        if [ -z "$SAVING" ]; then
            printf " %-7s│" "  —  "
        elif [ "$VERIFY" == "PASS" ] && [ "$fmt" == "$brain_fmt" ]; then
            printf " \033[1;32m%5s%%\033[0m │" "$SAVING"
        elif [ "$VERIFY" == "PASS" ]; then
            printf " \033[1;33m%5s%%\033[0m │" "$SAVING"
        else
            printf " \033[0;31m%5s%%\033[0m │" "$SAVING"
        fi
    done
    echo ""
done
echo ""

log_ok "Matriz cruzada: $TOTAL_COMBOS combinações | $PASS_COUNT/$TOTAL_COMBOS PASS lossless"
log_ok "Resultados em: $METRICS"
