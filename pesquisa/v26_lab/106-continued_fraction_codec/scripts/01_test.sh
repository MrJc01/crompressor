#!/bin/bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "▶ Pesquisa 106-continued_fraction_codec"
cd "$DIR"
go run cf_codec.go
echo "✅ Pesquisa 106-continued_fraction_codec concluída."
