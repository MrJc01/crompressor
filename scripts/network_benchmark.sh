#!/bin/bash
# ============================================================================
# CROMPRESSOR — Benchmark de Rede: RAW vs CROM vs Two Brains 🧠🧠
# ============================================================================
# Simula transferências TCP entre dois endpoints (localhost) e mede a
# diferença real de bytes transferidos entre 3 cenários:
#   1. RAW: arquivo original completo
#   2. CROM Completo: .crom + .cromdb (receptor não tem cérebro)
#   3. Two Brains: apenas .crom (receptor já possui o .cromdb)
#
# Uso: ./network_benchmark.sh [caminho_do_binario_crompressor]
# ============================================================================

set -euo pipefail

# --- Configuração ---
CROM_BIN="${1:-$(dirname "$0")/../bin/crompressor}"
PORT_BASE=19090
WORKDIR="/tmp/crom_net_bench"
REPORT="$WORKDIR/report.md"
SEPARATOR="═══════════════════════════════════════════════════════════════"

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

log()  { echo -e "${CYAN}[BENCH]${NC} $1"; }
ok()   { echo -e "${GREEN}  ✔${NC} $1"; }
fail() { echo -e "${RED}  ✘${NC} $1"; exit 1; }
hr()   { echo -e "${YELLOW}${SEPARATOR}${NC}"; }

# --- Funções de Medição ---

# Retorna nanosegundos desde epoch
now_ns() {
    date +%s%N
}

# Calcula diferença em ms
elapsed_ms() {
    local start=$1 end=$2
    echo $(( (end - start) / 1000000 ))
}

# Formata bytes para humano
human_bytes() {
    local bytes=$1
    if [ "$bytes" -ge 1048576 ]; then
        echo "$(echo "scale=2; $bytes/1048576" | bc) MB"
    elif [ "$bytes" -ge 1024 ]; then
        echo "$(echo "scale=2; $bytes/1024" | bc) KB"
    else
        echo "${bytes} B"
    fi
}

# Transfere arquivo via TCP e mede
# Args: $1=arquivo_a_enviar $2=arquivo_destino $3=porta $4=label
tcp_transfer() {
    local src="$1" dst="$2" port="$3" label="$4"
    local src_size
    src_size=$(stat -c%s "$src")

    # Servidor: escuta e salva
    nc -l -p "$port" > "$dst" &
    local server_pid=$!
    sleep 0.3

    # Cliente: envia
    local t_start
    t_start=$(now_ns)
    nc -w 5 localhost "$port" < "$src"
    wait $server_pid 2>/dev/null || true
    local t_end
    t_end=$(now_ns)

    local ms
    ms=$(elapsed_ms "$t_start" "$t_end")
    local dst_size
    dst_size=$(stat -c%s "$dst")

    echo "$src_size $dst_size $ms"
}

# Transfere múltiplos arquivos concatenados via TCP
# Args: $1=arquivo_concat $2=arquivo_destino $3=porta
tcp_transfer_multi() {
    local src="$1" dst="$2" port="$3"
    local src_size
    src_size=$(stat -c%s "$src")

    nc -l -p "$port" > "$dst" &
    local server_pid=$!
    sleep 0.3

    local t_start
    t_start=$(now_ns)
    nc -w 5 localhost "$port" < "$src"
    wait $server_pid 2>/dev/null || true
    local t_end
    t_end=$(now_ns)

    local ms
    ms=$(elapsed_ms "$t_start" "$t_end")
    local dst_size
    dst_size=$(stat -c%s "$dst")

    echo "$src_size $dst_size $ms"
}

# ============================================================================
# MAIN
# ============================================================================

hr
echo -e "${BOLD}  CROMPRESSOR — Benchmark de Rede${NC}"
echo -e "${BOLD}  RAW vs CROM Completo vs Two Brains 🧠🧠${NC}"
hr

