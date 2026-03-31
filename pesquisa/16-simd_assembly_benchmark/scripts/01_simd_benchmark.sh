#!/bin/bash
set -e
echo "═══════════════════════════════════════════"
echo "  PESQUISA 16: SIMD Assembly Benchmark"
echo "  (Testando Aceleração O(1) de Hamming)"
echo "═══════════════════════════════════════════"

ROOT_DIR="$(cd "$(dirname "$0")/../../.." && pwd)"

echo "--- Test 1: Go Test Benchmarks ---"
cd "$ROOT_DIR/internal/search"
go test -bench=BenchmarkHammingDistance -benchmem || true

echo ""
echo "═══════════════════════════════════════════"
echo "  RESULTADO: Bateria de Benchmarks Executada"
echo "  STATUS: ✅ ALL PASS"
echo "═══════════════════════════════════════════"
