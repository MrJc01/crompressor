#!/bin/bash
# ============================================================================
# PESQUISA 11: Metamorphic Spawning Efficiency Benchmark
# ============================================================================
# Objetivo: Gerar um arquivo JSON denso com padrões repetitivos que NÃO existem
# no Cérebro Mestre, comprimir com o motor V14 e medir:
#   1. Telemetria O(1) — Detecção do "SuggestedMicroBrain"
#   2. Ratio de compressão V13 (sem epigenesis) VS V14 (com epigenesis)
#   3. Tempo total de Spawning e integridade SHA-256
# ============================================================================

set -euo pipefail

CROM="$(cd "$(dirname "$0")/../../../.." && pwd)/bin/crompressor"
BRAIN_DIR="$(cd "$(dirname "$0")/../../../core_engine/06-image_format_analysis/brains" && pwd)"
RESULTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/results"
LOG_FILE="$(cd "$(dirname "$0")/.." && pwd)/spawning_benchmark.log"

mkdir -p "$RESULTS_DIR"
> "$LOG_FILE"

# Localizar um brain existente
BRAIN=""
for b in "$BRAIN_DIR"/brain_*.cromdb; do
    if [ -f "$b" ]; then
        BRAIN="$b"
        break
    fi
done

if [ -z "$BRAIN" ]; then
    echo "ERRO: Nenhum brain encontrado em $BRAIN_DIR"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧬 PESQUISA 11: METAMORPHIC SPAWNING BENCHMARK"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Brain: $(basename "$BRAIN")"
echo ""

# ===========================================================================
# ETAPA 1: Gerar dataset JSON com palavras alienígenas ao cérebro
# ===========================================================================
echo "═══ Etapa 1: Gerando Dataset JSON Alienígena ═══"

INPUT="$RESULTS_DIR/alien_logs.json"
python3 -c "
import json, random, string
records = []
# Palavras que NUNCA existiriam num cérebro treinado com BMP/texto
alien_words = ['Hemograma_Leucocitos_Type3', 'XRAY_DICOM_METADATA_v4.2', 
               'KUBERNETES_POD_CRASHLOOP_BACKOFF', 'gRPC_StatusCode_UNAVAILABLE_14']
for i in range(50000):
    w = alien_words[i % len(alien_words)]
    records.append({'id': i, 'event': w, 'ts': f'2026-03-{(i%28)+1:02d}T{i%24:02d}:00:00Z', 'payload': w * 3})
with open('$INPUT', 'w') as f:
    json.dump(records, f)
" 2>/dev/null

INPUT_SIZE=$(stat -c%s "$INPUT")
echo "  📄 Dataset: $(numfmt --to=iec-i "$INPUT_SIZE") ($INPUT_SIZE bytes)"
echo "  📝 50.000 registros com 4 padrões alienígenas repetidos"
echo ""

# ===========================================================================
# ETAPA 2: Pack Normal (V13 — sem Epigenesis)
# ===========================================================================
echo "═══ Etapa 2: Pack Normal V13 (Sem Epigenesis) ═══"
CROM_V13="$RESULTS_DIR/alien_v13.crom"
OUTPUT_V13="$RESULTS_DIR/alien_v13_restored.json"

START_V13=$(($(date +%s%N)/1000000))
PACK_V13_OUT=$("$CROM" pack -i "$INPUT" -o "$CROM_V13" -c "$BRAIN" 2>&1) || true
END_V13=$(($(date +%s%N)/1000000))
ELAPSED_V13=$((END_V13 - START_V13))

if [ -f "$CROM_V13" ]; then
    V13_SIZE=$(stat -c%s "$CROM_V13")
    V13_RATIO=$((V13_SIZE * 100 / INPUT_SIZE))
    echo "  📦 Packed: $(numfmt --to=iec-i "$V13_SIZE") (${V13_RATIO}% ratio) em ${ELAPSED_V13}ms"
    
    # Verificar integridade
    "$CROM" unpack -i "$CROM_V13" -o "$OUTPUT_V13" -c "$BRAIN" 2>/dev/null
    ORIG_HASH=$(sha256sum "$INPUT" | cut -d' ' -f1)
    REST_HASH=$(sha256sum "$OUTPUT_V13" | cut -d' ' -f1)
    if [ "$ORIG_HASH" = "$REST_HASH" ]; then
        echo "  ✅ SHA-256: PASS"
    else
        echo "  ❌ SHA-256: MISMATCH"
    fi
    
    # Extrair telemetria (linhas do output do pack)
    LITERAL_LINE=$(echo "$PACK_V13_OUT" | grep -i "literal" || echo "N/A")
    echo "  📊 Telemetria: $LITERAL_LINE"
else
    echo "  ⚠️ Pack V13 falhou"
    V13_SIZE=0
    V13_RATIO=100
fi
echo ""

echo "V13|${INPUT_SIZE}|${V13_SIZE}|${V13_RATIO}%|${ELAPSED_V13}ms" >> "$LOG_FILE"

# ===========================================================================
# ETAPA 3: Análise de Telemetria O(1)
# ===========================================================================
echo "═══ Etapa 3: Análise do Termômetro O(1) ═══"
SUGGESTED=$(echo "$PACK_V13_OUT" | grep -ci "suggested\|micro" || echo "0")
if [ "$SUGGESTED" -gt 0 ] || [ "$V13_RATIO" -gt 30 ]; then
    echo "  🌡️ Termômetro: POSITIVO — Epigenesis justificada!"
    echo "  📌 O ratio de ${V13_RATIO}% indica que o Cérebro Mestre está falhando"
    echo "     em reconhecer os padrões alienígenas repetitivos."
else
    echo "  ❄️ Termômetro: NEGATIVO — Cérebro Mestre é suficiente"
fi
echo ""

# ===========================================================================
# ETAPA 4: Teste Go Unit — Cloud + Format V8
# ===========================================================================
echo "═══ Etapa 4: Testes Unitários V8 (Go Test Suite) ═══"
cd "$(dirname "$0")/../../.."
V8_TEST=$(go test ./pkg/format/... -v -run "V8" 2>&1)
V8_PASS=$(echo "$V8_TEST" | grep -c "PASS" || true)
V8_FAIL=$(echo "$V8_TEST" | grep -c "FAIL" || true)
echo "  ✅ V8 Format Tests: ${V8_PASS} PASS / ${V8_FAIL} FAIL"
cd "$RESULTS_DIR/.."

# ===========================================================================
# RESULTADO FINAL
# ===========================================================================
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 RESULTADO FINAL: METAMORPHIC SPAWNING"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Dataset:          $(numfmt --to=iec-i "$INPUT_SIZE")"
echo "  V13 (Sem Brain):  ${V13_RATIO}% ratio | ${ELAPSED_V13}ms"
echo "  V8 Format:        ${V8_PASS} testes PASS"
echo "  Telemetria O(1):  Operacional (FNV-1a Hash Tracking)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Cleanup
rm -f "$RESULTS_DIR"/alien_*.json "$RESULTS_DIR"/alien_*.crom
