#!/bin/bash
# ==============================================================
# APP DATABASE VFS: SQLite sobre FUSE CASCADING
# Camadas: CROM Mount -> SquashFuse -> OverlayFs
# ==============================================================

APP_NAME="app_postgres_vfs"
BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
TIMESTAMP=$(date '+%Y-%m-%d_%H-%M-%S')
LOG_DIR="./logs"; LOG_FILE="$LOG_DIR/${TIMESTAMP}.log"

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
BOLD='\033[1m'; NC='\033[0m'
log() { echo -e "$1"; echo -e "$1" | sed 's/\x1b\[[0-9;]*m//g' >> "$LOG_FILE"; }
mkdir -p "$LOG_DIR"

log "========================================================"
log "🗄️  LABORATÓRIO: $APP_NAME (via SQLite3 Nativo)"
log "📅  $TIMESTAMP"
log "========================================================"

if ! command -v sqlite3 &>/dev/null; then log "${RED}❌ sqlite3 não encontrado${NC}"; exit 1; fi
if [ ! -x "$CROM_BIN" ]; then log "${RED}❌ Motor CROM não encontrado!${NC}"; exit 1; fi

# ── FASE 1: GERAÇÃO CROM (MONÓLITO) ──
if [ ! -f "$CROM_FILE" ]; then
    log "  📦 [Camada 1] Escrevendo SquashFS Dummy..."
    mkdir -p _db_base && echo "Identidade BD VFS" > _db_base/id.txt && dd if=/dev/urandom of=_db_base/sample.bin bs=4K count=8 status=none
    mksquashfs _db_base "$SQSH_FILE" -noI -noD -noX -noF -no-xattrs >/dev/null

    log "  🧠 [Motor] Treinando Codebook..."
    "$CROM_BIN" train -i "$SQSH_FILE" -o "$CROMDB_FILE" -s 256 >/dev/null 2>&1 || true

    log "  📥 [Motor] Compilando Compressão LSH..."
    "$CROM_BIN" pack -i "$SQSH_FILE" -o "$CROM_FILE" -c "$CROMDB_FILE" >/dev/null 2>&1
    rm -rf _db_base
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

DB_FILE="$MERGED_DIR/test_db.sqlite"

log ""
log "${BOLD}🧪 Teste: Operações transacionais ACID sobre Múltiplos VFS...${NC}"
START=$(date +%s%N)

log "  1. Criando banco OverlayFs..."
sqlite3 "$DB_FILE" <<EOF 2>&1 | tee -a "$LOG_FILE"
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS vfs_test (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT, ts DATETIME DEFAULT CURRENT_TIMESTAMP);
INSERT INTO vfs_test (data) VALUES ('Crompressor BD Write');
EOF
EXIT_CODE=${PIPESTATUS[0]}

if [ $EXIT_CODE -eq 0 ]; then
    log "  2. Realizando consulta LSH..."
    sqlite3 "$DB_FILE" "SELECT * FROM vfs_test;" 2>&1 | tee -a "$LOG_FILE"
    EXIT_CODE=${PIPESTATUS[0]}
fi

END=$(date +%s%N); ELAPSED=$(( (END - START) / 1000000 ))

if [ $EXIT_CODE -eq 0 ]; then
    log "${GREEN}✅ RESULTADO: PASS — SQLite3 operou sobre Cascading (${ELAPSED}ms)${NC}"
else
    log "${RED}❌ RESULTADO: FAIL (${ELAPSED}ms)${NC}"
fi

log "🔒 Desmontando VFS Cascading..."
fusermount -uz "$MERGED_DIR" 2>/dev/null || true
fusermount -uz "$MNT_RO" 2>/dev/null || true
fusermount -uz "$MNT_CROM" 2>/dev/null || true
kill -SIGINT $(jobs -p) 2>/dev/null || true
log "========================================================"
