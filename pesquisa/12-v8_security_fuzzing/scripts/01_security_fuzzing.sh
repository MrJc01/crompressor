#!/bin/bash
# ============================================================================
# PESQUISA 12: V8 Security Fuzzing (Zero-Trust Header Validation)
# ============================================================================
# Objetivo: Simular ataques cibernéticos contra o formato V8:
#   1. OOM Attack: Forjar MicroDictSize = 4GB e verificar rejeição instantânea
#   2. Truncated Payload: MicroDictSize válido mas dados truncados
#   3. Corrupt Magic: Cabeçalho V8 com magic bytes corrompidos
#   4. Go Unit Tests: Executar TestFormat_V8_OOM_Defense
# ============================================================================

set -euo pipefail

CROM="$(cd "$(dirname "$0")/../../.." && pwd)/bin/crompressor"
RESULTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/results"
LOG_FILE="$(cd "$(dirname "$0")/.." && pwd)/security_fuzzing.log"

mkdir -p "$RESULTS_DIR"
> "$LOG_FILE"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🛡️ PESQUISA 12: V8 SECURITY FUZZING"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

PASS=0
FAIL=0
TOTAL=4

# ===========================================================================
# TESTE 1: OOM Attack — MicroDictSize = 0xFFFFFFFF
# ===========================================================================
echo "═══ Teste 1: OOM Attack (4GB MicroDictSize Forjado) ═══"

FORGED="$RESULTS_DIR/forged_oom.crom"
# Construir um header V8 forjado com Python
python3 -c "
import struct
# CROM Magic (4) + Version 8 (2) + IsEncrypted (1) + IsPassthrough (1)
buf = b'CROM' + struct.pack('<H', 8) + b'\x00\x00'
# Salt (16)
buf += b'\x00' * 16
# OriginalHash (32)
buf += b'\x00' * 32
# OriginalSize (8)
buf += struct.pack('<Q', 1000000)
# ChunkCount (4)
buf += struct.pack('<I', 100)
# ChunkSize (4)
buf += struct.pack('<I', 128)
# CodebookHash (8)
buf += b'\x00' * 8
# MerkleRoot (32)
buf += b'\x00' * 32
# IsConvergentEncrypted (1)
buf += b'\x00'
# CodebookHashes (24)
buf += b'\x00' * 24
# MicroDictSize = 0xFFFFFFFF (4GB = FORGED)
buf += struct.pack('<I', 0xFFFFFFFF)
# No payload (truncated on purpose)
with open('$FORGED', 'wb') as f:
    f.write(buf)
" 2>/dev/null

# Tentar desempacotar o arquivo forjado (dummy codebook path is enough to trigger format parsing)
DUMMY_CB="$RESULTS_DIR/dummy.cromdb"
touch "$DUMMY_CB"
UNPACK_OUT=$("$CROM" unpack -i "$FORGED" -o "$RESULTS_DIR/oom_output.bin" -c "$DUMMY_CB" 2>&1) || true
UNPACK_EXIT=$?

if echo "$UNPACK_OUT" | grep -qi "OOM\|safety\|too small\|exceeds\|error\|invalid"; then
    echo "  ✅ Ataque OOM REJEITADO corretamente"
    echo "  📋 Saída: $(echo "$UNPACK_OUT" | head -1)"
    PASS=$((PASS + 1))
    echo "PASS|OOM_4GB|Rejected" >> "$LOG_FILE"
else
    echo "  ❌ Ataque OOM NÃO foi detectado!"
    FAIL=$((FAIL + 1))
    echo "FAIL|OOM_4GB|NotDetected" >> "$LOG_FILE"
fi
echo ""

# ===========================================================================
# TESTE 2: Truncated Payload Attack
# ===========================================================================
echo "═══ Teste 2: Truncated Payload (MicroDictSize=1024, Payload=0) ═══"

TRUNC="$RESULTS_DIR/forged_truncated.crom"
python3 -c "
import struct
buf = b'CROM' + struct.pack('<H', 8) + b'\x00\x00'
buf += b'\x00' * 16   # Salt
buf += b'\x00' * 32   # Hash
buf += struct.pack('<Q', 5000)   # OriginalSize
buf += struct.pack('<I', 40)     # ChunkCount
buf += struct.pack('<I', 128)    # ChunkSize
buf += b'\x00' * 8              # CodebookHash
buf += b'\x00' * 32             # MerkleRoot
buf += b'\x00'                  # IsConvergentEncrypted
buf += b'\x00' * 24             # CodebookHashes
buf += struct.pack('<I', 1024)  # MicroDictSize = 1024 (legit size, but no data!)
# Header is 141 bytes. No micro-dict payload follows (truncated)
with open('$TRUNC', 'wb') as f:
    f.write(buf)
