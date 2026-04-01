#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 107-genetic_equation_search"
cd "$DIR"
go run genetic_search.go
echo "✅ Pesquisa 107-genetic_equation_search concluída."
