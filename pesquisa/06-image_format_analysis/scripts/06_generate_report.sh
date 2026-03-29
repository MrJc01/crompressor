#!/bin/bash
# ==============================================================================
# 📝 Pesquisa 06 — Script 06: Geração Automática de Relatório
# Lê todos os CSVs e gera relatorio.md
# ==============================================================================

set -e
source "$(dirname "$0")/utils.sh"

log_phase "GERANDO RELATÓRIO CONSOLIDADO"

REPORT="$BASE_DIR/relatorio.md"
SAME="$RESULTS/same_brain/metrics.csv"
CROSS="$RESULTS/cross_brain/matrix.csv"
INFER="$RESULTS/inference/degradation.csv"
UNIV="$RESULTS/universal/universal_vs_specialist.csv"
TRAIN_M="$BRAINS/training_metrics.csv"

cat > "$REPORT" <<'HEADER'
# 📊 Relatório 06: Impacto de Extensões de Imagem na Compressão por Cérebros

Este relatório apresenta os resultados da análise científica de como diferentes formatos de imagem
interagem com o sistema de compressão do Crompressor (codebooks/cérebros).

- **Formatos Testados**: BMP, PNG, JPEG, WebP, GIF, TIFF, SVG
- **Cérebros Treinados**: 7 específicos + 1 universal
- **Total de Combinações**: ~77 testes com verificação SHA-256
- **Data da Auditoria**: AUDIT_DATE
- **Status de Integridade**: INTEGRITY_STATUS

---

HEADER

# Substituir data
sed -i "s/AUDIT_DATE/$(date '+%Y-%m-%d %H:%M')/" "$REPORT"

# --------------------------------------------------------------------------
# Seção 1: Inventário do Dataset
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'
## 📦 1. Inventário do Dataset

Dataset controlado gerado via ImageMagick: mesmas imagens-fonte convertidas para 7 formatos distintos.

| Formato | Arquivos Treino | Arquivos Teste | Observação |
|:--------|:----------------|:---------------|:-----------|
EOF

for fmt in "${FORMATS[@]}"; do
    TRAIN_C=$(ls "$DATASETS/${fmt}/train" 2>/dev/null | wc -l)
    TEST_C=$(ls "$DATASETS/${fmt}/test" 2>/dev/null | wc -l)
    TRAIN_SZ=$(du -sh "$DATASETS/${fmt}/train" 2>/dev/null | cut -f1 || echo "0")
    case $fmt in
        bmp)  NOTE="24-bit não comprimido" ;;
        png)  NOTE="Deflate (LZ77+Huffman)" ;;
        jpg)  NOTE="DCT lossy Q95" ;;
        webp) NOTE="VP8L lossless" ;;
        gif)  NOTE="LZW 256 cores" ;;
        tiff) NOTE="Sem compressão" ;;
        svg)  NOTE="XML text-based" ;;
    esac
    echo "| **$fmt** | $TRAIN_C ($TRAIN_SZ) | $TEST_C | $NOTE |" >> "$REPORT"
done

# --------------------------------------------------------------------------
# Seção 2: Cérebros Treinados
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🧠 2. Cérebros Treinados

EOF

if [ -f "$TRAIN_M" ]; then
    echo "| Cérebro | Formato | Tempo (ms) | Tamanho (.cromdb) |" >> "$REPORT"
    echo "|:--------|:--------|:-----------|:------------------|" >> "$REPORT"
    tail -n +2 "$TRAIN_M" | while IFS=',' read -r name fmt size time db_size files; do
        echo "| **$name** | $fmt | ${time}ms | $(fmt_bytes $db_size) |" >> "$REPORT"
    done
fi

# --------------------------------------------------------------------------
# Seção 3: Experimento A — Same Brain
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 📊 3. Experimento A: Compressão Nativa (Formato × Seu Próprio Cérebro)

> Pergunta: "Qual formato o Crompressor comprime melhor quando treinado especificamente?"

EOF

