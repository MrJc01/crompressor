#!/bin/bash
# ==============================================================
# CROM LAB LAUNCHER V3: Montagem + Teste + Log (Autocontido)
# ==============================================================

APP_NAME=$(basename "$(pwd)")
BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')

# Diretórios de trabalho
MNT_RO="./mnt"
UPPER_DIR="./vfs_upper"
WORK_DIR="./vfs_work"
MERGED_DIR="./merged"
LOG_DIR="./logs"
SOURCE_CROM="./${APP_NAME}.crom"
LOG_FILE="$LOG_DIR/${TIMESTAMP}.log"

# Cores
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# Função de log (escreve no terminal E no arquivo)
log() {
    echo -e "$1"
    echo -e "$1" | sed 's/\x1b\[[0-9;]*m//g' >> "$LOG_FILE"
}

mkdir -p "$LOG_DIR"

log "========================================================"
log "🛠️  LABORATÓRIO: $APP_NAME"
log "📅  Timestamp: $TIMESTAMP"
log "========================================================"

# -------------------------------------------------------
# FASE 1: MONTAGEM DO AMBIENTE VIRTUAL
# -------------------------------------------------------
fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
mkdir -p "$MNT_RO" "$UPPER_DIR" "$WORK_DIR" "$MERGED_DIR"

# Montar CROM se existir
if [ -f "$SOURCE_CROM" ]; then
    log "🌌 Montando volume CROM: $SOURCE_CROM"
    CROMDB=$(find . -name "*.cromdb" -type f | head -1)
    if [ -n "$CROMDB" ]; then
        "$CROM_BIN" mount -i "$SOURCE_CROM" -m "$MNT_RO" -c "$CROMDB" &
        sleep 2
        log "✅ Volume CROM montado."
    else
        log "⚠️  .cromdb não encontrado. Montando sem codebook..."
        "$CROM_BIN" mount -i "$SOURCE_CROM" -m "$MNT_RO" &
        sleep 2
    fi
else
    log "ℹ️  Sem arquivo .crom — usando apenas OverlayFS."
fi

# Montar OverlayFS
fuse-overlayfs -o lowerdir="$MNT_RO",upperdir="$UPPER_DIR",workdir="$WORK_DIR" "$MERGED_DIR" 2>/dev/null
if mountpoint -q "$MERGED_DIR" 2>/dev/null; then
    log "${GREEN}✅ Ambiente virtual montado em '$MERGED_DIR'${NC}"
else
    log "${YELLOW}⚠️  OverlayFS indisponível — executando teste direto.${NC}"
    MERGED_DIR="."
fi

# -------------------------------------------------------
# FASE 2: EXECUÇÃO DO TESTE AUTOMATICO
# -------------------------------------------------------
log ""
log "${BOLD}${BLUE}🧪 EXECUTANDO TESTE: $APP_NAME${NC}"
log "--------------------------------------------------------"

if [ -f "./test_logic.sh" ]; then
    # Executa capturando output e tempo
    START_TIME=$(date +%s%N)
    bash ./test_logic.sh 2>&1 | tee -a "$LOG_FILE"
    EXIT_CODE=${PIPESTATUS[0]}
    END_TIME=$(date +%s%N)
    ELAPSED=$(( (END_TIME - START_TIME) / 1000000 ))

    log ""
    log "--------------------------------------------------------"
    if [ $EXIT_CODE -eq 0 ]; then
        log "${GREEN}✅ RESULTADO: PASS (${ELAPSED}ms)${NC}"
    else
        log "${RED}❌ RESULTADO: FAIL (exit code $EXIT_CODE, ${ELAPSED}ms)${NC}"
    fi
else
    log "${RED}❌ test_logic.sh não encontrado!${NC}"
fi

# -------------------------------------------------------
# FASE 3: RELATÓRIO E LIMPEZA
# -------------------------------------------------------
log ""
log "📊 Tamanho do diretório de trabalho:"
du -sh "$UPPER_DIR" 2>/dev/null | tee -a "$LOG_FILE" || true
log ""
log "📁 Log salvo em: $LOG_FILE"

# Desmontagem
fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
log "${GREEN}🔒 Ambiente desmontado.${NC}"
log "========================================================"
