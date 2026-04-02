#!/bin/bash
# Wrapper: executa o benchmark base (testes 01-05) a partir do teste 01
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$DIR/../scripts"
bash run_benchmarks.sh