if [ -f "$SAME" ]; then
    echo "| Formato | Tam. Original (Méd.) | Tam. CROM (Méd.) | Ratio (%) | Economia (%) | Verify |" >> "$REPORT"
    echo "|:--------|:---------------------|:-----------------|:----------|:-------------|:-------|" >> "$REPORT"
    for fmt in "${FORMATS[@]}"; do
        DATA=$(grep "^$fmt," "$SAME" | awk -F',' '{
            o+=($3); c+=($4); r+=($5); s+=($6); n++;
            if($11!="PASS") fail=1
        } END {
            if(n>0) printf "%d|%d|%.2f|%.2f|%s", o/n, c/n, r/n, s/n, (fail?"⚠️ MIXED":"✅ ALL PASS");
            else print "N/A|N/A|N/A|N/A|N/A"
        }')
        IFS='|' read -r orig crom ratio saving verify <<< "$DATA"
        echo "| **$fmt** | $(fmt_bytes ${orig:-0}) | $(fmt_bytes ${crom:-0}) | ${ratio}% | **${saving}%** | $verify |" >> "$REPORT"
    done
fi

# --------------------------------------------------------------------------
# Seção 4: Experimento B — Cross Brain
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🔀 4. Experimento B: Matriz Cruzada (Formato × Cérebro Alheio)

> Pergunta: "Um cérebro treinado em JPEGs comprime bem BMPs? E vice-versa?"

### Heatmap de Economia (%) — Linha=Formato, Coluna=Cérebro

EOF

if [ -f "$CROSS" ]; then
    # Header
    printf "| FMT↓ BR→ |" >> "$REPORT"
    for b in "${FORMATS[@]}"; do
        printf " **%s** |" "$b" >> "$REPORT"
    done
    echo "" >> "$REPORT"
    
    printf "|:---------|" >> "$REPORT"
    for b in "${FORMATS[@]}"; do
        printf ":------|" >> "$REPORT"
    done
    echo "" >> "$REPORT"
    
    for fmt in "${FORMATS[@]}"; do
        printf "| **%s** |" "$fmt" >> "$REPORT"
        for brain_fmt in "${FORMATS[@]}"; do
            SAVING=$(grep "^$fmt,$brain_fmt," "$CROSS" | head -1 | awk -F',' '{print $7}')
            if [ "$fmt" == "$brain_fmt" ]; then
                printf " **%s%%** |" "${SAVING:-—}" >> "$REPORT"
            else
                printf " %s%% |" "${SAVING:-—}" >> "$REPORT"
            fi
        done
        echo "" >> "$REPORT"
    done
fi

# --------------------------------------------------------------------------
# Seção 5: Experimento C — Inferência
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🔮 5. Experimento C: Inferência Pós-Treino (Imagens Novas)

> Pergunta: "O cérebro generaliza para imagens que NUNCA viu durante o treinamento?"

EOF

if [ -f "$INFER" ]; then
    echo "| Formato | Ratio Treino (%) | Ratio Inferência (%) | Degradação (%) | Generaliza? |" >> "$REPORT"
    echo "|:--------|:-----------------|:---------------------|:---------------|:------------|" >> "$REPORT"
    for fmt in "${FORMATS[@]}"; do
        DATA=$(grep "^$fmt," "$INFER" 2>/dev/null | awk -F',' '{
            r+=($5); d+=($10); t+=($9); n++
        } END {
            if(n>0) printf "%.2f|%.2f|%.2f", t/n, r/n, d/n;
            else print "N/A|N/A|N/A"
        }')
        IFS='|' read -r train_r infer_r degrad <<< "$DATA"
        if [ "$degrad" != "N/A" ] && (( $(echo "${degrad:-100} < 30" | bc -l 2>/dev/null || echo 0) )); then
            GEN="✅ SIM (<30%)"
        elif [ "$degrad" != "N/A" ] && (( $(echo "${degrad:-100} < 50" | bc -l 2>/dev/null || echo 0) )); then
            GEN="🟡 PARCIAL"
        else
            GEN="🔴 NÃO (>50%)"
        fi
        echo "| **$fmt** | ${train_r}% | ${infer_r}% | ${degrad}% | $GEN |" >> "$REPORT"
    done
fi

