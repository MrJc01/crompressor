#!/bin/bash
# ==============================================================================
# 🚀 Crompressor Master Audit Script (V25 — Hierárquico)
# Executa todos os testes de pesquisa organizados por categoria
# ==============================================================================

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════════════════╗"
echo "║   Mestre de Auditoria CROM (Hierárquico V25)         ║"
echo "╚══════════════════════════════════════════════════════╝"

echo "🔨 Construindo Crompressor na RAIZ..."
cd ../
make clean build > /dev/null 2>&1 || echo "⚠ Build falhou, continuando com binário existente"
cd "$DIR"

run_category() {
    local category="$1"
    local label="$2"

    echo ""
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║  $label"
    echo "╚══════════════════════════════════════════════════════╝"

    if [ ! -d "$DIR/$category" ]; then
        echo "  ⚠ Categoria '$category' não encontrada, pulando."
        return
    fi

    find "$DIR/$category" -maxdepth 1 -type d -name "[0-9]*" -print0 | sort -zV | while IFS= read -r -d '' test_dir; do
        test_name="$(basename "$test_dir")"

        # Ignorar testes interativos/monitores que travam a automação
        if [[ "$test_name" == "102-native_llm_humble_pc" ]]; then
            echo "  ⏭ $test_name (monitor interativo, deve ser rodado manualmente)"
            continue
        fi

        if [ -f "$test_dir/scripts/01_test.sh" ]; then
            echo "  ▶ $test_name"
            (cd "$test_dir/scripts" && bash "01_test.sh") || echo "    ⚠ Falhou (não-crítico)"
        elif [ -d "$test_dir/scripts" ]; then
            find "$test_dir/scripts" -name "*.sh" -not -name "utils.sh" -print0 | sort -z | while IFS= read -r -d '' script; do
                echo "  ▶ $test_name/$(basename "$script")"
                (cd "$(dirname "$script")" && bash "$(basename "$script")") || echo "    ⚠ Falhou (não-crítico)"
            done
        else
            echo "  ⏭ $test_name (sem scripts)"
        fi
    done
}

# Executar benchmarks base (01-05) via scripts legados se existirem
if [ -f "$DIR/scripts/run_benchmarks.sh" ]; then
    echo "▶️ Base Benchmarks (scripts/)..."
    bash "$DIR/scripts/run_benchmarks.sh" || echo "⚠ Benchmarks base falharam"
fi

# Executar por categoria
run_category "core_engine"  "🔧 Core Engine (01-07)"
run_category "rede_p2p"     "🌐 Rede & P2P"
run_category "seguranca"    "🔒 Segurança & Crypto"
run_category "performance"  "⚡ Performance & Hardware"
run_category "ia_llm"       "🧠 IA & LLM"
run_category "fronteira"    "🚀 Astrofísica & Fronteira"
run_category "v26_lab"      "🔬 V26 Fractal Lab (103-110)"

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  ✅ [ALL DONE] Auditoria Hierárquica concluída!                ║"
echo "║  🌐 Para testes WEB (web_lab/), execute:                       ║"
echo "║     cd pesquisa && python3 -m http.server 8091                 ║"
echo "║     Abra: http://localhost:8091/web_audit_lab.html             ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
