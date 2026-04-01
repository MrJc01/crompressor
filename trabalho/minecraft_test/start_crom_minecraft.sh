#!/bin/bash
# ==============================================================
# Laboratório de Teste SRE - JVM no Crompressor FUSE VFS
# ==============================================================
set -e

echo "=== 🚀 INICIANDO TESTE DO MINECRAFT OUT-OF-CORE ==="

# Constantes de Caminho
WORK_DIR="/home/j/Área de trabalho/crompressor/trabalho/minecraft_test"
CROM_BIN="/home/j/Área de trabalho/crompressor/crompressor-novo"
MNT_DIR="/tmp/mine_vfs"

cd "$WORK_DIR"

# Verifica se o jar do Minecraft existe
if [ ! -f "server.jar" ]; then
    echo "❌ Erro Fatal: server.jar (PaperMC) não localizado em $WORK_DIR"
    exit 1
fi

echo "🧠 [FASE 0] Treinando Codebook Neural para densidade Minecraft..."
# Criamos um codebook específico para o Jar para garantir Hit Rate de 100%
"$CROM_BIN" train -i . -o minecraft.cromdb

echo "📦 [FASE 1] Empacotando Carga (Pack) com o novo Codebook..."
"$CROM_BIN" pack -i server.jar -o server.jar.crom -c minecraft.cromdb

echo "🧹 [SRE] Limpando alocações fantasmas no FUSE..."
fusermount -u "$MNT_DIR" 2>/dev/null || true
rm -rf "$MNT_DIR" && mkdir -p "$MNT_DIR"

echo "🔮 [FASE 2] Montando Daemon FUSE (VFS Singularity)..."
# Agora passamos todas as flags obrigatórias: -i (input), -m (mountpoint), -c (codebook)
"$CROM_BIN" mount -i server.jar.crom -m "$MNT_DIR" -c minecraft.cromdb &
MOUNT_PID=$!

# Aguarda a tabela de inode do FUSE subir ao Kernel
sleep 4 

echo "✅ Arquivo VFS Detectado (Projeção JIT):"
ls -lh "$MNT_DIR"

echo "📜 [FASE 3] Preparando ambiente Server..."
echo "eula=true" > eula.txt

echo "🔥 [FASE 4] IGNIZÃO DA MÁQUINA VIRTUAL JAVA..."
# Executa o JAR projetado do Diretório Virtual
java -Xms1G -Xmx1G -jar "$MNT_DIR/server.jar" --nogui || true

echo "--------------------------------------------------------"
echo "🛑 [ENCERRAMENTO] Matando instâncias zumbis FUSE..."
kill $MOUNT_PID 2>/dev/null || true
fusermount -u "$MNT_DIR" 2>/dev/null || true
echo "✅ Teste Neural FUSE Concluído. PC preservado."