# --------------------------------------------------------------------------
# Seção 6: Experimento D — Universal
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🌐 6. Experimento D: Cérebro Universal vs Especialistas

> Pergunta: "Vale a pena manter 7 cérebros ou 1 universal resolve tudo?"

EOF

if [ -f "$UNIV" ]; then
    echo "| Formato | Ratio Universal (%) | Ratio Especialista (%) | Penalty (pp) | Veredicto |" >> "$REPORT"
    echo "|:--------|:--------------------|:-----------------------|:-------------|:----------|" >> "$REPORT"
    for fmt in "${FORMATS[@]}"; do
        DATA=$(grep "^$fmt," "$UNIV" 2>/dev/null | awk -F',' '{
            u+=($5); s+=($9); p+=($10); n++
        } END {
            if(n>0) printf "%.2f|%.2f|%.2f", u/n, s/n, p/n;
            else print "N/A|N/A|N/A"
        }')
        IFS='|' read -r univ_r spec_r penalty <<< "$DATA"
        if (( $(echo "${penalty:-100} < 5" | bc -l 2>/dev/null || echo 0) )); then
            VERDICT="✅ Universal OK"
        elif (( $(echo "${penalty:-100} < 15" | bc -l 2>/dev/null || echo 0) )); then
            VERDICT="🟡 Aceitável"
        else
            VERDICT="🔴 Use Especialista"
        fi
        echo "| **$fmt** | ${univ_r}% | ${spec_r}% | ${penalty}pp | $VERDICT |" >> "$REPORT"
    done
fi

# --------------------------------------------------------------------------
# Seção 7: Validação de Hipóteses
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🧪 7. Validação de Hipóteses

### H1: Formatos brutos (BMP/TIFF) comprimem melhor
EOF

