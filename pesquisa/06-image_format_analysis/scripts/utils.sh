#!/bin/bash
# ==============================================================================
# 🛠️ Pesquisa 06: Funções Utilitárias
# ==============================================================================

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
_BIN_REAL="$(cd "$BASE_DIR/../../" && pwd)/bin/crompressor"
# Symlink sem espaço no path para evitar problemas de quoting em subshells
BIN="/tmp/crompressor_research_bin"
if [ ! -L "$BIN" ] || [ "$(readlink -f "$BIN")" != "$(readlink -f "$_BIN_REAL")" ]; then
    ln -sf "$_BIN_REAL" "$BIN"
fi
DATASETS="$BASE_DIR/datasets"
BRAINS="$BASE_DIR/brains"
RESULTS="$BASE_DIR/results"
VERIFY_DIR="$BASE_DIR/verify"

FORMATS=("bmp" "png" "jpg" "webp" "gif" "tiff" "svg")

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}ℹ️  $1${NC}"; }
log_ok()    { echo -e "${GREEN}✅ $1${NC}"; }
log_warn()  { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_err()   { echo -e "${RED}❌ $1${NC}"; }
log_phase() { echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${BOLD}🧪 $1${NC}"; echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# Timing em milissegundos
timer_start() { echo $(date +%s%N); }
timer_elapsed_ms() {
    local start=$1
    local end=$(date +%s%N)
    echo $(( (end - start) / 1000000 ))
}

# Calcular tamanho formatado
fmt_bytes() {
    local bytes=$1
    if [ "$bytes" -ge 1048576 ]; then
        echo "$(echo "scale=2; $bytes / 1048576" | bc) MB"
    elif [ "$bytes" -ge 1024 ]; then
        echo "$(echo "scale=2; $bytes / 1024" | bc) KB"
    else
        echo "${bytes} B"
    fi
}

# Append para CSV com header automático
csv_append() {
    local file=$1
    shift
    echo "$*" >> "$file"
}

csv_init() {
    local file=$1
    shift
    echo "$*" > "$file"
}

# Calcular entropia Shannon de um arquivo
calc_entropy() {
    local file=$1
    # Usa od + awk para calcular entropia byte-level
    od -An -tu1 -v "$file" | tr -s ' ' '\n' | grep -v '^$' | sort -n | uniq -c | awk -v total=$(stat -c%s "$file") '
    BEGIN { entropy = 0 }
    {
        count = $1
        p = count / total
        if (p > 0) entropy -= p * log(p) / log(2)
    }
    END { printf "%.4f", entropy }
    '
}

# Verificar integridade lossless
verify_integrity() {
    local original=$1
    local restored=$2
    local log_file=$3
    
    local hash_orig=$(sha256sum "$original" | cut -d' ' -f1)
    local hash_rest=$(sha256sum "$restored" | cut -d' ' -f1)
    
    if [ "$hash_orig" == "$hash_rest" ]; then
        echo "PASS|$hash_orig" | tee -a "$log_file"
        return 0
    else
        echo "FAIL|orig=$hash_orig|rest=$hash_rest" | tee -a "$log_file"
        return 1
    fi
}
