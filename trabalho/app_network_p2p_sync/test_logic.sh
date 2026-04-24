echo "Network P2P Sync: Calculando Diferencial Delta"
echo "Simulando sync de 15mb de dicionários de rede..."
dd if=/dev/urandom of=./merged/network_dict.bin bs=1M count=15 status=none
sha256sum ./merged/network_dict.bin
