#!/bin/bash
# ==============================================================
# CROM MONITOR V3: Auditoria Autocontida com Logs
# ==============================================================

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Detecta todos os app_*
readarray -d '' APPS < <(find "$SCRIPT_DIR" -maxdepth 1 -type d -name "app_*" -print0 | sort -z)

clear
echo -e "${BLUE}══════════════════════════════════════════════════════${NC}"
echo -e "🚀 ${BOLD}${GREEN}CROM MONITOR V3 — AUDITORIA AUTOCONTIDA${NC}"
echo -e "${BLUE}══════════════════════════════════════════════════════${NC}"
echo -e "   ${#APPS[@]} laboratórios detectados"
echo ""

menu_principal() {
    echo -e "${BOLD}${YELLOW}MODO DE OPERAÇÃO:${NC}"
    echo "  1) Testar UM projeto (individual)"
    echo "  2) Ciclo completo (passa por todos, você decide cada)"
    echo "  3) Executar TODOS automaticamente (sem parar)"
    echo "  4) Ver logs de testes anteriores"
    echo "  5) Limpar montagens FUSE (emergência)"
    echo -e "  ${RED}q) Sair${NC}"
    echo -e "${BLUE}──────────────────────────────────────────────────────${NC}"

    read -p "Opção: " OPT
    case $OPT in
        1) modo_individual ;;
        2) modo_ciclo ;;
        3) modo_todos ;;
        4) ver_logs ;;
        5) limpar_tudo ;;
        q|Q) echo -e "${GREEN}Até logo!${NC}"; exit 0 ;;
        *) echo -e "${RED}Inválido.${NC}"; menu_principal ;;
    esac
}

# ─── MODO 1: INDIVIDUAL ────────────────────────────────
modo_individual() {
    echo -e "\n${BLUE}── SELECIONE UM PROJETO ──${NC}"
    local i=1
    for APP in "${APPS[@]}"; do
        printf "  %2d) %s\n" "$i" "$(basename "$APP")"
        i=$((i+1))
    done
    echo "   v) Voltar"

    read -p "Número: " NUM
    [[ "$NUM" == "v" ]] && { menu_principal; return; }

    if [[ "$NUM" =~ ^[0-9]+$ ]] && [ "$NUM" -ge 1 ] && [ "$NUM" -le "${#APPS[@]}" ]; then
        executar_lab "${APPS[$((NUM-1))]}"
    else
        echo -e "${RED}Número inválido.${NC}"
    fi
    modo_individual
}

# ─── MODO 2: CICLO (passo a passo) ─────────────────────
modo_ciclo() {
    echo -e "\n${BLUE}── CICLO DE AUDITORIA (passo a passo) ──${NC}"
    local total=${#APPS[@]}
    local atual=0

    for APP in "${APPS[@]}"; do
        atual=$((atual+1))
        local nome=$(basename "$APP")
        echo -e "\n${BOLD}${YELLOW}[$atual/$total] Próximo: $nome${NC}"
        echo -e "  [Enter/s] Executar  [n] Pular  [q] Parar"
        read -p "  > " RESP

        case $RESP in
            ""|s|S|1|y|Y) executar_lab "$APP" ;;
            q|Q) echo -e "${RED}Ciclo interrompido.${NC}"; break ;;
            *) echo -e "${YELLOW}⏭️  Pulado: $nome${NC}" ;;
        esac
    done
    echo -e "\n${GREEN}Ciclo finalizado.${NC}"
    menu_principal
}

# ─── MODO 3: TODOS AUTOMÁTICOS ─────────────────────────
modo_todos() {
    echo -e "\n${BLUE}── EXECUTANDO TODOS OS TESTES AUTOMATICAMENTE ──${NC}"
    local total=${#APPS[@]}
    local atual=0
    local passou=0
    local falhou=0

    for APP in "${APPS[@]}"; do
        atual=$((atual+1))
        local nome=$(basename "$APP")
        echo -e "\n${BOLD}[$atual/$total] $nome${NC}"
        executar_lab "$APP"

        if [ $? -eq 0 ]; then
            passou=$((passou+1))
        else
            falhou=$((falhou+1))
        fi
    done

    echo -e "\n${BLUE}══════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}📊 RELATÓRIO FINAL:${NC}"
    echo -e "   Total: $total | ${GREEN}Pass: $passou${NC} | ${RED}Fail: $falhou${NC}"
    echo -e "${BLUE}══════════════════════════════════════════════════════${NC}"
    read -p "Pressione [Enter] para voltar ao menu..."
    menu_principal
}

# ─── MODO 4: VER LOGS ──────────────────────────────────
ver_logs() {
    echo -e "\n${BLUE}── LOGS DISPONÍVEIS ──${NC}"
    local encontrou=0
    for APP in "${APPS[@]}"; do
        local nome=$(basename "$APP")
        local logdir="$APP/logs"
        if [ -d "$logdir" ] && [ "$(ls -A "$logdir" 2>/dev/null)" ]; then
            encontrou=1
            local ultimo=$(ls -t "$logdir"/*.log 2>/dev/null | head -1)
            local qtd=$(ls "$logdir"/*.log 2>/dev/null | wc -l)
            echo -e "  ${GREEN}●${NC} $nome ($qtd logs) — último: $(basename "$ultimo")"
        fi
    done

    if [ $encontrou -eq 0 ]; then
        echo -e "  ${YELLOW}Nenhum log encontrado. Execute testes primeiro.${NC}"
        read -p "Pressione [Enter]..."
        menu_principal
        return
    fi

    echo ""
    echo "  a) Ver log mais recente de um projeto"
    echo "  v) Voltar"
    read -p "  > " RESP

    if [[ "$RESP" == "a" ]]; then
        read -p "  Nome do projeto (ex: app_io_performance): " PROJ
        local logdir="$SCRIPT_DIR/$PROJ/logs"
        if [ -d "$logdir" ]; then
            local ultimo=$(ls -t "$logdir"/*.log 2>/dev/null | head -1)
            if [ -f "$ultimo" ]; then
                echo -e "\n${BLUE}── LOG: $ultimo ──${NC}"
                cat "$ultimo"
                echo ""
            else
                echo -e "${RED}Nenhum log nesse projeto.${NC}"
            fi
        else
            echo -e "${RED}Projeto não encontrado.${NC}"
        fi
        read -p "Pressione [Enter]..."
    fi
    menu_principal
}

# ─── EXECUTAR UM LABORATÓRIO ───────────────────────────
executar_lab() {
    local TARGET_DIR="$1"
    local APP_NAME=$(basename "$TARGET_DIR")

    (
        cd "$TARGET_DIR"
        if [ -f "./launch.sh" ]; then
            bash ./launch.sh
        else
            echo -e "${RED}❌ launch.sh não encontrado em $APP_NAME${NC}"
            return 1
        fi
    )
    local result=$?

    echo ""
    read -p "Pressione [Enter] para continuar..."
    return $result
}

# ─── LIMPEZA GERAL ─────────────────────────────────────
limpar_tudo() {
    echo -e "\n${RED}🧹 Desmontando todos os FUSE/Overlay...${NC}"
    for APP in "${APPS[@]}"; do
        fusermount -uz "$APP/merged" 2>/dev/null || true
        fusermount -uz "$APP/mnt" 2>/dev/null || true
    done
    echo -e "${GREEN}Limpo.${NC}"
    menu_principal
}

# ─── INICIALIZAÇÃO ─────────────────────────────────────
if [ ${#APPS[@]} -eq 0 ]; then
    echo -e "${RED}Erro: Nenhum app_* encontrado.${NC}"
    exit 1
fi

menu_principal
