echo "SRE I/O Benchmark: Teste de Escrita em VFS vs Buffer"
time dd if=/dev/zero of=./merged/test_100m.bin bs=1M count=100 status=progress
echo "--------------------------------------------------------"
echo "Lendo arquivo de volta para validar integridade:"
sha256sum ./merged/test_100m.bin