" 2>/dev/null

TRUNC_OUT=$("$CROM" unpack -i "$TRUNC" -o "$RESULTS_DIR/trunc_output.bin" -c "$DUMMY_CB" 2>&1) || true

if echo "$TRUNC_OUT" | grep -qi "too small\|truncat\|error\|invalid\|short"; then
    echo "  ✅ Payload Truncado REJEITADO corretamente"
    PASS=$((PASS + 1))
    echo "PASS|TRUNCATED|Rejected" >> "$LOG_FILE"
else
    echo "  ❌ Payload Truncado NÃO foi detectado!"
    FAIL=$((FAIL + 1))
    echo "FAIL|TRUNCATED|NotDetected" >> "$LOG_FILE"
fi
echo ""

# ===========================================================================
# TESTE 3: Corrupt Magic Bytes
# ===========================================================================
echo "═══ Teste 3: Corrupt Magic Bytes (HACK ao invés de CROM) ═══"

CORRUPT="$RESULTS_DIR/forged_magic.crom"
python3 -c "
import struct
buf = b'HACK' + struct.pack('<H', 8) + b'\x00\x00'
buf += b'\x00' * 133  # fill rest of V8 header
with open('$CORRUPT', 'wb') as f:
    f.write(buf)
" 2>/dev/null

MAGIC_OUT=$("$CROM" unpack -i "$CORRUPT" -o "$RESULTS_DIR/magic_output.bin" -c "$DUMMY_CB" 2>&1) || true

if echo "$MAGIC_OUT" | grep -qi "magic\|invalid\|error"; then
    echo "  ✅ Magic Corrompido REJEITADO corretamente"
    PASS=$((PASS + 1))
    echo "PASS|CORRUPT_MAGIC|Rejected" >> "$LOG_FILE"
else
    echo "  ❌ Magic Corrompido NÃO foi detectado!"
    FAIL=$((FAIL + 1))
    echo "FAIL|CORRUPT_MAGIC|NotDetected" >> "$LOG_FILE"
fi
echo ""

# ===========================================================================
# TESTE 4: Go Unit Test Suite V8
# ===========================================================================
echo "═══ Teste 4: Go Test Suite (V8 OOM Defense + Roundtrip) ═══"

cd "$(dirname "$0")/../../.."
GO_OUT=$(go test ./pkg/format/... -v -run "V8" 2>&1)
GO_PASS=$(echo "$GO_OUT" | grep -c "^--- PASS" || true)
GO_FAIL=$(echo "$GO_OUT" | grep -c "^--- FAIL" || true)

if [ "$GO_FAIL" -eq 0 ] && [ "$GO_PASS" -ge 2 ]; then
    echo "  ✅ Go Tests: ${GO_PASS} PASS / ${GO_FAIL} FAIL"
    PASS=$((PASS + 1))
    echo "PASS|GO_UNIT_TESTS|${GO_PASS}_pass" >> "$LOG_FILE"
else
    echo "  ❌ Go Tests: ${GO_PASS} PASS / ${GO_FAIL} FAIL"
    FAIL=$((FAIL + 1))
    echo "FAIL|GO_UNIT_TESTS|${GO_FAIL}_fail" >> "$LOG_FILE"
fi
cd "$RESULTS_DIR/.."
echo ""

# ===========================================================================
# RESULTADO FINAL
# ===========================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 RESULTADO FINAL: V8 SECURITY FUZZING"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ${PASS}/${TOTAL} PASS | ${FAIL}/${TOTAL} FAIL"
echo ""

if [ "$FAIL" -eq 0 ]; then
    echo "  🛡️ ZERO-TRUST V8: 100% VALIDADO"
    echo "     Nenhum ataque OOM, Truncation ou Corruption penetrou o parser."
else
    echo "  ⚠️ ZERO-TRUST V8: ${FAIL} vulnerabilidades detectadas!"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Cleanup forged files
rm -f "$RESULTS_DIR"/forged_*.crom "$RESULTS_DIR"/*_output.bin