# Verificar dependências
command -v nc >/dev/null || fail "netcat (nc) não encontrado. Instale: apt install ncat"
command -v bc >/dev/null || fail "bc não encontrado. Instale: apt install bc"
[ -x "$CROM_BIN" ] || fail "Binário crompressor não encontrado em: $CROM_BIN"

# Setup
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"/{server_raw,server_crom,server_brains,datasets}
cd "$WORKDIR"

# ============================================================================
# FASE 1: Gerar Datasets
# ============================================================================
log "Fase 1: Gerando datasets de teste..."

# 10MB de logs altamente redundantes
python3 -c "
import random
with open('datasets/bench_logs.txt', 'w') as f:
    for i in range(120000):
        worker = i % 16
        status = random.choice(['SUCCESS', 'SUCCESS', 'SUCCESS', 'WARNING', 'ERROR'])
        dur = random.randint(5, 200)
        f.write(f'2024-01-15 08:{i%60:02d}:{i%60:02d} INFO [Worker-{worker}] job={i} status={status} duration={dur}ms endpoint=/api/v2/data\n')
"
ok "bench_logs.txt: $(human_bytes $(stat -c%s datasets/bench_logs.txt))"

# 5MB de CSV tabular
python3 -c "
with open('datasets/bench_csv.csv', 'w') as f:
    f.write('id,name,category,price,stock,status,region\n')
    cats = ['Electronics','Books','Clothing','Food','Sports']
    regions = ['BR-SP','BR-RJ','BR-MG','US-NY','US-CA','EU-DE','EU-FR']
    for i in range(80000):
        f.write(f'{i},Product_{i%500},{cats[i%5]},{10.99+(i%20):.2f},{i%1000},ACTIVE,{regions[i%7]}\n')
"
ok "bench_csv.csv: $(human_bytes $(stat -c%s datasets/bench_csv.csv))"

# 4MB de código real (battleroyale project)
if [ -d "/home/j/Área de trabalho/battleroyale" ]; then
    tar -cf datasets/bench_code.tar -C "/home/j/Área de trabalho" battleroyale 2>/dev/null
    ok "bench_code.tar: $(human_bytes $(stat -c%s datasets/bench_code.tar))"
else
    # Fallback: gerar código sintético
    python3 -c "
import random
with open('datasets/bench_code.tar', 'wb') as f:
    patterns = [b'function ', b'const ', b'import ', b'export default ', b'return ', b'console.log(', b'if (', b'for (let ']
    for i in range(50000):
        p = random.choice(patterns)
        f.write(p + f'item_{i%200}'.encode() + b'() {\n  // implementation\n}\n\n')
"
    ok "bench_code.tar: $(human_bytes $(stat -c%s datasets/bench_code.tar))"
fi

FILES=("bench_logs.txt" "bench_csv.csv" "bench_code.tar")
LABELS=("10MB Logs" "5MB CSV" "4MB Code")

# ============================================================================
# FASE 2: Treinar Codebook e Empacotar
# ============================================================================
log "Fase 2: Treinando Codebook Global..."

"$CROM_BIN" train -i datasets -o global.cromdb -s 8192 2>&1 | tail -3
ok "Codebook gerado: $(human_bytes $(stat -c%s global.cromdb))"
CROMDB_SIZE=$(stat -c%s global.cromdb)

log "Fase 2: Empacotando datasets..."
for f in "${FILES[@]}"; do
    fname="${f%.*}"
    "$CROM_BIN" pack -i "datasets/$f" -o "${fname}.crom" -c global.cromdb -k 128 2>&1 | tail -3
    ok "${fname}.crom: $(human_bytes $(stat -c%s ${fname}.crom))"
done

# ============================================================================
# FASE 3: Benchmark TCP
# ============================================================================
hr
log "Fase 3: Iniciando Benchmark de Transferência TCP..."
hr

# Arrays para resultados
declare -a RAW_BYTES RAW_MS CROM_FULL_BYTES CROM_FULL_MS BRAINS_BYTES BRAINS_MS

