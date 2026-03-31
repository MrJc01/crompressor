#!/bin/bash
set -e
echo "=== CROM V9 vs ZSTD - Benchmark Pós-Atualização (30/03/2026) ==="

FILES=("testdata/test_med.bin" "testdata/test_urandom.bin" "testdata/small.txt")

for file in "${FILES[@]}"; do
  echo -e "\n\n📊 === TESTANDO: $file ($(du -sh "$file" | cut -f1)) ==="

  # ZSTD baseline
  echo "→ zstd -19"
  /usr/bin/time -f "Tempo: %E | RAM: %M KB" zstd -19 -k -f "$file" -o "${file}.zst" || true

  # CROM com todas as otimizações V9
  echo "→ crompressor pack (V9 + Smart Passthrough)"
  /usr/bin/time -f "Tempo: %E | RAM: %M KB" ./crompressor-novo pack \
    -i "$file" \
    -o "${file}.crom" \
    --verbose 2>&1 | tee crom_log_${file##*/}.txt || true

  # Tamanhos
  echo "Tamanhos finais:"
  ls -lh "$file" "${file}.zst" "${file}.crom" | awk '{print $9 " → " $5}'

  # Verifica se usou passthrough
  grep -E "passthrough|abort|bypass|entropia|zero-overhead" crom_log_${file##*/}.txt || echo "Nenhuma mensagem de bypass encontrada"
done

echo -e "\n✅ Benchmark finalizado."
