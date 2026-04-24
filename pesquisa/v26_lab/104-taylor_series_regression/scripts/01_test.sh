#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 104-taylor_series_regression"
cd "$DIR"
go run taylor_regression.go
echo "✅ Pesquisa 104-taylor_series_regression concluída."
