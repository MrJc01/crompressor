#!/bin/bash
# ==============================================================
# SRE Master Launcher: CROM VFS + TLauncher
# Este script levanta a infraestrutura virtual de arquivos, inicia
# o jogo e desmonta tudo sozinho quando você fechar a janela.
# ==============================================================
set -e

BASE_DIR="/home/j/Área de trabalho/crompressor"
CLIENT_DIR="$BASE_DIR/trabalho/minecraft_client"

echo "========================================================"
echo "🚀 INICIANDO ENGINES CROM E MONTANDO FILE SYSTEMS..."
cd "$CLIENT_DIR"

# Chama o orquestrador que montará a arquitetura de 3 camadas
./vfs_squash_deploy.sh

echo "========================================================"
echo "🎮 INICIANDO O MINECRAFT VIA TLAUNCHER (OVERLAY FS)"
echo "   Nota: Deixe o Minecraft usar até 4.8GB (Tuning JVM via Menu)."
echo "   O terminal ficará ocupado até você fechar o Launcher/Jogo..."
echo "--------------------------------------------------------"

# Execução via Binário Físico da Raiz
JAR_PATH="${JAR_PATH:-$CLIENT_DIR/TLauncher.jar}"

if [ ! -f "$JAR_PATH" ]; then
    echo "🚨 Erro SRE: Arquivo físico '$JAR_PATH' não encontrado."
else
    echo "📌 JAR Físico Detectado: $JAR_PATH"
    # Roda o launcher e prende o terminal (Foreground)
    java -jar "$JAR_PATH"
fi

echo "========================================================"
echo "🛑 JOGO ENCERRADO. DESFAZENDO ARQUITETURA VFS (SRE TEARDOWN)..."
echo "Matando processos CROM e desmontando fusermount..."

killall -9 crompressor-novo 2>/dev/null || true
fusermount -uz /home/j/.minecraft 2>/dev/null || true
fusermount -uz /home/j/.tlauncher 2>/dev/null || true
fusermount -uz /tmp/vfs_mc_ro 2>/dev/null || true
fusermount -uz /tmp/vfs_tl_ro 2>/dev/null || true
fusermount -uz /tmp/vfs_crom_mc 2>/dev/null || true
fusermount -uz /tmp/vfs_crom_tl 2>/dev/null || true

# Restaurando backups originais passivamente
[ -d "/home/j/.minecraft.bak_sre" ] && rm -rf "/home/j/.minecraft" && mv "/home/j/.minecraft.bak_sre" "/home/j/.minecraft" 2>/dev/null || true
[ -d "/home/j/.tlauncher.bak_sre" ] && rm -rf "/home/j/.tlauncher" && mv "/home/j/.tlauncher.bak_sre" "/home/j/.tlauncher" 2>/dev/null || true

echo "✅ AMBIENTE TOTALMENTE RESTAURADO. O Sistema Físico voltou ao normal."
