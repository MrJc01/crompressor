#!/bin/bash
# ==============================================================================
# 🔮 Pesquisa 06 — Script 04: Experimento C — Inferência Pós-Treino
# Usa imagens NOVAS (20% reservado) para testar generalização do cérebro.
# ==============================================================================

source "$(dirname "$0")/utils.sh"

log_phase "EXPERIMENTO C: INFERÊNCIA PÓS-TREINO (IMAGENS NOVAS)"

OUT_DIR="$RESULTS/inference"
METRICS="$OUT_DIR/degradation.csv"
csv_init "$METRICS" "formato,arquivo,orig_bytes,crom_bytes,ratio_pct,saving_pct,time_ms,verify,ratio_treino_avg,degradation_pct"

SAME_BRAIN_METRICS="$RESULTS/same_brain/metrics.csv"

for fmt in "${FORMATS[@]}"; do
    BRAIN="$BRAINS/brain_${fmt}.cromdb"
    TEST_DIR="$DATASETS/${fmt}/test"
    
    [ -f "$BRAIN" ] || continue
    [ -d "$TEST_DIR" ] || continue
    ls "$TEST_DIR"/* > /dev/null 2>&1 || continue
    
    # Ratio de treino como baseline
    TRAIN_AVG_RATIO=$(grep "^$fmt," "$SAME_BRAIN_METRICS" 2>/dev/null | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f", sum/n; else print "100"}')
    
    log_info "Testando inferência para $fmt (baseline treino: ${TRAIN_AVG_RATIO}%)..."
    
    files=("$TEST_DIR/"*)
    for src in "${files[@]}"; do
        [ -f "$src" ] || continue
        BASENAME=$(basename "$src")
        CROM_OUT="$OUT_DIR/${fmt}_infer_${BASENAME}.crom"
        RESTORED="/tmp/crom_infer_$$_${BASENAME}"
        
        # Pack com cérebro treinado (que NUNCA viu esta imagem)
        local_start=$(timer_start)
        $BIN pack -i "$src" -c "$BRAIN" -o "$CROM_OUT" > /dev/null 2>&1
        PACK_EXIT=$?
        ELAPSED=$(timer_elapsed_ms "$local_start")
        
        if [ $PACK_EXIT -ne 0 ] || [ ! -f "$CROM_OUT" ]; then
            csv_append "$METRICS" "$fmt,$BASENAME,$(stat -c%s "$src"),0,0,0,$ELAPSED,PACK_FAIL,$TRAIN_AVG_RATIO,N/A"
            continue
        fi
        
        ORIG_SIZE=$(stat -c%s "$src")
        CROM_SIZE=$(stat -c%s "$CROM_OUT")
        RATIO=$(echo "scale=2; $CROM_SIZE / $ORIG_SIZE * 100" | bc)
        SAVING=$(echo "scale=2; 100 - $RATIO" | bc)
        
        # Unpack + Verify
        $BIN unpack -i "$CROM_OUT" -c "$BRAIN" -o "$RESTORED" > /dev/null 2>&1
        UNPACK_EXIT=$?
        
        VERIFY_RESULT="UNPACK_FAIL"
        if [ $UNPACK_EXIT -eq 0 ] && [ -f "$RESTORED" ]; then
            HASH_ORIG=$(sha256sum "$src" | cut -d' ' -f1)
            HASH_REST=$(sha256sum "$RESTORED" | cut -d' ' -f1)
            [ "$HASH_ORIG" == "$HASH_REST" ] && VERIFY_RESULT="PASS" || VERIFY_RESULT="HASH_MISMATCH"
        fi
        rm -f "$RESTORED"
        
        # Calcular degradação
        DEGRADATION="N/A"
        if [ "$TRAIN_AVG_RATIO" != "100" ] && [ "$TRAIN_AVG_RATIO" != "0" ]; then
            DEGRADATION=$(echo "scale=2; ($RATIO - $TRAIN_AVG_RATIO) / $TRAIN_AVG_RATIO * 100" | bc 2>/dev/null || echo "N/A")
        fi
        
        csv_append "$METRICS" "$fmt,$BASENAME,$ORIG_SIZE,$CROM_SIZE,$RATIO,$SAVING,$ELAPSED,$VERIFY_RESULT,$TRAIN_AVG_RATIO,$DEGRADATION"
        
        log_info "  $BASENAME: ratio=${RATIO}% (treino=${TRAIN_AVG_RATIO}%) | degradação=${DEGRADATION}% | $VERIFY_RESULT"
    done
done

# --------------------------------------------------------------------------
# Ranking de generalização
# --------------------------------------------------------------------------
log_phase "RANKING DE GENERALIZAÇÃO PÓS-TREINO"
echo ""
printf "  %-6s │ %-15s │ %-15s │ %-15s\n" "FMT" "RATIO TREINO" "RATIO INFER." "DEGRADAÇÃO"
printf "  %-6s─┼─%-15s─┼─%-15s─┼─%-15s\n" "──────" "───────────────" "───────────────" "───────────────"

for fmt in "${FORMATS[@]}"; do
    TRAIN_R=$(grep "^$fmt," "$SAME_BRAIN_METRICS" 2>/dev/null | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f%%", sum/n; else print "N/A"}')
    INFER_R=$(grep "^$fmt," "$METRICS" 2>/dev/null | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f%%", sum/n; else print "N/A"}')
    DEGRAD=$(grep "^$fmt," "$METRICS" 2>/dev/null | awk -F',' '{if($10!="N/A"){sum+=$10; n++}} END {if(n>0) printf "%.2f%%", sum/n; else print "N/A"}')
    printf "  %-6s │ %-15s │ %-15s │ %-15s\n" "$fmt" "$TRAIN_R" "$INFER_R" "$DEGRAD"
done
echo ""

log_ok "Experimento C concluído! Resultados em: $METRICS"
