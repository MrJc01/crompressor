echo "Auditoria de Segurança: Gerando Manifesto de Hashes do Volume"
find ./merged -type f -exec sha256sum {} \; > ./merged/security_manifest.log
echo "Manifesto Criado em ./merged/security_manifest.log"
