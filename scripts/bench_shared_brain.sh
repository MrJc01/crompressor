#!/usr/bin/env bash
# Mede os bytes que precisam ir pela rede quando os dois lados já têm o cérebro.
# Uso: scripts/bench_shared_brain.sh <binário crompressor> <diretório de trabalho>
# Roda com prioridade baixa e 2 threads para não travar a máquina.
set -u
CROM=$(realpath "$1"); W=$2; REPO=$(pwd)
mkdir -p "$W"; cd "$W" || exit 1
ulimit -v 6000000

sz() { stat -c%s "$1"; }
z() { nice -n 19 zstd -19 -T2 -q -c "$1" | wc -c; }
patch() { nice -n 19 zstd -19 -T2 --long=25 -q --patch-from="$1" -c "$2" 2>/dev/null | wc -c; }
train() { mkdir -p "tr_$1"; cp "$2" "tr_$1/"; nice -n 19 "$CROM" train -i "tr_$1" -o "$1.cromdb" -k 4096 -s 40000 --concurrency 2 >/dev/null 2>&1; }
row() { # rótulo, cérebro, base do patch, arquivo
  : > o.crom; : > o.out
  nice -n 19 "$CROM" pack -i "$4" -o o.crom -c "$2" --concurrency 2 >/dev/null 2>&1
  nice -n 19 "$CROM" unpack -i o.crom -o o.out -c "$2" >/dev/null 2>&1
  cmp -s "$4" o.out && r=ok || r=FALHOU
  : > m.crom
  nice -n 19 "$CROM" pack -i "$4" -o m.crom -c "$2" --concurrency 2 --max-compression >/dev/null 2>&1
  : > m.out
  nice -n 19 "$CROM" unpack -i m.crom -o m.out -c "$2" >/dev/null 2>&1
  cmp -s "$4" m.out || r="$r, max FALHOU"
  echo "| $1 | $(sz "$4") | $(sz "$2") | $(sz o.crom) | $(sz m.crom) | $r | $(z "$4") | $(patch "$3" "$4") |"
}

OLD=$(git -C "$REPO" rev-list origin/dev -- pesquisa | sed -n 4p)
git -C "$REPO" archive --format=tar -o "$W/v1.tar" "$OLD" -- pkg internal cmd docs pesquisa
git -C "$REPO" archive --format=tar -o "$W/v2.tar" origin/dev -- pkg internal cmd docs pesquisa
{ printf X; cat v1.tar; } > shift.tar
head -c 20000000 /dev/urandom > r1.bin
{ head -c 10000000 r1.bin; printf novo; tail -c 10000000 r1.bin; } > r2.bin
python3 - <<'P'
import random
def logs(s):
    random.seed(s)
    return "".join("2026-10-%02dT10:%02d:%02d INFO api[%d] GET /v1/items/%d 200 %dms user=u%d\n"%(s,random.randrange(60),random.randrange(60),random.randrange(900),random.randrange(5000),random.randrange(300),random.randrange(120)) for _ in range(100000))
open('log1.txt','w').write(logs(1)); open('log2.txt','w').write(logs(2))
P
find "$HOME/go/pkg/mod" -name '*.go' -size +2k -size -60k 2>/dev/null | sort | head -3000 > list.txt
sed -n '1~2p' list.txt | tar cf go_treino.tar -T - 2>/dev/null
sed -n '2~2p' list.txt | tar cf go_novo.tar -T - 2>/dev/null

train proj v1.tar; train rnd r1.bin; train log log1.txt; train go go_treino.tar
echo "| Caso | Original | Cérebro | \`.crom\` | \`--max-compression\` | Restaurou | zstd -19 | zstd --patch-from |"
echo "|---|---|---|---|---|---|---|---|"
row "Visto: projeto idêntico" proj.cromdb v1.tar v1.tar
row "Visto: projeto + 1 byte no início" proj.cromdb v1.tar shift.tar
row "Parecido: versão real mais nova do projeto" proj.cromdb v1.tar v2.tar
row "Visto: binário aleatório idêntico" rnd.cromdb r1.bin r1.bin
row "Parecido: binário aleatório + 4 bytes no meio" rnd.cromdb r1.bin r2.bin
row "Novo: logs de outro dia, mesmo formato" log.cromdb log1.txt log2.txt
row "Novo: código Go de outros arquivos" go.cromdb go_treino.tar go_novo.tar
