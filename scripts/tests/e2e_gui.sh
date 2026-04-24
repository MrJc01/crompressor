#!/bin/bash
set -e
# ═══════════════════════════════════════════════
# Crompressor E2E Test Suite
# Testa todos os endpoints REST + integridade do frontend
# ═══════════════════════════════════════════════

PORT=9100
BASE="http://127.0.0.1:$PORT"
PASS=0
FAIL=0
TOTAL=0

log_test() {
    TOTAL=$((TOTAL + 1))
    local name="$1"
    local result="$2"
    local detail="$3"
    if [ "$result" = "OK" ]; then
        PASS=$((PASS + 1))
        echo "✅ [$TOTAL] $name"
    else
        FAIL=$((FAIL + 1))
        echo "❌ [$TOTAL] $name — $detail"
    fi
}

echo "╔═══════════════════════════════════════════════╗"
echo "║  CROMPRESSOR E2E TEST SUITE                   ║"
echo "╚═══════════════════════════════════════════════╝"
echo ""

# Inicia o servidor em background
echo "→ Iniciando crompressor_gui..."
./crompressor_gui &>/dev/null &
SERVER_PID=$!
sleep 2

# Verifica se o servidor subiu
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ FATAL: Servidor não iniciou!"
    exit 1
fi
echo "→ Servidor rodando (PID: $SERVER_PID)"
echo ""

# ─── TESTE 1: Health Check ───
echo "═══ API Tests ═══"
RESP=$(curl -s "$BASE/api/health")
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/health → retorna success:true" "OK"
else
    log_test "/api/health → retorna success:true" "FAIL" "$RESP"
fi

# ─── TESTE 2: Listar diretório home ───
RESP=$(curl -s "$BASE/api/list")
if echo "$RESP" | grep -q '"success":true'; then
    COUNT=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['count'])" 2>/dev/null || echo "?")
    log_test "/api/list → lista o diretório home ($COUNT itens)" "OK"
else
    log_test "/api/list → lista o diretório home" "FAIL" "$RESP"
fi

# ─── TESTE 3: Listar diretório específico ───
RESP=$(curl -s "$BASE/api/list?dir=/tmp")
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/list?dir=/tmp → lista /tmp" "OK"
else
    log_test "/api/list?dir=/tmp → lista /tmp" "FAIL" "$RESP"
fi

# ─── TESTE 4: Listar diretório inexistente (deve dar erro) ───
RESP=$(curl -s "$BASE/api/list?dir=/non_existent_path_xyz")
if echo "$RESP" | grep -q '"error"'; then
    log_test "/api/list dir inexistente → retorna erro" "OK"
else
    log_test "/api/list dir inexistente → retorna erro" "FAIL" "$RESP"
fi

# ─── TESTE 5: Info de arquivo real ───
RESP=$(curl -s "$BASE/api/info?path=/etc/hostname")
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/info?path=/etc/hostname → retorna metadados" "OK"
else
    log_test "/api/info?path=/etc/hostname → retorna metadados" "FAIL" "$RESP"
fi

# ─── TESTE 6: Info sem path (deve dar erro) ───
RESP=$(curl -s "$BASE/api/info")
if echo "$RESP" | grep -q '"error"'; then
    log_test "/api/info sem path → retorna erro" "OK"
else
    log_test "/api/info sem path → retorna erro" "FAIL" "$RESP"
fi

# ─── TESTE 7: Pack com JSON inválido ───
RESP=$(curl -s -X POST "$BASE/api/pack" -d "not json")
if echo "$RESP" | grep -q '"error"'; then
    log_test "/api/pack JSON inválido → retorna erro" "OK"
else
    log_test "/api/pack JSON inválido → retorna erro" "FAIL" "$RESP"
fi

# ─── TESTE 8: Pack sem campos obrigatórios ───
RESP=$(curl -s -X POST "$BASE/api/pack" -H "Content-Type: application/json" -d '{"input":""}')
if echo "$RESP" | grep -q '"error"'; then
    log_test "/api/pack campos vazios → retorna erro" "OK"
else
    log_test "/api/pack campos vazios → retorna erro" "FAIL" "$RESP"
fi

# ─── TESTE 9: Train endpoint aceita request ───
RESP=$(curl -s -X POST "$BASE/api/train" -H "Content-Type: application/json" -d '{"input":"/tmp","output":"/tmp/test.cromdb","size":256}')
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/train → aceita request e inicia background" "OK"
else
    log_test "/api/train → aceita request e inicia background" "FAIL" "$RESP"
fi