if [ -f "$SAME" ]; then
    BMP_SAVING=$(grep "^bmp," "$SAME" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    TIFF_SAVING=$(grep "^tiff," "$SAME" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    JPG_SAVING=$(grep "^jpg," "$SAME" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    PNG_SAVING=$(grep "^png," "$SAME" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    
    echo "- BMP saving: **${BMP_SAVING}%** | TIFF saving: **${TIFF_SAVING}%**" >> "$REPORT"
    echo "- JPG saving: **${JPG_SAVING}%** | PNG saving: **${PNG_SAVING}%**" >> "$REPORT"
    
    if (( $(echo "${BMP_SAVING:-0} > ${JPG_SAVING:-0}" | bc -l 2>/dev/null || echo 0) )); then
        echo "- **Resultado: ✅ CONFIRMADA** — Formatos brutos apresentam maior economia." >> "$REPORT"
    else
        echo "- **Resultado: ❌ REFUTADA** — Formatos comprimidos também são bem comprimidos pelo CROM." >> "$REPORT"
    fi
fi

cat >> "$REPORT" <<'EOF'

### H2: JPEG/WebP (pré-comprimidos) têm desempenho inferior
EOF

if [ -f "$SAME" ]; then
    WEBP_SAVING=$(grep "^webp," "$SAME" | awk -F',' '{sum+=$6; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    echo "- JPEG saving: **${JPG_SAVING}%** | WebP saving: **${WEBP_SAVING}%**" >> "$REPORT"
    echo "- BMP saving: **${BMP_SAVING}%** (referência raw)" >> "$REPORT"
    
    if (( $(echo "${JPG_SAVING:-0} < ${BMP_SAVING:-0}" | bc -l 2>/dev/null || echo 0) )); then
        echo "- **Resultado: ✅ CONFIRMADA** — Formatos pré-comprimidos sofrem com double-compression." >> "$REPORT"
    else
        echo "- **Resultado: ❌ REFUTADA** — O CROM encontra padrões mesmo em dados pré-comprimidos." >> "$REPORT"
    fi
fi

cat >> "$REPORT" <<'EOF'

### H3: Cross-format penalty é significativo
EOF

if [ -f "$CROSS" ]; then
    # Comparar diagonal (nativo) vs off-diagonal (cruzado)
    NATIVE_AVG=$(for fmt in "${FORMATS[@]}"; do grep "^$fmt,$fmt," "$CROSS" | awk -F',' '{print $7}'; done | awk '{sum+=$1; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    CROSS_AVG=$(for fmt in "${FORMATS[@]}"; do for bf in "${FORMATS[@]}"; do [ "$fmt" != "$bf" ] && grep "^$fmt,$bf," "$CROSS" | awk -F',' '{print $7}'; done; done | awk '{sum+=$1; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    
    echo "- Saving médio nativo (diagonal): **${NATIVE_AVG}%**" >> "$REPORT"
    echo "- Saving médio cruzado (off-diagonal): **${CROSS_AVG}%**" >> "$REPORT"
    PENALTY=$(echo "scale=2; $NATIVE_AVG - $CROSS_AVG" | bc 2>/dev/null || echo "N/A")
    echo "- Cross-format penalty: **${PENALTY}pp**" >> "$REPORT"
    
    if (( $(echo "${PENALTY:-0} > 10" | bc -l 2>/dev/null || echo 0) )); then
        echo "- **Resultado: ✅ CONFIRMADA** — Usar cérebro alheio degrada significativamente a compressão." >> "$REPORT"
    else
        echo "- **Resultado: ❌ REFUTADA** — O penalty cruzado é menor que 10pp, aceitável." >> "$REPORT"
    fi
fi

cat >> "$REPORT" <<'EOF'

### H4: Imagens novas mantêm ≥70% da taxa do treino
EOF

if [ -f "$INFER" ]; then
    AVG_DEGRAD=$(tail -n +2 "$INFER" 2>/dev/null | awk -F',' '{sum+=$10; n++} END {if(n>0) printf "%.2f", sum/n; else print "N/A"}')
    echo "- Degradação média pós-treino: **${AVG_DEGRAD}%**" >> "$REPORT"
    
    if (( $(echo "${AVG_DEGRAD:-100} < 30" | bc -l 2>/dev/null || echo 0) )); then
        echo "- **Resultado: ✅ CONFIRMADA** — O cérebro generaliza bem, degradação < 30%." >> "$REPORT"
    else
        echo "- **Resultado: ❌ REFUTADA** — O cérebro tem dificuldade para generalizar (degradação > 30%)." >> "$REPORT"
    fi
fi

# --------------------------------------------------------------------------
# Seção 8: Conclusões e Recomendações
# --------------------------------------------------------------------------
cat >> "$REPORT" <<'EOF'

---

## 🏆 8. Conclusões e Recomendações

### Recomendações Práticas

1. **Para pipelines de imagens brutas (BMP/TIFF)**: O Crompressor é uma alternativa viável a Gzip/Zstd, oferecendo compressão com acesso aleatório via VFS.
2. **Para imagens pré-comprimidas (JPEG/WebP)**: Avaliar se o overhead do codebook justifica a economia adicional.
3. **Cérebro Universal vs Especialista**: Consultar a tabela do Experimento D para decidir.
4. **Generalização**: Se a degradação pós-treino for baixa, o cérebro pode ser compartilhado entre nós P2P sem retreino.

---

> [!TIP]
> Em ambientes de produção com milhões de imagens médicas (DICOM/TIFF), a economia do Crompressor pode representar TB de redução de storage.

**"Não comprimimos pixels. Compilamos a realidade visual."**
EOF

# Verificar integridade global
TOTAL_TESTS=$(cat "$VERIFY_DIR"/*.log 2>/dev/null | wc -l)
PASS_TESTS=$(cat "$VERIFY_DIR"/*.log 2>/dev/null | grep -c "PASS" || echo 0)

if [ "$TOTAL_TESTS" -eq "$PASS_TESTS" ] && [ "$TOTAL_TESTS" -gt 0 ]; then
    sed -i "s/INTEGRITY_STATUS/✅ PASS ($PASS_TESTS\/$TOTAL_TESTS testes lossless)/" "$REPORT"
else
    sed -i "s/INTEGRITY_STATUS/⚠️ $PASS_TESTS\/$TOTAL_TESTS PASS/" "$REPORT"
fi

log_ok "Relatório gerado: $REPORT"
log_ok "Total de verificações: $PASS_TESTS/$TOTAL_TESTS PASS"
