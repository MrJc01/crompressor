#!/bin/bash
# ==============================================================================
# 🌐 Pesquisa 06 — Script 05: Experimento D — Cérebro Universal
# Testa brain_universal contra todos os formatos e compara com especialistas.
# ==============================================================================

source "$(dirname "$0")/utils.sh"

log_phase "EXPERIMENTO D: CÉREBRO UNIVERSAL VS ESPECIALISTAS"

OUT_DIR="$RESULTS/universal"
METRICS="$OUT_DIR/universal_vs_specialist.csv"
csv_init "$METRICS" "formato,arquivo,orig_bytes,crom_univ_bytes,ratio_univ_pct,saving_univ_pct,time_univ_ms,verify_univ,ratio_specialist_pct,penalty_pct"

BRAIN_UNIV="$BRAINS/brain_universal.cromdb"

if [ ! -f "$BRAIN_UNIV" ]; then
    log_err "brain_universal.cromdb não encontrado!"
    exit 1
fi

SAME_BRAIN_METRICS="$RESULTS/same_brain/metrics.csv"

for fmt in "${FORMATS[@]}"; do
    log_info "Testando $fmt com brain_universal..."
    
    SPECIALIST_AVG=$(grep "^$fmt," "$SAME_BRAIN_METRICS" 2>/dev/null | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f", sum/n; else print "100"}')
    
    for split in train test; do
        dir="$DATASETS/${fmt}/$split"
        [ -d "$dir" ] || continue
        ls "$dir"/* > /dev/null 2>&1 || continue
        
        files=("$dir/"*)
        for src in "${files[@]}"; do
            [ -f "$src" ] || continue
            BASENAME=$(basename "$src")
            CROM_OUT="$OUT_DIR/${fmt}_univ_${split}_${BASENAME}.crom"
            RESTORED="/tmp/crom_univ_$$_${BASENAME}"
            
            # Pack com brain universal
            local_start=$(timer_start)
            $BIN pack -i "$src" -c "$BRAIN_UNIV" -o "$CROM_OUT" > /dev/null 2>&1
            PACK_EXIT=$?
            ELAPSED=$(timer_elapsed_ms "$local_start")
            
            if [ $PACK_EXIT -ne 0 ] || [ ! -f "$CROM_OUT" ]; then
                csv_append "$METRICS" "$fmt,$BASENAME,$(stat -c%s "$src"),0,0,0,$ELAPSED,PACK_FAIL,$SPECIALIST_AVG,N/A"
                continue
            fi
            
            ORIG_SIZE=$(stat -c%s "$src")
            CROM_SIZE=$(stat -c%s "$CROM_OUT")
            RATIO=$(echo "scale=2; $CROM_SIZE / $ORIG_SIZE * 100" | bc)
            SAVING=$(echo "scale=2; 100 - $RATIO" | bc)
            
            # Verify
            $BIN unpack -i "$CROM_OUT" -c "$BRAIN_UNIV" -o "$RESTORED" > /dev/null 2>&1
            UNPACK_EXIT=$?
            
            VERIFY_RESULT="UNPACK_FAIL"
            if [ $UNPACK_EXIT -eq 0 ] && [ -f "$RESTORED" ]; then
                HASH_ORIG=$(sha256sum "$src" | cut -d' ' -f1)
                HASH_REST=$(sha256sum "$RESTORED" | cut -d' ' -f1)
                [ "$HASH_ORIG" == "$HASH_REST" ] && VERIFY_RESULT="PASS" || VERIFY_RESULT="HASH_MISMATCH"
            fi
            rm -f "$RESTORED"
            
            PENALTY=$(echo "scale=2; $RATIO - $SPECIALIST_AVG" | bc 2>/dev/null || echo "N/A")
            
            csv_append "$METRICS" "$fmt,$BASENAME,$ORIG_SIZE,$CROM_SIZE,$RATIO,$SAVING,$ELAPSED,$VERIFY_RESULT,$SPECIALIST_AVG,$PENALTY"
            
            log_info "  $BASENAME: univ=${RATIO}% vs spec=${SPECIALIST_AVG}% | penalty=${PENALTY}pp | $VERIFY_RESULT"
        done
    done
done

# --------------------------------------------------------------------------
# Resumo Universal vs Especialista
# --------------------------------------------------------------------------
log_phase "COMPARAÇÃO: UNIVERSAL VS ESPECIALISTA"
echo ""
printf "  %-6s │ %-15s │ %-15s │ %-12s │ %-12s\n" "FMT" "RATIO UNIV(%)" "RATIO SPEC(%)" "PENALTY(pp)" "VEREDICTO"
printf "  %-6s─┼─%-15s─┼─%-15s─┼─%-12s─┼─%-12s\n" "──────" "───────────────" "───────────────" "────────────" "────────────"

for fmt in "${FORMATS[@]}"; do
    UNIV_R=$(grep "^$fmt," "$METRICS" 2>/dev/null | awk -F',' '{sum+=$5; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    SPEC_R=$(grep "^$fmt," "$METRICS" 2>/dev/null | awk -F',' '{sum+=$9; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    PENALTY=$(grep "^$fmt," "$METRICS" 2>/dev/null | awk -F',' '{if($10!="N/A"){sum+=$10; n++}} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    
    VERDICT="N/A"
    if [ "$PENALTY" != "N/A" ]; then
        P_ABS=$(echo "$PENALTY" | tr -d '-')
        if (( $(echo "${P_ABS:-0} < 5" | bc -l 2>/dev/null || echo 0) )); then
            VERDICT="✅ EQUIV."
        elif (( $(echo "${P_ABS:-0} < 15" | bc -l 2>/dev/null || echo 0) )); then
            VERDICT="🟡 ACEIT."
        else
            VERDICT="🔴 ESPEC."
        fi
    fi
    
    printf "  %-6s │ %-15s │ %-15s │ %-12s │ %-12s\n" "$fmt" "$UNIV_R" "$SPEC_R" "$PENALTY" "$VERDICT"
done
echo ""

log_ok "Experimento D concluído! Resultados em: $METRICS"
