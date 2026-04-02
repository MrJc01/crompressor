#!/bin/bash
# ==============================================================================
# 🚀 Crompressor Master Audit Script (V26 — Hierárquico Corrigido)
# Executa todos os testes de pesquisa organizados por categoria
# ==============================================================================

# NÃO usar set -e: falha em um teste não deve abortar a pipeline inteira
# set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

TOTAL=0
OK=0
FAIL=0
SKIP=0

echo "╔══════════════════════════════════════════════════════╗"
echo "║   Mestre de Auditoria CROM (Hierárquico V26)         ║"
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
        TOTAL=$((TOTAL + 1))

        # Ignorar testes interativos/monitores que travam a automação
        if [[ "$test_name" == "102-native_llm_humble_pc" ]]; then
            echo "  ⏭ $test_name (monitor interativo, deve ser rodado manualmente)"
            SKIP=$((SKIP + 1))
            continue
        fi

        # Prioridade 1: scripts/01_test.sh (padrão)
        if [ -f "$test_dir/scripts/01_test.sh" ]; then
            echo "  ▶ $test_name"
            if (cd "$test_dir/scripts" && bash "01_test.sh"); then
                OK=$((OK + 1))
            else
                echo "    ⚠ Falhou (não-crítico)"
                FAIL=$((FAIL + 1))
            fi
        # Prioridade 2: qualquer *.sh em scripts/ (exceto utils.sh)
        elif [ -d "$test_dir/scripts" ]; then
            local found_scripts=0
            find "$test_dir/scripts" -maxdepth 1 -name "*.sh" -not -name "utils.sh" -print0 | sort -z | while IFS= read -r -d '' script; do
                found_scripts=1
                echo "  ▶ $test_name/$(basename "$script")"
                if (cd "$(dirname "$script")" && bash "$(basename "$script")"); then
                    OK=$((OK + 1))
                else
                    echo "    ⚠ Falhou (não-crítico)"
                    FAIL=$((FAIL + 1))
                fi
            done
            if [ "$found_scripts" -eq 0 ]; then
                echo "  ⏭ $test_name (scripts/ vazio)"
                SKIP=$((SKIP + 1))
            fi
        # Prioridade 3: run_benchmark.sh ou run_test.sh na raiz do diretório
        elif [ -f "$test_dir/run_benchmark.sh" ]; then
            echo "  ▶ $test_name/run_benchmark.sh"
            if (cd "$test_dir" && bash "run_benchmark.sh"); then
                OK=$((OK + 1))
            else
                echo "    ⚠ Falhou (não-crítico)"
                FAIL=$((FAIL + 1))
            fi
        elif [ -f "$test_dir/run_test.sh" ]; then
            echo "  ▶ $test_name/run_test.sh"
            if (cd "$test_dir" && bash "run_test.sh"); then
                OK=$((OK + 1))
            else
                echo "    ⚠ Falhou (não-crítico)"
                FAIL=$((FAIL + 1))
            fi
        else
            echo "  ⏭ $test_name (sem scripts)"
            SKIP=$((SKIP + 1))
        fi
    done
}

# Executar por categoria (01-05 agora têm wrappers em scripts/01_test.sh)
run_category "core_engine"  "🔧 Core Engine (01-07)"
run_category "rede_p2p"     "🌐 Rede & P2P"
run_category "seguranca"    "🔒 Segurança & Crypto"
run_category "performance"  "⚡ Performance & Hardware"
run_category "ia_llm"       "🧠 IA & LLM"
run_category "fronteira"    "🚀 Astrofísica & Fronteira"
run_category "v26_lab"      "🔬 V26 Fractal Lab (103-110)"

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  ✅ [ALL DONE] Auditoria Hierárquica V26 concluída!            ║"
echo "║  🌐 Para testes WEB (web_lab/), execute:                       ║"
echo "║     cd pesquisa && python3 -m http.server 8091                 ║"
echo "║     Abra: http://localhost:8091/web_audit_lab.html             ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
