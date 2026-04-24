#!/bin/bash
# ==============================================================
# CROM LAB LAUNCHER V3.1: FUSE CASCADING Global (SRE)
# ==============================================================

APP_NAME=$(basename "$(pwd)")
BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')

# Diretórios de trabalho
MNT_CROM="./mnt_crom"
MNT_RO="./mnt_ro"
UPPER_DIR="./vfs_upper"
WORK_DIR="./vfs_work"
MERGED_DIR="./merged"
LOG_DIR="./logs"
SOURCE_CROM="./${APP_NAME}.crom"
LOG_FILE="$LOG_DIR/${TIMESTAMP}.log"

GREEN='\033[0;32m'; RED='\033[0;31m'; BLUE='\033[0;34m'
YELLOW='\033[1;33m'; BOLD='\033[1m'; NC='\033[0m'

log() { echo -e "$1"; echo -e "$1" | sed 's/\x1b\[[0-9;]*m//g' >> "$LOG_FILE"; }
mkdir -p "$LOG_DIR"

log "========================================================"
log "🛠️  LABORATÓRIO: $APP_NAME"
log "📅  Timestamp: $TIMESTAMP"
log "========================================================"

# --- DISK SAFETY GUARD (SRE) ---
FREE_SPACE=$(df -k . | awk 'NR==2 {print $4}')
if [ "$FREE_SPACE" -lt 2097152 ]; then # Menos de 2GB
    log "${RED}❌ ERRO: Espaço em disco crítico (${FREE_SPACE}K). Abortando build pesada para proteger OS.${NC}"
    exit 1
fi

# --- LOG HEARTBEAT (Pulse) ---
start_heartbeat() {
    local target_file=$1
    (
        while true; do
            sleep 10
            if [ -f "$target_file" ]; then
                SIZE=$(du -sh "$target_file" | cut -f1)
                echo -e "  💓 ${BLUE}[Pulse]${NC} Progresso I/O: ${BOLD}$SIZE${NC} gravados..."
            fi
        done
    ) &
    HEARTBEAT_PID=$!
}

stop_heartbeat() {
    [ -n "$HEARTBEAT_PID" ] && kill "$HEARTBEAT_PID" 2>/dev/null || true
}

fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
killall crompressor-novo 2>/dev/null || true
mkdir -p "$MNT_CROM" "$MNT_RO" "$UPPER_DIR" "$WORK_DIR" "$MERGED_DIR"

if [ -f "$SOURCE_CROM" ]; then
    log "🌌 Montando volume CROM: $SOURCE_CROM"
    CROMDB=$(find . -maxdepth 1 -name "*.cromdb" -type f | head -1)
    
    if [ -n "$CROMDB" ]; then
        log "🚀 Disparando Motor CROM (Fundo)..."
        "$CROM_BIN" mount -i "$SOURCE_CROM" -m "$MNT_CROM" -c "$CROMDB" --cache 512 2>&1 | tee -a "$LOG_FILE" &
        CROM_PID=$!
        sleep 3
        
        FILE_IN_CROM="$APP_NAME"
        if [ -n "$FILE_IN_CROM" ] && [ -f "$MNT_CROM/$FILE_IN_CROM" ]; then
            log "  2️⃣ [SquashFuse] Expandindo blocos em Árvore (Target: $FILE_IN_CROM)..."
            squashfuse "$MNT_CROM/$FILE_IN_CROM" "$MNT_RO" 2>/dev/null || true
        else
            log "${RED}⚠️  CROM Montou mas o arquivo '$FILE_IN_CROM' não apareceu ou é inválido.${NC}"
        fi
    else
        log "${RED}⚠️  .cromdb não encontrado. Falha FUSE provável.${NC}"
    fi
else
    log "ℹ️  Sem arquivo .crom — usando apenas OverlayFS."
fi

# Heartbeat do Pack/Compile (Se estiver gerando o .crom no launch.sh individual)
# Chamamos apenas se o script individual disparar o pack.
# Para o monitor geral, o progresso agora sairá no terminal pois tiramos o >/dev/null.

fuse-overlayfs -o lowerdir="$MNT_RO",upperdir="$UPPER_DIR",workdir="$WORK_DIR" "$MERGED_DIR" 2>/dev/null
if mountpoint -q "$MERGED_DIR" 2>/dev/null; then
    log "${GREEN}✅ Ambiente virtual montado em '$MERGED_DIR'${NC}"
else
    log "${YELLOW}⚠️  OverlayFS indisponível — executando teste direto.${NC}"
    MERGED_DIR="."
fi

log ""
log "${BOLD}${BLUE}🧪 EXECUTANDO TESTE: $APP_NAME${NC}"
log "--------------------------------------------------------"

if [ -f "./test_logic.sh" ]; then
    START_TIME=$(date +%s%N)
    ( cd "$MERGED_DIR" 2>/dev/null || true; bash "$OLDPWD/test_logic.sh" ) 2>&1 | tee -a "$LOG_FILE"
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

log ""
log "📊 Tamanho do diretório de trabalho:"
du -sh "$UPPER_DIR" 2>/dev/null | tee -a "$LOG_FILE" || true
log ""
log "📁 Log salvo em: $LOG_FILE"

fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
stop_heartbeat
[ -n "$CROM_PID" ] && kill -SIGINT "$CROM_PID" 2>/dev/null || true
log "${GREEN}🔒 Ambiente desmontado.${NC}"
log "========================================================"