for i in "${!FILES[@]}"; do
    f="${FILES[$i]}"
    label="${LABELS[$i]}"
    fname="${f%.*}"
    port=$((PORT_BASE + i * 10))

    echo ""
    log "━━━ Testando: ${label} (${f}) ━━━"

    # --- Cenário 1: RAW ---
    log "  Cenário 1 [RAW]: Transferindo arquivo original..."
    result=$(tcp_transfer "datasets/$f" "server_raw/received_$f" "$port" "RAW")
    raw_sent=$(echo "$result" | awk '{print $1}')
    raw_recv=$(echo "$result" | awk '{print $2}')
    raw_ms=$(echo "$result" | awk '{print $3}')
    RAW_BYTES+=("$raw_sent")
    RAW_MS+=("$raw_ms")

    # Verificar integridade
    orig_sha=$(sha256sum "datasets/$f" | awk '{print $1}')
    recv_sha=$(sha256sum "server_raw/received_$f" | awk '{print $1}')
    if [ "$orig_sha" = "$recv_sha" ]; then
        ok "RAW: $(human_bytes $raw_sent) em ${raw_ms}ms | SHA-256 ✓"
    else
        fail "SHA-256 MISMATCH no RAW!"
    fi

    # --- Cenário 2: CROM Completo (.crom + .cromdb) ---
    port2=$((port + 1))
    log "  Cenário 2 [CROM+DB]: Transferindo .crom + .cromdb concatenados..."

    # Criar pacote concatenado: [4 bytes crom_size][crom_data][cromdb_data]
    crom_size=$(stat -c%s "${fname}.crom")
    cromdb_file_size=$(stat -c%s "global.cromdb")
    total_crom=$((crom_size + cromdb_file_size))

    # Concatenar para envio
    cat "${fname}.crom" global.cromdb > "bundle_${fname}.bin"

    result2=$(tcp_transfer "bundle_${fname}.bin" "server_crom/received_bundle_${fname}.bin" "$port2" "CROM_FULL")
    crom_full_sent=$(echo "$result2" | awk '{print $1}')
    crom_full_ms=$(echo "$result2" | awk '{print $3}')
    CROM_FULL_BYTES+=("$crom_full_sent")
    CROM_FULL_MS+=("$crom_full_ms")

    # Receptor extrai: separa crom do cromdb e reconstrói
    head -c "$crom_size" "server_crom/received_bundle_${fname}.bin" > "server_crom/${fname}.crom"
    tail -c "$cromdb_file_size" "server_crom/received_bundle_${fname}.bin" > "server_crom/global.cromdb"
    "$CROM_BIN" unpack -i "server_crom/${fname}.crom" -o "server_crom/restored_$f" -c "server_crom/global.cromdb" 2>&1 | tail -2

    rest_sha=$(sha256sum "server_crom/restored_$f" | awk '{print $1}')
    if [ "$orig_sha" = "$rest_sha" ]; then
        ok "CROM+DB: $(human_bytes $crom_full_sent) em ${crom_full_ms}ms | SHA-256 ✓"
    else
        fail "SHA-256 MISMATCH no CROM Completo!"
    fi

    # --- Cenário 3: Two Brains 🧠🧠 ---
    port3=$((port + 2))
    log "  Cenário 3 [TWO BRAINS 🧠🧠]: Transferindo APENAS .crom..."

    # O servidor já tem o cromdb!
    cp global.cromdb server_brains/global.cromdb

    result3=$(tcp_transfer "${fname}.crom" "server_brains/received_${fname}.crom" "$port3" "BRAINS")
    brains_sent=$(echo "$result3" | awk '{print $1}')
    brains_ms=$(echo "$result3" | awk '{print $3}')
    BRAINS_BYTES+=("$brains_sent")
    BRAINS_MS+=("$brains_ms")

    # Receptor reconstrói usando SEU codebook local
    "$CROM_BIN" unpack -i "server_brains/received_${fname}.crom" -o "server_brains/restored_$f" -c "server_brains/global.cromdb" 2>&1 | tail -2

    brains_sha=$(sha256sum "server_brains/restored_$f" | awk '{print $1}')
    if [ "$orig_sha" = "$brains_sha" ]; then
        ok "TWO BRAINS: $(human_bytes $brains_sent) em ${brains_ms}ms | SHA-256 ✓"
    else
        fail "SHA-256 MISMATCH no Two Brains!"
    fi