# ─── TESTE 10: Delete de arquivo inexistente (deve dar erro) ───
RESP=$(curl -s -X POST "$BASE/api/delete" -H "Content-Type: application/json" -d '{"path":"/tmp/nonexistent_crom_test_xyz"}')
if echo "$RESP" | grep -q '"error"'; then
    log_test "/api/delete inexistente → retorna erro" "OK"
else
    log_test "/api/delete inexistente → retorna erro" "FAIL" "$RESP"
fi

# -- Cryptography / Identity
# ─── TESTE 11: Gerar par de chaves ───
RESP=$(curl -s -X GET "$BASE/api/identity/generate")
if echo "$RESP" | grep -q '"public_key"'; then
    log_test "/api/identity/generate → retorna par de chaves" "OK"
else
    log_test "/api/identity/generate → retorna par de chaves" "FAIL" "$RESP"
fi

# -- P2P Swarm
# ─── TESTE 12: Iniciar peer P2P ───
RESP=$(curl -s -X POST "$BASE/api/swarm/start" -H "Content-Type: application/json" -d '{"port":4005,"data_dir":"/tmp/swarm"}')
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/swarm/start → inicia o peer P2P" "OK"
else
    log_test "/api/swarm/start → inicia o peer P2P" "FAIL" "$RESP"
fi

# ─── TESTE 13: Listar peers P2P (0 no início) ───
RESP=$(curl -s -X GET "$BASE/api/swarm/peers")
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/swarm/peers → lista peers p2p (0 no início)" "OK"
else
    log_test "/api/swarm/peers → lista peers p2p (0 no início)" "FAIL" "$RESP"
fi

# ─── TESTE 14: Parar peer P2P ───
RESP=$(curl -s -X POST "$BASE/api/swarm/stop")
if echo "$RESP" | grep -q '"success":true'; then
    log_test "/api/swarm/stop → para o peer P2P" "OK"
else
    log_test "/api/swarm/stop → para o peer P2P" "FAIL" "$RESP"
fi

# -- Verify SHA-256
# ─── TESTE 15: Validar hash de dois arquivos iguais ───
touch /tmp/crom_test_verify.txt
RESP=$(curl -s -X POST "$BASE/api/verify" -H "Content-Type: application/json" -d '{"original":"/tmp/crom_test_verify.txt","restored":"/tmp/crom_test_verify.txt"}')
if echo "$RESP" | grep -q '"match":true'; then
    log_test "/api/verify → valida hash de dois arquivos iguais" "OK"
else
    log_test "/api/verify → valida hash de dois arquivos iguais" "FAIL" "$RESP"
fi

# ─── TESTE 16: Frontend servido ───
echo ""
echo "═══ Frontend Tests ═══"
RESP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$RESP" = "200" ]; then
    log_test "GET / → retorna HTTP 200 (index.html)" "OK"
else
    log_test "GET / → retorna HTTP 200" "FAIL" "HTTP $RESP"
fi

# ─── TESTE 17: JS bundle existe ───
RESP=$(curl -s "$BASE/" | grep -c "assets/index")
if [ "$RESP" -ge 1 ]; then
    log_test "index.html referencia JS bundle" "OK"
else
    log_test "index.html referencia JS bundle" "FAIL" "nenhum script encontrado"
fi

# ─── TESTE 18: CSS bundle existe ───
RESP=$(curl -s "$BASE/" | grep -c ".css")
if [ "$RESP" -ge 1 ]; then
    log_test "index.html referencia CSS bundle" "OK"
else
    log_test "index.html referencia CSS bundle" "FAIL" "nenhum css encontrado"
fi

# ─── TESTE 14: WebSocket conecta ───
# Usa timeout de 2s via curl (suporta upgrade mas não sustenta)
WS_RESP=$(curl -s -o /dev/null -w "%{http_code}" -H "Upgrade: websocket" -H "Connection: Upgrade" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" -H "Sec-WebSocket-Version: 13" "$BASE/ws")
if [ "$WS_RESP" = "101" ]; then
    log_test "WebSocket /ws → upgrade 101" "OK"
else
    log_test "WebSocket /ws → upgrade 101" "FAIL" "HTTP $WS_RESP"
fi

# ═══ RESULTADO FINAL ═══
echo ""
echo "═══════════════════════════════════════════════"
echo "  RESULTADO: $PASS/$TOTAL testes passaram"
if [ $FAIL -gt 0 ]; then
    echo "  ⚠ $FAIL teste(s) falharam"
fi
echo "═══════════════════════════════════════════════"

# Cleanup
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
echo "→ Servidor encerrado."
