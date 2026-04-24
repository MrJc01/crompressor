#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 110-entropy_adaptive_hybrid"
cd "$DIR"
go run adaptive_hybrid.go
echo "✅ Pesquisa 110-entropy_adaptive_hybrid concluída."
