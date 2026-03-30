#!/bin/bash
# ==============================================================================
# 🚀 Crompressor Master Audit Script (V11)
# Executa todos os testes de pesquisa do 01 ao 07 sequencialmente
# ==============================================================================

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════════════════╗"
echo "║      Mestre de Auditoria CROM (Testes 01-07)         ║"
echo "╚══════════════════════════════════════════════════════╝"

echo "🔨 Construindo Crompressor na RAIZ..."
cd ../
make clean build > /dev/null
cd "$DIR"

# 1. 01 a 05: Run Benchmarks Base
echo "▶️ Iniciando Bateria 01 a 05 (Logs, CDC, VFS, P2P, TCO)..."
cd scripts
bash run_benchmarks.sh
cd ..

# 2. 06: Imagens Experimentais (Testes Completo e Relatório)
echo "▶️ Iniciando Bateria 06 (Treinamento e Inferência Completa)..."
cd 06-image_format_analysis/scripts
# Executa todos os scripts da pesquisa de imagem sequencialmente
echo "  [06.0] Gerando Datasets..."
bash 00_generate_datasets.sh
echo "  [06.1] Treinando Cérebros..."
bash 01_train_brains.sh
echo "  [06.2] Benchmark Same-Brain..."
bash 02_same_brain_test.sh
echo "  [06.3] Benchmark Cross-Brain..."
bash 03_cross_brain_test.sh
echo "  [06.4] Testes de Inferência Pós-Treino..."
bash 04_inference_test.sh
echo "  [06.5] Teste Cérebro Universal..."
bash 05_universal_brain_test.sh
echo "  [06.6] Consolidando Relatório 06..."
bash 06_generate_report.sh
cd ../../

# 3. 07: Benchmark Comparativo (Zstd, Gzip)
echo "▶️ Iniciando Bateria 07 (Comparativo Zstd/Gzip)..."
cd 07-benchmark_comparativo
bash run_benchmark.sh
cd ..

echo "✅ [ALL DONE] Auditoria de 01 a 07 concluída com sucesso."
