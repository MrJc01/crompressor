#!/bin/bash
# ==============================================================================
# 📊 Pesquisa 06 — Script 02: Experimento A — Compressão Nativa (Same Brain)
# Cada formato é comprimido com seu próprio cérebro treinado.
# NOTA: Formatos com grandes delta pools podem falhar no unpack (bug documentado).
# ==============================================================================

source "$(dirname "$0")/utils.sh"

log_phase "EXPERIMENTO A: COMPRESSÃO NATIVA (SAME BRAIN)"

OUT_DIR="$RESULTS/same_brain"
METRICS="$OUT_DIR/metrics.csv"
csv_init "$METRICS" "formato,arquivo,orig_bytes,crom_bytes,ratio_pct,saving_pct,time_ms,verify"

for fmt in "${FORMATS[@]}"; do
    BRAIN="$BRAINS/brain_${fmt}.cromdb"
    
    if [ ! -f "$BRAIN" ]; then
        log_warn "Cérebro brain_${fmt}.cromdb não encontrado — pulando"
        continue
    fi
    
    log_info "Testando formato $fmt com brain_${fmt}..."
    
    files=("$DATASETS/${fmt}/train/"*)
    for src in "${files[@]:0:2}"; do
        [ -f "$src" ] || continue
        BASENAME=$(basename "$src")
        CROM_OUT="$OUT_DIR/${fmt}_${BASENAME}.crom"
        RESTORED="/tmp/crom_restored_$$_${BASENAME}"
        
        # Pack
        local_start=$(timer_start)
        PACK_OUTPUT=$($BIN pack -i "$src" -c "$BRAIN" -o "$CROM_OUT" 2>&1)
        PACK_EXIT=$?
        ELAPSED=$(timer_elapsed_ms "$local_start")
        
        if [ $PACK_EXIT -ne 0 ] || [ ! -f "$CROM_OUT" ]; then
            log_err "  $BASENAME: PACK FALHOU"
            csv_append "$METRICS" "$fmt,$BASENAME,$(stat -c%s "$src"),0,0,0,$ELAPSED,PACK_FAIL"
            continue
        fi
        
        ORIG_SIZE=$(stat -c%s "$src")
        CROM_SIZE=$(stat -c%s "$CROM_OUT")
        RATIO=$(echo "scale=2; $CROM_SIZE / $ORIG_SIZE * 100" | bc)
        SAVING=$(echo "scale=2; 100 - $RATIO" | bc)
        
        # Unpack + Verify
        UNPACK_OUTPUT=$($BIN unpack -i "$CROM_OUT" -c "$BRAIN" -o "$RESTORED" 2>&1)
        UNPACK_EXIT=$?
        
        VERIFY_RESULT="UNPACK_FAIL"
        if [ $UNPACK_EXIT -eq 0 ] && [ -f "$RESTORED" ]; then
            HASH_ORIG=$(sha256sum "$src" | cut -d' ' -f1)
            HASH_REST=$(sha256sum "$RESTORED" | cut -d' ' -f1)
            if [ "$HASH_ORIG" == "$HASH_REST" ]; then
                VERIFY_RESULT="PASS"
            else
                VERIFY_RESULT="HASH_MISMATCH"
            fi
            echo "$VERIFY_RESULT|$HASH_ORIG|$fmt|$BASENAME" >> "$VERIFY_DIR/same_brain.log"
        else
            # Extrair mensagem de erro para documentação
            ERR_MSG=$(echo "$UNPACK_OUTPUT" | grep -i "error\|erro" | head -1 | cut -c1-80)
            echo "UNPACK_FAIL|$fmt|$BASENAME|$ERR_MSG" >> "$VERIFY_DIR/same_brain.log"
        fi
        rm -f "$RESTORED"
        
        csv_append "$METRICS" "$fmt,$BASENAME,$ORIG_SIZE,$CROM_SIZE,$RATIO,$SAVING,$ELAPSED,$VERIFY_RESULT"
        
        case $VERIFY_RESULT in
            PASS)
                log_ok "  $BASENAME: ratio=${RATIO}% saving=${SAVING}% | ${ELAPSED}ms | SHA-256 ✅ PASS"
                ;;
            UNPACK_FAIL)
                log_warn "  $BASENAME: ratio=${RATIO}% saving=${SAVING}% | ${ELAPSED}ms | ⚠️ UNPACK_FAIL (delta pool overflow)"
                ;;
            *)
                log_err "  $BASENAME: ratio=${RATIO}% | VERIFY $VERIFY_RESULT"
                ;;
        esac
    done
done

# --------------------------------------------------------------------------
# Ranking por formato
# --------------------------------------------------------------------------
log_phase "RANKING: COMPRESSÃO NATIVA POR FORMATO"
echo ""
printf "  %-6s │ %-12s │ %-12s │ %-10s │ %-12s │ %-10s\n" "FMT" "AVG RATIO" "AVG SAVING" "AVG TIME" "VERIFY" "NOTA"
printf "  %-6s─┼─%-12s─┼─%-12s─┼─%-10s─┼─%-12s─┼─%-10s\n" "──────" "────────────" "────────────" "──────────" "────────────" "──────────"

for fmt in "${FORMATS[@]}"; do
    AVG_RATIO=$(grep "^$fmt," "$METRICS" | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    AVG_SAVING=$(grep "^$fmt," "$METRICS" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    AVG_TIME=$(grep "^$fmt," "$METRICS" | awk -F',' '{sum+=$7; n++} END {if(n>0) printf "%d", sum/n; else print "N/A"}')
    PASS_CNT=$(grep "^$fmt," "$METRICS" | awk -F',' '$8=="PASS"' | wc -l)
    TOTAL_CNT=$(grep "^$fmt," "$METRICS" | wc -l)
    UNPACK_FAIL=$(grep "^$fmt," "$METRICS" | awk -F',' '$8=="UNPACK_FAIL"' | wc -l)
    
    NOTA=""
    if [ "$UNPACK_FAIL" -gt 0 ]; then
        NOTA="⚠️ DeltaPool"
    fi
    
    printf "  %-6s │ %-12s │ %-12s │ %-10s │ %-12s │ %-10s\n" \
        "$fmt" "${AVG_RATIO}%" "${AVG_SAVING}%" "${AVG_TIME}ms" "$PASS_CNT/$TOTAL_CNT PASS" "$NOTA"
done
echo ""

log_ok "Experimento A concluído! Resultados em: $METRICS"
