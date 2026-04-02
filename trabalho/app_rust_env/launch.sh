#!/bin/bash
# ==============================================================
# APP RUST ENV: Compilação Rust via FUSE CASCADING
# Camadas: CROM Mount -> SquashFuse -> OverlayFs
# ==============================================================

APP_NAME="app_rust_env"
BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')
LOG_DIR="./logs"; LOG_FILE="$LOG_DIR/${TIMESTAMP}.log"

CARGO_SRC="/home/j/.cargo"
SQSH_FILE="./base.sqsh"
CROMDB_FILE="./base.cromdb"
CROM_FILE="./base.crom"

# Camadas Cascading
MNT_CROM="./mnt_crom"
MNT_RO="./mnt_ro"
UPPER_DIR="./vfs_upper"
WORK_DIR="./vfs_work"
MERGED_DIR="./merged"

GREEN='\033[0;32m'; RED='\033[0;31m'; BLUE='\033[0;34m'
YELLOW='\033[1;33m'; BOLD='\033[1m'; NC='\033[0m'
log() { echo -e "$1"; echo -e "$1" | sed 's/\x1b\[[0-9;]*m//g' >> "$LOG_FILE"; }
mkdir -p "$LOG_DIR"

log "========================================================"
log "🦀 LABORATÓRIO: $APP_NAME"
log "📅  $TIMESTAMP"
log "========================================================"

# --- DISK SAFETY GUARD (SRE) ---
FREE_SPACE=$(df -k . | awk 'NR==2 {print $4}')
if [ "$FREE_SPACE" -lt 2097152 ]; then # 2GB
    log "${RED}❌ ERRO: Espaço em disco insuficiente (${FREE_SPACE}K). build Rust abortada.${NC}"
    exit 1
fi

# --- LOG HEARTBEAT (Pulse) ---
start_heartbeat() {
    local target_file=$1
    ( while true; do sleep 10; [ -f "$target_file" ] && echo -e "  💓 ${BLUE}[Pulse]${NC} Pack Rust: ${BOLD}$(du -sh "$target_file" | cut -f1)${NC}..." ; done ) &
    HEARTBEAT_PID=$!
}
stop_heartbeat() { [ -n "$HEARTBEAT_PID" ] && kill "$HEARTBEAT_PID" 2>/dev/null || true; }

# ── FASE 1: GERAÇÃO CROM (MONÓLITO) ──
if [ ! -f "$CROM_FILE" ]; then
    log "☕ ${YELLOW}Construindo Monólito CROM Nativo (~700MB).${NC}"
    log "☕ ${YELLOW}Isto processa 1 só vez e leva alguns minutos.${NC}"

    if [ ! -f "$SQSH_FILE" ]; then
        log "  📦 [Camada 1] Escrevendo SquashFS..."
        # Nomeamos o objeto como 'base' para o CROM expor como 'base'
        mksquashfs "$CARGO_SRC" "./base" -noI -noD -noX -noF -no-xattrs >/dev/null
        SQSH_FILE="./base"
    fi

    log "  🧠 [Motor] Treinando Codebook..."
    "$CROM_BIN" train -i "$SQSH_FILE" -o "$CROMDB_FILE" -s 8192 --concurrency 4 2>&1 | tee -a "$LOG_FILE" || true

    log "  📥 [Motor] Compilando Compressão LSH..."
    "$CROM_BIN" pack -i "$SQSH_FILE" -o "$CROM_FILE" -c "$CROMDB_FILE" --concurrency 4 2>&1 | tee -a "$LOG_FILE" &
    CROM_PACK_PID=$!
    start_heartbeat "$CROM_FILE"
    wait $CROM_PACK_PID
    stop_heartbeat
    log "✅ CROM Gerado: $(du -sh "$CROM_FILE" | cut -f1)"
    rm -f "./base" # Limpa a imagem temporária
fi

# ── FASE 2: FUSE CASCADING ──
log "${BOLD}${BLUE}🌌 Montando FUSE Cascading...${NC}"

fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
killall crompressor-novo 2>/dev/null || true
mkdir -p "$MNT_CROM" "$MNT_RO" "$UPPER_DIR" "$WORK_DIR" "$MERGED_DIR"

log "  1️⃣ [CROM VFS] Montando Virtual Block..."
"$CROM_BIN" mount -i "$CROM_FILE" -m "$MNT_CROM" -c "$CROMDB_FILE" --cache 512 &
sleep 2

FILE_IN_CROM="base"

log "  2️⃣ [SquashFuse] Expandindo blocos em Árvore..."
squashfuse "$MNT_CROM/$FILE_IN_CROM" "$MNT_RO"

log "  3️⃣ [OverlayFS] Fundindo permissões Write..."
fuse-overlayfs -o lowerdir="$MNT_RO",upperdir="$UPPER_DIR",workdir="$WORK_DIR" "$MERGED_DIR"
if ! mountpoint -q "$MERGED_DIR"; then log "${RED}❌ FUSE Cascading falhou!${NC}"; exit 1; fi

log "${GREEN}✅ Infraestrutura Tri-Camada Estabelecida.${NC}"

# ── FASE 3: TESTE RUST ──
log ""
log "${BOLD}🧪 Executando Rust NATIVO no Módulo CROM Cascading...${NC}"
START=$(date +%s%N)

export CARGO_HOME="$(pwd)/$MERGED_DIR"
export PATH="$CARGO_HOME/bin:$PATH"

log "  cargo --version (via Cascading):"
"$MERGED_DIR/bin/cargo" --version 2>&1 | tee -a "$LOG_FILE"
EXIT_CODE=${PIPESTATUS[0]}

END=$(date +%s%N); ELAPSED=$(( (END - START) / 1000000 ))

if [ $EXIT_CODE -eq 0 ]; then
    log "${GREEN}✅ RESULTADO: PASS — Binários auditados sobre 3 Camadas FUSE (${ELAPSED}ms)${NC}"
else
    log "${RED}❌ RESULTADO: FAIL (${ELAPSED}ms)${NC}"
fi

log "🔒 Desmontando VFS Cascading..."
fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
kill -SIGINT $(jobs -p) 2>/dev/null || true
log "========================================================"
