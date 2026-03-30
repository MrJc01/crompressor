#!/bin/bash
# ============================================================================
# PESQUISA 08: Cloud VFS Latency vs Local SSD
# ============================================================================
# Simula latência de cloud storage usando um servidor HTTP local (python)
# servindo o .crom file, e compara performance do grep/mount contra disco local.
# ============================================================================

set -euo pipefail

CROM="$(cd "$(dirname "$0")/../../.." && pwd)/bin/crompressor"
BRAIN_DIR="$(cd "$(dirname "$0")/../../06-image_format_analysis/brains" && pwd)"
RESULTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/results"
LOG_FILE="$(cd "$(dirname "$0")/.." && pwd)/cloud_latency.log"

mkdir -p "$RESULTS_DIR"
> "$LOG_FILE"

# Selecionar brain e criar dataset de teste
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
echo "🧪 PESQUISA 08: CLOUD VFS LATENCY vs LOCAL SSD"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Brain: $(basename "$BRAIN")"
echo ""

# 1. Criar dataset de texto de teste (~1MB)
TEST_INPUT="$RESULTS_DIR/test_cloud.txt"
python3 -c "
import random, string
lines = [''.join(random.choices(string.ascii_letters + string.digits + ' \n', k=80)) for _ in range(13000)]
with open('$TEST_INPUT', 'w') as f:
    f.write('\n'.join(lines))
" 2>/dev/null

if [ ! -f "$TEST_INPUT" ]; then
    echo "Fallback: gerando dados com /dev/urandom"
    head -c 1048576 /dev/urandom | base64 > "$TEST_INPUT"
fi

TEST_CROM="$RESULTS_DIR/test_cloud.crom"

echo "📁 Empacotando dataset de teste..."
"$CROM" pack -i "$TEST_INPUT" -o "$TEST_CROM" -c "$BRAIN" 2>/dev/null || true

if [ ! -f "$TEST_CROM" ]; then
    echo "ERRO: Falha ao empacotar dataset de teste"
    exit 1
fi

CROM_SIZE=$(stat -c%s "$TEST_CROM")
echo "   Arquivo .crom: $(numfmt --to=iec-i $CROM_SIZE 2>/dev/null || echo ${CROM_SIZE}B)"

# 2. Iniciar servidor HTTP local (simula S3/CDN)
HTTP_PORT=18765
HTTP_DIR="$RESULTS_DIR"

echo ""
echo "🌐 Iniciando servidor HTTP simulando S3 na porta $HTTP_PORT..."
cd "$HTTP_DIR"
python3 -m http.server $HTTP_PORT --bind 127.0.0.1 &>/dev/null &
HTTP_PID=$!
cd - > /dev/null
sleep 1

# Verificar se o servidor está rodando
if ! kill -0 $HTTP_PID 2>/dev/null; then
    echo "ERRO: Servidor HTTP não iniciou"
    exit 1
fi

cleanup() {
    kill $HTTP_PID 2>/dev/null || true
    rm -f "$TEST_INPUT" "$TEST_CROM" "$RESULTS_DIR/test_cloud_restored.txt"
}
trap cleanup EXIT

REMOTE_URL="http://127.0.0.1:${HTTP_PORT}/test_cloud.crom"

echo "   URL remoto: $REMOTE_URL"
echo ""

# 3. Benchmark: LOCAL Grep
echo "═══ Teste A: Grep Local (Disco SSD) ═══"
LOCAL_TIMES=()
for i in $(seq 1 5); do
    START_NS=$(date +%s%N)
    "$CROM" grep "test" -i "$TEST_CROM" -c "$BRAIN" &>/dev/null || true
    END_NS=$(date +%s%N)
    ELAPSED_MS=$(( (END_NS - START_NS) / 1000000 ))
    LOCAL_TIMES+=($ELAPSED_MS)
    echo "  Run $i: ${ELAPSED_MS}ms"
done

# Calcular média local
LOCAL_SUM=0
for t in "${LOCAL_TIMES[@]}"; do LOCAL_SUM=$((LOCAL_SUM + t)); done
LOCAL_AVG=$((LOCAL_SUM / ${#LOCAL_TIMES[@]}))

echo "  📊 Média Local: ${LOCAL_AVG}ms"
echo ""

# 4. Benchmark: REMOTE Grep (HTTP Range)
echo "═══ Teste B: Grep Remoto (HTTP Range/S3 Simulado) ═══"
REMOTE_TIMES=()
for i in $(seq 1 5); do
    START_NS=$(date +%s%N)
    "$CROM" grep "test" -i "$REMOTE_URL" -c "$BRAIN" &>/dev/null || true
    END_NS=$(date +%s%N)
    ELAPSED_MS=$(( (END_NS - START_NS) / 1000000 ))
    REMOTE_TIMES+=($ELAPSED_MS)
    echo "  Run $i: ${ELAPSED_MS}ms"
done

# Calcular média remota
REMOTE_SUM=0
for t in "${REMOTE_TIMES[@]}"; do REMOTE_SUM=$((REMOTE_SUM + t)); done
REMOTE_AVG=$((REMOTE_SUM / ${#REMOTE_TIMES[@]}))

echo "  📊 Média Remota: ${REMOTE_AVG}ms"
echo ""

# 5. Benchmark: LOCAL Unpack
echo "═══ Teste C: Unpack Local vs Overhead Estimado ═══"
START_NS=$(date +%s%N)
"$CROM" unpack -i "$TEST_CROM" -o "$RESULTS_DIR/test_cloud_restored.txt" -c "$BRAIN" 2>/dev/null || true
END_NS=$(date +%s%N)
UNPACK_MS=$(( (END_NS - START_NS) / 1000000 ))
echo "  Unpack Local: ${UNPACK_MS}ms"

# 6. Relatório
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 RELATÓRIO FINAL: CLOUD VFS LATENCY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Grep Local (SSD):    ${LOCAL_AVG}ms (média 5 runs)"
echo "  Grep Remoto (HTTP):  ${REMOTE_AVG}ms (média 5 runs)"
if [ "$LOCAL_AVG" -gt 0 ]; then
    OVERHEAD=$((REMOTE_AVG * 100 / LOCAL_AVG - 100))
    echo "  Overhead HTTP:       ${OVERHEAD}%"
fi
echo "  Unpack Local:        ${UNPACK_MS}ms"
echo "  .crom Size:          $(numfmt --to=iec-i $CROM_SIZE 2>/dev/null || echo ${CROM_SIZE}B)"
echo ""
echo "  Modo: HTTP Range Requests (Zero-Download)"
echo "  API Calls estimadas: 2 (HEAD + GET Range por bloco)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Log
echo "LOCAL_AVG=${LOCAL_AVG}ms|REMOTE_AVG=${REMOTE_AVG}ms|UNPACK=${UNPACK_MS}ms|SIZE=${CROM_SIZE}" >> "$LOG_FILE"
