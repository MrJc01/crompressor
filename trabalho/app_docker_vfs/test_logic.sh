echo "Docker VFS: Verificação de Camadas de Container"
mkdir -p ./merged/overlay2
echo "{\"ID\": \"sha256:7bc89d7\", \"layers\": 4}" > ./merged/overlay2/metadata.json
ls -lh ./merged/overlay2
