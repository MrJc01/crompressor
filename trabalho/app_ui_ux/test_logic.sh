echo "Simulando Carregamento de UI Assets..."
echo "Carregando 10 assets de 1MB pseudo-aleatórios..."
for i in {1..10}; do dd if=/dev/urandom of=./merged/asset_$i.png bs=1M count=1 status=none; done
echo "Assets carregados com sucesso."
