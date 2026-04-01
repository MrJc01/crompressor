#!/bin/bash
# ==============================================================
# Script de Implantação: Minecraft Client VFS Singularity (V9)
# ==============================================================
set -e

BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
CB_DB="$BASE_DIR/minecraft_client.cromdb"
OUTPUT_DIR="$BASE_DIR/trabalho/minecraft_client/vfs_output"

mkdir -p "$OUTPUT_DIR"

# Função para montar e sincronizar
deploy_vfs() {
    local source_dir="$1"
    local pool_name="$2"
    local mount_point="/tmp/vfs_${pool_name}"
    local pool_file="${OUTPUT_DIR}/${pool_name}_pool.crom"

    echo "🔗 [MONTANDO] $pool_name no $mount_point..."
    # Limpeza de montagens anteriores
    fusermount -u "$mount_point" 2>/dev/null || true
    rm -rf "$mount_point" && mkdir -p "$mount_point"

    # Inicia CromFS em background
    "$CROM_BIN" cromfs -m "$mount_point" -o "$pool_file" -c "$CB_DB" &
    local daemon_pid=$!
    sleep 5 # Espera FUSE estabilizar

    echo "🚚 [SINCRONIZANDO] Dados de $source_dir -> $mount_point..."
    # rsync preservando atributos, mas passando pelo motor CROM
    rsync -aP "$source_dir/" "$mount_point/"

    echo "✅ [SUCESSO] $pool_name sincronizado via Crompressor."
    
    # Backup e Link Simbólico
    local bak_dir="${source_dir}.bak_sre"
    if [ ! -d "$bak_dir" ]; then
        echo "🛡️  [BACKUP] Criando backup de segurança em $bak_dir..."
        mv "$source_dir" "$bak_dir"
        echo "🔗 [LINKING] Criando link simbólico para o VFS..."
        ln -s "$mount_point" "$source_dir"
    fi
}

# 1. Processar .minecraft (~1GB)
deploy_vfs "/home/j/.minecraft" "minecraft"

# 2. Processar .tlauncher (~350MB)
deploy_vfs "/home/j/.tlauncher" "tlauncher"

echo "--------------------------------------------------------"
echo "🌟 [VFS SOBERANA ONLINE]"
echo "Tudo pronto! Agora você pode abrir o TLauncher normalmente:"
echo "java -jar '$BASE_DIR/trabalho/minecraft_client/tlauncher/usr/games/tlauncher/starter-core.jar'"
echo ""
echo "O launcher pensará que está no HD, mas estará lendo via Crompressor VOS."
echo "Monitore o FPS e os logs de I/O!"
echo "--------------------------------------------------------"
