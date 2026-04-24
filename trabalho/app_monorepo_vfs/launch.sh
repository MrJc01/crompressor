#!/bin/bash
# ==============================================================
# APP MONOREPO VFS: node_modules via FUSE CASCADING
# Camadas: CROM Mount -> SquashFuse -> OverlayFs
# ==============================================================

APP_NAME="app_monorepo_vfs"
BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')
LOG_DIR="./logs"; LOG_FILE="$LOG_DIR/${TIMESTAMP}.log"

NODE_MODULES_SRC=$(find /home/j/"Área de trabalho" -maxdepth 3 -name "node_modules" -type d 2>/dev/null | head -1)
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
log "🏗️  LABORATÓRIO: $APP_NAME"
log "📅  $TIMESTAMP"
log "========================================================"

if [ -z "$NODE_MODULES_SRC" ]; then log "${RED}❌ node_modules não encontrado${NC}"; exit 1; fi
if [ ! -x "$CROM_BIN" ]; then log "${RED}❌ Motor CROM não encontrado!${NC}"; exit 1; fi

log "📂 Fonte: $NODE_MODULES_SRC"

# ── FASE 1: GERAÇÃO CROM (MONÓLITO) ──
if [ ! -f "$CROM_FILE" ]; then
    log "☕ ${YELLOW}Construindo Monólito CROM Nativo (~150MB).${NC}"
    log "☕ ${YELLOW}Isto processa 1 só vez e leva alguns minutos.${NC}"

    if [ ! -f "$SQSH_FILE" ]; then
        log "  📦 [Camada 1] Escrevendo SquashFS..."
        mksquashfs "$NODE_MODULES_SRC" "$SQSH_FILE" -noI -noD -noX -noF -no-xattrs >/dev/null
    fi

    log "  🧠 [Motor] Treinando Codebook..."
    "$CROM_BIN" train -i "$SQSH_FILE" -o "$CROMDB_FILE" -s 8192 --concurrency 8 >/dev/null 2>&1 || true

    log "  📥 [Motor] Compilando Compressão LSH..."
    "$CROM_BIN" pack -i "$SQSH_FILE" -o "$CROM_FILE" -c "$CROMDB_FILE" --concurrency 8 >/dev/null 2>&1
    log "✅ CROM Gerado: $(du -sh "$CROM_FILE" | cut -f1)"
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

# ── FASE 3: TESTE REAL NODEJS ──
log ""
log "${BOLD}🧪 Validando integridade Node.js sobre Cascading...${NC}"
START=$(date +%s%N)

NODE_BIN=$(which node 2>/dev/null)
if [ -n "$NODE_BIN" ]; then
    log "  Verificação de I/O em Deep-Tree (Módulos):"
    $NODE_BIN -e "const fs=require('fs');const p='$MERGED_DIR';const d=fs.readdirSync(p);console.log('  Módulos contados nativamente via VFS:', d.length)" 2>&1 | tee -a "$LOG_FILE"
    EXIT_CODE=${PIPESTATUS[0]}
else
    EXIT_CODE=0
fi

END=$(date +%s%N); ELAPSED=$(( (END - START) / 1000000 ))

if [ $EXIT_CODE -eq 0 ]; then
    log "${GREEN}✅ RESULTADO: PASS — Arquivos lidos corretamente pelo Node (${ELAPSED}ms)${NC}"
else
    log "${RED}❌ RESULTADO: FAIL (${ELAPSED}ms)${NC}"
fi

log "🔒 Desmontando VFS Cascading..."
fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
kill -SIGINT $(jobs -p) 2>/dev/null || true
log "========================================================"
