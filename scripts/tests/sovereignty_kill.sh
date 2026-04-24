#!/bin/bash
# sovereignty_kill.sh: Auto-Unmount do FUSE
# Valida se a montagem cai quando a fonte do codebook é apagada.

set -e

WORKSPACE="/tmp/crom_sov_$$"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

CROM_BIN="/home/j/Área de trabalho/crompressor/crompressor"
CB_PATH="$WORKSPACE/sov.cromdb"
VAULT_PATH="$WORKSPACE/vault.crom"
MNT_POINT="$WORKSPACE/mnt"

echo "DADOS SECRETOS E PODEROSOS" > secret.txt
head -c 1M </dev/urandom >> secret.txt
echo "[*] Treinando Codebook e empacotando..."
"$CROM_BIN" train --input "$WORKSPACE/secret.txt" --output "$CB_PATH" 2>/dev/null || true
"$CROM_BIN" pack --input "$WORKSPACE/secret.txt" --codebook "$CB_PATH" --output "$VAULT_PATH" 2>/dev/null || true

echo "[*] Montando vault VFS via FUSE"
mkdir -p "$MNT_POINT"

# Roda em background
"$CROM_BIN" mount --input "$VAULT_PATH" --codebook "$CB_PATH" --mountpoint "$MNT_POINT" &
MNT_PID=$!

sleep 2

# Testa se montou lendo o arquivo de lá de dentro
if cat "$MNT_POINT/vault" | grep "SECRETOS" >/dev/null 2>&1; then
    echo "[OK] Aquivo lido com sucesso do FUSE Mount"
else
    echo "[FAIL] Não foi possível ler o arquivo do FUSE Mount!"
    kill $MNT_PID 2>/dev/null || true
    exit 1
fi

echo "[*] Acionando Sovereignty Kill (Apagando Codebook)"
rm -f "$CB_PATH"

sleep 3

if mount | grep "$MNT_POINT" >/dev/null; then
    echo "[FAIL] Ponto de montagem não foi desmontado após 3 segundos!"
    kill $MNT_PID 2>/dev/null || true
    # Limpeza forçada FUSE
    fusermount -u "$MNT_POINT" 2>/dev/null || true
    exit 1
else
    echo "[OK] Ponto de montagem desapareceu instantaneamente. Sovereignty garantida."
    kill $MNT_PID 2>/dev/null || true
fi

# Cleanup
rm -rf "$WORKSPACE"
