#!/bin/bash
# Wrapper para executar o benchmark comparativo da pesquisa 07
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"
bash run_benchmark.sh
