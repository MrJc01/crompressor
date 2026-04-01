#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 105-mandelbrot_chunk_mapping"
cd "$DIR"
go run mandelbrot_mapping.go
echo "✅ Pesquisa 105-mandelbrot_chunk_mapping concluída."
