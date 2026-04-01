#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 108-pi_digit_extraction"
cd "$DIR"
go run pi_extraction.go
echo "✅ Pesquisa 108-pi_digit_extraction concluída."
