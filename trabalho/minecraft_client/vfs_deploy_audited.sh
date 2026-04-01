#!/bin/bash
# ==============================================================
# Script de Implantação SOBERANA: Minecraft Client VFS (Verbose)
# Função: Ingerir 1.3GB e rodar Minecraft 100% sobre FUSE CROM V9
# ==============================================================
set -e

BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
CB_DB="$BASE_DIR/minecraft_client.cromdb"
OUTPUT_DIR="$BASE_DIR/trabalho/minecraft_client/vfs_output"

echo "========================================================"
echo "🛑 [FASE 1] Limpeza de Segurança..."
# NUNCA USAR pkill -f crompressor AQUI. Usa killall no nome DE EXATIDÃO.
killall -9 crompressor-novo 2>/dev/null || true
fusermount -u /tmp/vfs_minecraft 2>/dev/null || true
fusermount -u /tmp/vfs_tlauncher 2>/dev/null || true

# Restaurando a integridade física em caso de crash anterior (desfaz os links simbólicos)
rm -f /home/j/.minecraft /home/j/.tlauncher 2>/dev/null || true
[ -d "/home/j/.minecraft.bak_sre" ] && mv "/home/j/.minecraft.bak_sre" "/home/j/.minecraft"
[ -d "/home/j/.tlauncher.bak_sre" ] && mv "/home/j/.tlauncher.bak_sre" "/home/j/.tlauncher"

mkdir -p /tmp/vfs_minecraft /tmp/vfs_tlauncher
mkdir -p "$OUTPUT_DIR"

echo "========================================================"
echo "🔮 [FASE 2] Subindo Daemons FUSE do CROM V9..."
# Montando e Redirecionando os logs do FUSE para arquivos rastreáveis
"$CROM_BIN" cromfs -m /tmp/vfs_minecraft -o "$OUTPUT_DIR/minecraft_vfs_pool.crom" -c "$CB_DB" > /tmp/mc_fuse_engine.log 2>&1 &
"$CROM_BIN" cromfs -m /tmp/vfs_tlauncher -o "$OUTPUT_DIR/tlauncher_vfs_pool.crom" -c "$CB_DB" > /tmp/tl_fuse_engine.log 2>&1 &

# Dando tempo à Tabela Inode do Kernel
sleep 4 

if mountpoint -q /tmp/vfs_minecraft && mountpoint -q /tmp/vfs_tlauncher; then
    echo "✅ [VFS ONLINE] Montagens detectadas com sucesso no Kernel."
else
    echo "❌ [ERRO CRÍTICO] Falha. O VFS não ancorou. Leia os logs de FUSE."
    cat /tmp/mc_fuse_engine.log
    exit 1
fi

echo "========================================================"
echo "🚚 [FASE 3] Sincronização e Ingestão Paging (Pode demorar)"
echo "Você verá os arquivos sendo injetados na malha de LSH:"

# Usando o rsync em modo verbose para você auditar a injeção ao VFS
echo "--> Ingerindo .tlauncher... (Aprox. 350MB)"
rsync -avP /home/j/.tlauncher/ /tmp/vfs_tlauncher/

echo "--> Ingerindo .minecraft... (Aprox. 1.0GB)"
rsync -avP /home/j/.minecraft/ /tmp/vfs_minecraft/

echo "✅ [INGESTÃO SUCESSO] Matriz LSH carregada."

echo "========================================================"
echo "🔗 [FASE 4] Inversão de Rotas (Shadowing)..."
mv /home/j/.minecraft /home/j/.minecraft.bak_sre
ln -s /tmp/vfs_minecraft /home/j/.minecraft

mv /home/j/.tlauncher /home/j/.tlauncher.bak_sre
ln -s /tmp/vfs_tlauncher /home/j/.tlauncher

echo "✅ A partir de agora, o Sistema Operacional acha que a pasta está física,"
echo "   mas cada I/O é interceptado pelo CROM V9 VFS."
echo ""
echo "========================================================"
echo "🚀 TUDO PRONTO. APERTE ENTER PARA ABRIR O TLAUNCHER SOBRE FUSE CROM."
read -p "..."

# Lançando o jogo com o redirecionamento.
java -jar "$BASE_DIR/trabalho/minecraft_client/tlauncher/usr/games/tlauncher/starter-core.jar"

echo "========================================================"
echo "🧹 [RETORNO AO NORMAL] Jogo fechado. Restaurando seu SSD..."
rm -f /home/j/.minecraft /home/j/.tlauncher
mv /home/j/.minecraft.bak_sre /home/j/.minecraft
mv /home/j/.tlauncher.bak_sre /home/j/.tlauncher
fusermount -u /tmp/vfs_minecraft 2>/dev/null || true
fusermount -u /tmp/vfs_tlauncher 2>/dev/null || true
echo "✅ Sua máquina voltou à arquitetura original de Discos."