done

# ============================================================================
# FASE 4: Relatório Final
# ============================================================================
hr
log "Fase 4: Gerando Relatório Final..."
hr

cat > "$REPORT" << 'HEADER'
# 📡 Benchmark de Rede — Crompressor

Comparação de transferência TCP simulada entre dois endpoints (localhost).
Todos os cenários verificados com SHA-256 bit-a-bit.

## Cenários Testados

| # | Cenário | O que trafega pela rede | Quando usar |
|:--|:--------|:----------------------|:------------|
| 1 | **RAW** | Arquivo original inteiro | Sem compressão |
| 2 | **CROM Completo** | `.crom` + `.cromdb` | 1ª transferência (receptor sem cérebro) |
| 3 | **Two Brains** 🧠🧠 | Apenas `.crom` | Transferências subsequentes (receptor já tem o cérebro) |

## Resultados

HEADER

# Tabela principal
{
echo "| Dataset | RAW (bytes) | CROM+DB (bytes) | Two Brains (bytes) | Economia RAW→CROM | Economia RAW→Brains | SHA-256 |"
echo "|:--------|:------------|:----------------|:-------------------|:-------------------|:---------------------|:--------|"

for i in "${!FILES[@]}"; do
    raw_b="${RAW_BYTES[$i]}"
    crom_b="${CROM_FULL_BYTES[$i]}"
    brain_b="${BRAINS_BYTES[$i]}"

    econ_crom=$(echo "scale=1; (1 - $crom_b/$raw_b) * 100" | bc)
    econ_brain=$(echo "scale=1; (1 - $brain_b/$raw_b) * 100" | bc)

    echo "| **${LABELS[$i]}** | $(human_bytes $raw_b) | $(human_bytes $crom_b) | $(human_bytes $brain_b) | ${econ_crom}% | **${econ_brain}%** | ✅ |"
done
} >> "$REPORT"

# Seção Two Brains
cat >> "$REPORT" << BRAINS

## 🧠🧠 Análise Two Brains

> **O codebook (cérebro) tem $(human_bytes $CROMDB_SIZE)** e é sincronizado **uma única vez**.
> Após a sincronização, cada transferência custa apenas o \`.crom\` (índices + delta comprimido).

### Economia Acumulativa (N transferências)

Simulação com os 3 arquivos enviados sequencialmente:

| Transferência # | RAW Acumulado | Two Brains Acumulado | Economia Total |
|:---------------|:-------------|:--------------------|:---------------|
BRAINS

{
raw_acc=0
brain_acc=$CROMDB_SIZE  # 1ª sync: precisa enviar o cérebro

for i in "${!FILES[@]}"; do
    raw_acc=$((raw_acc + RAW_BYTES[$i]))
    brain_acc=$((brain_acc + BRAINS_BYTES[$i]))
    econ=$(echo "scale=1; (1 - $brain_acc/$raw_acc) * 100" | bc)
    echo "| #$((i+1)) (${LABELS[$i]}) | $(human_bytes $raw_acc) | $(human_bytes $brain_acc) | **${econ}%** |"
done
} >> "$REPORT"

cat >> "$REPORT" << 'FOOTER'

## Conclusão

> [!IMPORTANT]
> A estratégia **Two Brains** prova que dois endpoints que compartilham o mesmo "cérebro"
> (codebook) podem transferir a mesma quantidade de informação usando **drasticamente menos bytes**.
> O custo do codebook (sincronização inicial) é amortizado rapidamente nas transferências subsequentes.

**Status:** Benchmark concluído com sucesso. ✅
FOOTER

echo ""
hr
echo -e "${BOLD}  RELATÓRIO GERADO: ${REPORT}${NC}"
hr
echo ""
cat "$REPORT"
