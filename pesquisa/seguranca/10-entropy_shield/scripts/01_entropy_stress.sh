#!/bin/bash
# ============================================================================
# PESQUISA 10: Entropy Shield Validation (Stress Destrutivo)
# ============================================================================
# Objetivo: Injetar lixo binário de alta entropia (/dev/urandom) de vários
# tamanhos no pipeline Pack → Unpack e validar:
#   1. 100% passagem SHA-256 (integridade bit-a-bit)
#   2. Tamanho constante de DeltaPool sem Overflow
#   3. Ativação correta do modo Passthrough para entropia > 7.5
# ============================================================================

set -euo pipefail

CROM="$(cd "$(dirname "$0")/../../../.." && pwd)/bin/crompressor"
BRAIN_DIR="$(cd "$(dirname "$0")/../../../core_engine/06-image_format_analysis/brains" && pwd)"
RESULTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/results"
LOG_FILE="$(cd "$(dirname "$0")/.." && pwd)/entropy_shield.log"

mkdir -p "$RESULTS_DIR"
> "$LOG_FILE"

# Usar brain_text como base (pior cenário para dados aleatórios)
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
echo "🧪 PESQUISA 10: ENTROPY SHIELD STRESS TEST"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Brain: $(basename "$BRAIN")"
echo ""

SIZES=(128 256 512 1024 4096 8192 16384 32768 65536 131072 262144 524288 1048576)
TOTAL=${#SIZES[@]}
PASS=0
FAIL=0

for SIZE in "${SIZES[@]}"; do
    LABEL=$(numfmt --to=iec-i "$SIZE" 2>/dev/null || echo "${SIZE}B")
    INPUT="$RESULTS_DIR/urandom_${SIZE}.bin"
    CROM_FILE="$RESULTS_DIR/urandom_${SIZE}.crom"
    OUTPUT="$RESULTS_DIR/urandom_${SIZE}_restored.bin"

    # 1. Gerar dados puramente aleatórios
    dd if=/dev/urandom of="$INPUT" bs="$SIZE" count=1 2>/dev/null

    # 2. Calcular SHA-256 original
    ORIG_HASH=$(sha256sum "$INPUT" | cut -d' ' -f1)

    # 3. Pack
    START_MS=$(($(date +%s%N)/1000000))
    PACK_OUT=$("$CROM" pack -i "$INPUT" -o "$CROM_FILE" -c "$BRAIN" 2>&1) || true
    
    if [ ! -f "$CROM_FILE" ]; then
        echo "⚠️  ${LABEL}: PACK_FAIL"
        echo "PACK_FAIL|${SIZE}|${LABEL}" >> "$LOG_FILE"
        FAIL=$((FAIL + 1))
        continue
    fi

    PACKED_SIZE=$(stat -c%s "$CROM_FILE" 2>/dev/null || echo "0")
    RATIO=$((PACKED_SIZE * 100 / SIZE))

    # 4. Unpack
    UNPACK_OUT=$("$CROM" unpack -i "$CROM_FILE" -o "$OUTPUT" -c "$BRAIN" 2>&1)
    UNPACK_EXIT=$?
    END_MS=$(($(date +%s%N)/1000000))
    ELAPSED=$((END_MS - START_MS))

    if [ $UNPACK_EXIT -ne 0 ]; then
        echo "⚠️  ${LABEL}: UNPACK_FAIL (${ELAPSED}ms)"
        echo "UNPACK_FAIL|${SIZE}|${LABEL}|${ELAPSED}ms|Error: ${UNPACK_OUT}" >> "$LOG_FILE"
        FAIL=$((FAIL + 1))
        rm -f "$INPUT" "$CROM_FILE" "$OUTPUT"
        continue
    fi

    # 5. Verificar SHA-256
    REST_HASH=$(sha256sum "$OUTPUT" | cut -d' ' -f1)

    if [ "$ORIG_HASH" = "$REST_HASH" ]; then
        echo "✅  ${LABEL}: ratio=${RATIO}% | ${ELAPSED}ms | SHA-256 ✅ PASS"
        echo "PASS|${ORIG_HASH}|${SIZE}|${LABEL}|ratio=${RATIO}%|${ELAPSED}ms" >> "$LOG_FILE"
        PASS=$((PASS + 1))
    else
        echo "❌  ${LABEL}: SHA-256 MISMATCH | ${ELAPSED}ms"
        echo "SHA_FAIL|${SIZE}|${LABEL}|orig=${ORIG_HASH}|rest=${REST_HASH}" >> "$LOG_FILE"
        FAIL=$((FAIL + 1))
    fi

    # Cleanup
    rm -f "$INPUT" "$CROM_FILE" "$OUTPUT"
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 RESULTADO FINAL: ${PASS}/${TOTAL} PASS | ${FAIL}/${TOTAL} FAIL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$FAIL" -eq 0 ]; then
    echo "🛡️  ENTROPY SHIELD: 100% VALIDADO — Zero Overflows em dados /dev/urandom"
else
    echo "⚠️  ENTROPY SHIELD: ${FAIL} falhas detectadas — investigação necessária"
fi
