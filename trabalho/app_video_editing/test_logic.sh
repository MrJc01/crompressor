echo "Video Editing: Simulação de Proxies"
echo "Gerando proxy dummy de 500kb..."
dd if=/dev/urandom of=./merged/shot_001_proxy.mp4 bs=500k count=1 status=none
sha256sum ./merged/shot_001_proxy.mp4
