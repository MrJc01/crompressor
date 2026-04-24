#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 103: Motor Fractal V26 Multiestratégia"
cd "$DIR"
go run v26_fractal.go
echo "✅ Pesquisa 103 concluída."
