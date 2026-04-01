#!/bin/bash
# ==============================================================
# SRE FUSE CASCADING: SquashFS + fuse-overlayfs + Crompressor V9
# Arquitetura de 3 Camadas de Filesystem para Paging/Deduplicação Abstrata
# ==============================================================
set -e

# Assegura Ferramentas de Orquestração Abstratas de SO (User-Mode FS)
echo "========================================================"
echo "🛡️  [FASE 0] Verificação de Módulos (Nível Kernel)"
echo "O script instalará dependências apt, caso necessário."
if ! command -v mksquashfs >/dev/null || ! command -v squashfuse >/dev/null || ! command -v fuse-overlayfs >/dev/null; then
    echo "Instalando drivers de abstração (mksquashfs, squashfuse, fuse-overlayfs)..."
    sudo apt-get update && sudo apt-get install -y squashfs-tools squashfuse fuse-overlayfs
fi

BASE_DIR="/home/j/Área de trabalho/crompressor"
CROM_BIN="$BASE_DIR/crompressor-novo"
OUTPUT_DIR="$BASE_DIR/trabalho/minecraft_client/vfs_output"
CB_DB="$OUTPUT_DIR/minecraft_cascade.cromdb"

SQSH_MC="$OUTPUT_DIR/mc.sqsh"
SQSH_TL="$OUTPUT_DIR/tl.sqsh"

# Define pontos virtuais
CROM_MC_MNT="/tmp/vfs_crom_mc"
CROM_TL_MNT="/tmp/vfs_crom_tl"

MNT_RO_MC="/tmp/vfs_mc_ro"
MNT_RO_TL="/tmp/vfs_tl_ro"

MNT_UP_MC="/tmp/vfs_up_mc"
MNT_WK_MC="/tmp/vfs_wk_mc"
MNT_UP_TL="/tmp/vfs_up_tl"
MNT_WK_TL="/tmp/vfs_wk_tl"

echo "========================================================"
echo "🧹 [SRE] Esmagando conexões FUSE Zumbis e limpando alicerces..."
killall -9 crompressor-novo 2>/dev/null || true
fusermount -uz /home/j/.minecraft 2>/dev/null || true
fusermount -uz /home/j/.tlauncher 2>/dev/null || true
fusermount -uz "$MNT_RO_MC" 2>/dev/null || true
fusermount -uz "$MNT_RO_TL" 2>/dev/null || true
fusermount -uz "$CROM_MC_MNT" 2>/dev/null || true
fusermount -uz "$CROM_TL_MNT" 2>/dev/null || true

rm -f /home/j/.minecraft /home/j/.tlauncher 2>/dev/null || true
[ -d "/home/j/.minecraft.bak_sre" ] && mv "/home/j/.minecraft.bak_sre" "/home/j/.minecraft"
[ -d "/home/j/.tlauncher.bak_sre" ] && mv "/home/j/.tlauncher.bak_sre" "/home/j/.tlauncher"

mkdir -p "$CROM_MC_MNT" "$CROM_TL_MNT" "$MNT_RO_MC" "$MNT_RO_TL" 
mkdir -p "$MNT_UP_MC" "$MNT_WK_MC" "$MNT_UP_TL" "$MNT_WK_TL" "$OUTPUT_DIR"

echo "========================================================"
echo "🎯 [FASE 1] Empacotamento Maciço SquashFS (1.3GB -> Monólitos)"
echo "Isso elimina o problema hierárquico, fundindo tudo."
# Agrupando em Read-Only image de latência O(1) nativa sem compressão redundante 
if [ ! -f "$SQSH_MC" ]; then
   mksquashfs /home/j/.minecraft "$SQSH_MC" -noI -noD -noX -noF -no-xattrs
   mksquashfs /home/j/.tlauncher "$SQSH_TL" -noI -noD -noX -noF -no-xattrs
else
   echo "-> Imagens SquashFS monolitizadas já detectadas na arquitetura."
fi

echo "========================================================"
echo "🧠 [FASE 2] Rede Neural do Crompressor (Train & Pack)"
cd "$OUTPUT_DIR"
if [ ! -f "mc.crom" ]; then
   "$CROM_BIN" train -i "$SQSH_MC" -o "$CB_DB" --size 4096 || true
   # Pack converte o monólito SquashFS em um Matrix Petabyte-ready CROM
   "$CROM_BIN" pack -i "$SQSH_MC" -o "$OUTPUT_DIR/mc.crom" -c "$CB_DB"
   "$CROM_BIN" pack -i "$SQSH_TL" -o "$OUTPUT_DIR/tl.crom" -c "$CB_DB"
else
   echo "-> Camadas .CROM já pré-empacotadas detectadas."
fi

echo "========================================================"
echo "🌌 [FASE 3] FUSE CASCADING - Orquestração de 3 Camadas"
echo "Camada 1: CROM (Sobe o FUSE do File Abstrato em Milisegundos)"
"$CROM_BIN" mount -i "$OUTPUT_DIR/mc.crom" -m "$CROM_MC_MNT" -c "$CB_DB" --cache 1536 &
"$CROM_BIN" mount -i "$OUTPUT_DIR/tl.crom" -m "$CROM_TL_MNT" -c "$CB_DB" --cache 512 &
sleep 4 # Tabela de I/O

# Localizando arquivos abstratos reconstruídos
FILE_MC_IN_CROM=$(ls "$CROM_MC_MNT" | head -n 1)
FILE_TL_IN_CROM=$(ls "$CROM_TL_MNT" | head -n 1)

echo "Camada 2: SquashFuse (Extrai o File Abstrato em Subpastas virtuais)"
squashfuse "$CROM_MC_MNT/$FILE_MC_IN_CROM" "$MNT_RO_MC"
squashfuse "$CROM_TL_MNT/$FILE_TL_IN_CROM" "$MNT_RO_TL"

echo "Camada 3: Fuse-OverlayFs (Read/Write Mutacional Virtual)"
# Move reais para backup master (SRE rollback)
mv /home/j/.minecraft /home/j/.minecraft.bak_sre
mv /home/j/.tlauncher /home/j/.tlauncher.bak_sre
mkdir -p /home/j/.minecraft /home/j/.tlauncher

# Junta o Read-Only CROM com a pasta Temporária R/W e simula a original!
fuse-overlayfs -o lowerdir="$MNT_RO_MC",upperdir="$MNT_UP_MC",workdir="$MNT_WK_MC" /home/j/.minecraft
fuse-overlayfs -o lowerdir="$MNT_RO_TL",upperdir="$MNT_UP_TL",workdir="$MNT_WK_TL" /home/j/.tlauncher

echo "========================================================"
echo "✅ INFRAESTRUTURA HÍBRIDA (CROM + FUSE + SQSH + OVERLAY) ESTABELECIDA."
echo "   Inicie o jogo agora usando a interface física ou rodando este comando:"
echo "   java -jar $BASE_DIR/trabalho/minecraft_client/tlauncher/usr/games/tlauncher/starter-core.jar"
echo "--------------------------------------------------------"
echo "Para desfazer a teia virtual após o Teste FPS de Stuttering, basta dar:"
echo "fusermount -u /home/j/.minecraft && fusermount -u /home/j/.tlauncher"
echo "e voltar as pastas com '.bak_sre' pro lugar."
