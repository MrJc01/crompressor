#!/bin/bash
# ==============================================================
# Script de Implantação Soberana: Minecraft Client VFS Singularity
# ==============================================================
set -e

BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
CB_DB="$BASE_DIR/minecraft_client.cromdb"
OUTPUT_DIR="$BASE_DIR/trabalho/minecraft_client/vfs_output"
LOG_DIR="/tmp/crom_logs"

mkdir -p "$OUTPUT_DIR"
mkdir -p "$LOG_DIR"

# Função para limpeza
cleanup() {
    pkill -f "crompressor-novo cromfs" || true
    fusermount -u /tmp/vfs_minecraft 2>/dev/null || true
    fusermount -u /tmp/vfs_tlauncher 2>/dev/null || true
    rm -rf /tmp/vfs_minecraft /tmp/vfs_tlauncher
    mkdir -p /tmp/vfs_minecraft /tmp/vfs_tlauncher
}

cleanup

echo "🚀 [FASE 1] Iniciando Daemons CromFS..."
nohup "$CROM_BIN" cromfs -m /tmp/vfs_minecraft -o "$OUTPUT_DIR/minecraft_vfs_pool.crom" -c "$CB_DB" > "$LOG_DIR/mc_vfs.log" 2>&1 &
nohup "$CROM_BIN" cromfs -m /tmp/vfs_tlauncher -o "$OUTPUT_DIR/tlauncher_vfs_pool.crom" -c "$CB_DB" > "$LOG_DIR/tl_vfs.log" 2>&1 &

sleep 5

echo "📥 [FASE 2] Ingerindo Dados (1.3GB)..."
nohup rsync -a /home/j/.minecraft/ /tmp/vfs_minecraft/ > "$LOG_DIR/rsync_mc.log" 2>&1 &
nohup rsync -a /home/j/.tlauncher/ /tmp/vfs_tlauncher/ > "$LOG_DIR/rsync_tl.log" 2>&1 &

echo "Sincronização iniciada em background. Monitore via /tmp/vfs_minecraft"
