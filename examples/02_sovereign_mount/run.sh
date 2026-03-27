#!/bin/bash
set -e

echo "[*] Preparando cenário..."
echo "ARQUIVO ULTRA SECRETO" > segredo.txt
head -c 1M </dev/urandom >> segredo.txt
../../crompressor train --input segredo.txt --output sovereign.cromdb
../../crompressor pack --input segredo.txt --codebook sovereign.cromdb --output vault.crom

MNT_POINT="/tmp/trom_mnt_test"
mkdir -p "$MNT_POINT"

echo "[*] Montando com Sovereignty... (FUSE)"
../../crompressor mount --input vault.crom --codebook sovereign.cromdb --mountpoint "$MNT_POINT" &
MNT_PID=$!

sleep 2
echo "[*] Arquivo lido de dentro da montagem FUSE:"
cat "$MNT_POINT/segredo.txt"
echo "----------------------------------------"
echo "[!!!] MANTENHA ESTA ABA ABERTA."
echo "[!!!] EM OUTRA ABA, DIGITE O COMANDO ABAIXO E VEJA A MÁGICA:"
echo "rm examples/02_sovereign_mount/sovereign.cromdb"
echo "Esta montagem cairá num piscar de olhos."

# Aguarda indefinidamente até o processo de montagem cair
wait $MNT_PID
echo "[*] O PROCESSO CAIU! A pasta virtual foi explodida pelas regras da Soberania!"
