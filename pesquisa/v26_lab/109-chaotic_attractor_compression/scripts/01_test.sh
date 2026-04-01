#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 109-chaotic_attractor_compression"
cd "$DIR"
go run chaotic_attractor.go
echo "✅ Pesquisa 109-chaotic_attractor_compression concluída."
