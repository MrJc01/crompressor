echo "Blender Assets: Gerenciando Texturas Brutas"
echo "Simulando assets de 2MB..."
dd if=/dev/urandom of=./merged/texture_diffuse.png bs=1M count=2 status=none
sha256sum ./merged/texture_diffuse.png
