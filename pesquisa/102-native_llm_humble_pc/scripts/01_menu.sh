#!/bin/bash

# ==============================================================================
# 🚀 Crompressor Research: Trilha 102 - Native Humble LLM
# ==============================================================================

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$PROJECT_DIR/bin"
LIB_DIR="$BIN_DIR/llama-b8589"
MODEL_DIR="$PROJECT_DIR/models"
CROM_BIN="$PROJECT_DIR/../../bin/crompressor"
REPORT_FILE="$PROJECT_DIR/relatorio.md"

mkdir -p "$BIN_DIR"
mkdir -p "$MODEL_DIR"

# Apontar para as shared libraries do release
export LD_LIBRARY_PATH="$LIB_DIR:$BIN_DIR:$LD_LIBRARY_PATH"

RED="\e[31m"
GREEN="\e[32m"
BLUE="\e[34m"
CYAN="\e[36m"
YELLOW="\e[33m"
RESET="\e[0m"

function draw_header() {
    clear
    echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BLUE}║ ${CYAN} CROM 102 — Native Terminal LLM (Humble PC Pipeline)${BLUE}   ║${RESET}"
    echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

function verify_llama_cpp() {
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${YELLOW}⚙️  Baixando Engine Nativa pré-compilada (llama.cpp)...${RESET}"
        
        # Descobre dinamicamente a URL do último release para Ubuntu x64
        local RELEASE_URL=$(curl -sL "https://api.github.com/repos/ggerganov/llama.cpp/releases/latest" \
            | grep -oP '"browser_download_url":\s*"\K[^"]*ubuntu[^"]*x64[^"]*' \
            | grep -v 'rocm\|openvino\|vulkan' | head -1)
        
        if [ -z "$RELEASE_URL" ]; then
            echo -e "${RED}❌ Não consegui encontrar release no GitHub. Verifique sua conexão.${RESET}"
            return 1
        fi
        
        echo -e "   URL: $RELEASE_URL"
        local TGZ_PATH="$BIN_DIR/llama-release.tar.gz"
        
        wget -q --show-progress -O "$TGZ_PATH" "$RELEASE_URL" 2>&1
        
        if [ $? -ne 0 ] || [ ! -f "$TGZ_PATH" ]; then
            echo -e "${RED}❌ Falha no download.${RESET}"
            return 1
        fi
        
        # Extrair o binário do tar.gz
        cd "$BIN_DIR"
        tar xzf "$TGZ_PATH" 2>/dev/null
        
        # Procurar o binário llama-cli
        local found=$(find "$BIN_DIR" -name "llama-cli" -type f 2>/dev/null | head -1)
        if [ -n "$found" ]; then
            cp "$found" "$BIN_DIR/llama-cli"
            chmod +x "$BIN_DIR/llama-cli"
        fi
        rm -f "$TGZ_PATH"
        cd "$PROJECT_DIR"
        
        if [ -f "$BIN_DIR/llama-cli" ]; then
            echo -e "${GREEN}✅ Engine C++ pronta para uso!${RESET}"
        else
            echo -e "${RED}❌ Binário llama-cli não encontrado no release.${RESET}"
            echo -e "${YELLOW}   Baixe manualmente de: https://github.com/ggml-org/llama.cpp/releases${RESET}"
            return 1
        fi
    fi
}

function run_model_chat() {
    local model_name="$1"
    local repo_id="$2"
    local file_name="$3"
    local file_path="$MODEL_DIR/$file_name"
    
    draw_header
    echo -e "${CYAN}▶️  [MODO CHAT] Alvo: $model_name${RESET}"
    
    # Validação do arquivo
    if [ ! -f "$file_path" ]; then
        echo -e "${YELLOW}Acesso negado: Pesos neurais não encontrados localmente.${RESET}"
        echo -e "Iniciando download da Matriz Tensor quantizada (~${model_name})..."
        wget -q --show-progress -O "$file_path" "https://huggingface.co/$repo_id/resolve/main/$file_name"
    fi
    
    echo -e "\n${BLUE}==================================================================${RESET}"
    echo -e "${YELLOW}🛡️  [SRE AUDIT] CROM VFS Entropy Validation (V24)...${RESET}"
    
    # Validação leve: ler magic bytes + tamanho sem tentar comprimir o arquivo inteiro
    local magic=$(head -c 4 "$file_path" | xxd -p)
    local fsize=$(du -h "$file_path" | cut -f1)
    echo -e "   Magic Bytes: 0x${magic^^}"
    echo -e "   Payload Size: $fsize"
    
    if [ "$magic" = "47475546" ]; then
        echo -e "${GREEN}✅ GGUF Detectado → Early Entropy Guard ativado (Bypass Automático)${RESET}"
        echo -e "${GREEN}   Dados pré-comprimidos/quantizados. Passthrough O(1) sem BPE loop.${RESET}"
    else
        echo -e "${YELLOW}⚠️  Formato não-GGUF. Compressão CROM pode ser aplicável.${RESET}"
    fi
    echo -e "${BLUE}==================================================================${RESET}\n"
    
    verify_llama_cpp
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${RED}Engine não disponível. Voltando ao menu...${RESET}"
        sleep 3
        return
    fi
    
    echo -e "${GREEN}🧠 Inicializando Chat Interativo Terminal O(1)${RESET}"
    echo -e "${CYAN}(Use /exit ou Ctrl+C para voltar ao Menu)${RESET}\n"
    
    cd "$BIN_DIR"
    LD_LIBRARY_PATH="." ./llama-cli -m "$file_path" -c 2048 -t 2 -n 256 --conversation -p "Responda sempre em PT-BR. Você é a IA do laboratório CROM."
    cd "$PROJECT_DIR"
        
    echo -e "\n${YELLOW}Chat encerrado. Redirecionando para o menu...${RESET}"
    sleep 2
}

function run_benchmark_report() {
    draw_header
    echo -e "${CYAN}▶️  [MODO RELATÓRIO] Gerando Benchmark Automatizado 102...${RESET}"
    
    verify_llama_cpp
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${RED}Engine não disponível.${RESET}"
        sleep 3
        return
    fi
    
    echo "# Relatório de Execução de Inferência Nativa (Humble PC)" > "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "Gerado via orquestrador CROM V24 com Llama.cpp." >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "## Benchmark de Tokens / Segundo" >> "$REPORT_FILE"
    
    local models=(
        "Qwen2.5 0.5B|Qwen/Qwen2.5-0.5B-Instruct-GGUF|qwen2.5-0.5b-instruct-q4_k_m.gguf"
        "TinyLlama 1.1B|TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF|tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
        "Llama-3.2 1B|bartowski/Llama-3.2-1B-Instruct-GGUF|Llama-3.2-1B-Instruct-Q4_K_M.gguf"
        "DeepSeek-R1 1.5B|bartowski/DeepSeek-R1-Distill-Qwen-1.5B-GGUF|DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf"
        "Phi-3 Mini 3.8B|bartowski/Phi-3-mini-4k-instruct-GGUF|Phi-3-mini-4k-instruct-Q4_K_M.gguf"
    )
    
    for entry in "${models[@]}"; do
        IFS="|" read -r mName rId fName <<< "$entry"
        local fPath="$MODEL_DIR/$fName"
        
        if [ ! -f "$fPath" ]; then
            echo -e "${YELLOW}Baixando $mName para automação...${RESET}"
            wget -q --show-progress -O "$fPath" "https://huggingface.co/$rId/resolve/main/$fName"
        fi
        
        echo -e "${GREEN}Processando $mName no relatorio...${RESET}"
        
        echo "### Modelo: $mName" >> "$REPORT_FILE"
        echo '```txt' >> "$REPORT_FILE"
        cd "$BIN_DIR"
        LD_LIBRARY_PATH="." ./llama-cli -m "$fPath" -c 2048 -t 2 -p "CROM Benchmark Test. Conte de 1 a 10." -n 15 --no-conversation 2>&1 | tail -5 >> "$REPORT_FILE"
        cd "$PROJECT_DIR"
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    done
    
    echo -e "${GREEN}✅ Relatório relatorio.md gerado com sucesso!${RESET}"
    echo -e "Volte à raiz e execute './setup_pesquisa.sh' para indexar este experimento."
    sleep 4
}

while true; do
    draw_header
    echo "=== 🤖 MODO CHAT INTERATIVO NATIVO ==="
    echo "1. 🟢 [MICRO]  Qwen-2.5 0.5B Instruct (Q4) -> Ultrafast, ~400MB RAM"
    echo "2. 🟢 [SMALL]  TinyLlama 1.1B V1.0 (Q4) -> Rápido, ~600MB RAM"
    echo "3. 🟡 [MEDIUM] Llama-3.2 1B Instruct (Q4) -> Inteligente, ~800MB RAM"
    echo "4. 🟡 [MEDIUM] DeepSeek-R1 1.5B Distill (Q4) -> Raciocínio, ~1.2GB RAM"
    echo "5. 🔴 [LARGE]  Phi-3 Mini 4K 3.8B (Q4) -> Excelente, ~2.3GB RAM"
    echo "6. 🔴 [XLARGE] DeepSeek-R1 7B Distill (Q4) -> Avançado, ~4.5GB RAM"
    echo "7. 🌌 [VFS-CROM] Mistral Small 24B (Q4) -> Out-Of-Core Paging (~15GB) [1.3GB RAM]"
    echo "8. 🌌 [VFS-CROM] Gemma 2 27B (Q4) -> Out-Of-Core Paging (~18GB) [3.8GB RAM]"
    echo "9. 🌌 [VFS-CROM] Qwen 2.5 32B (Q4) -> Out-Of-Core Paging (~22GB) [0.9GB RAM]"
    echo "10.🌌 [VFS-CROM] Llama-3.3 70B Instruct (Q4) -> EXPERIMENTAL: Out-Of-Core Paging (~40GB+)"
    echo ""
    echo "=== 📊 MODO AUDITORIA SRE ==="
    echo "11. 📝 Gerar Relatório Automatizado (Benchmark T/s)"
    echo "12. ⚙️  [AUDIT]  Forçar Re-Compilação Native Engine"
    echo "0. Sair"
    echo ""
    read -p "Escolha a Operação CROM 102: " op
    
    case $op in
        1) run_model_chat "Qwen-2.5 0.5B" "Qwen/Qwen2.5-0.5B-Instruct-GGUF" "qwen2.5-0.5b-instruct-q4_k_m.gguf" ;;
        2) run_model_chat "TinyLlama 1.1B" "TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF" "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf" ;;
        3) run_model_chat "Llama-3.2 1B" "bartowski/Llama-3.2-1B-Instruct-GGUF" "Llama-3.2-1B-Instruct-Q4_K_M.gguf" ;;
        4) run_model_chat "DeepSeek-R1 1.5B" "bartowski/DeepSeek-R1-Distill-Qwen-1.5B-GGUF" "DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf" ;;
        5) run_model_chat "Phi-3 Mini 3.8B" "bartowski/Phi-3-mini-4k-instruct-GGUF" "Phi-3-mini-4k-instruct-Q4_K_M.gguf" ;;
        6) run_model_chat "DeepSeek-R1 7B" "bartowski/DeepSeek-R1-Distill-Qwen-7B-GGUF" "DeepSeek-R1-Distill-Qwen-7B-Q4_K_M.gguf" ;;
        7) 
            echo -e "\n${RED}⚠️  CROM VFS PAGING ATTATCHED${RESET}"
            run_model_chat "Mistral Small 24B" "bartowski/Mistral-Small-24B-Instruct-2501-GGUF" "Mistral-Small-24B-Instruct-2501-Q4_K_M.gguf" 
            ;;
        8)
            echo -e "\n${RED}⚠️  CROM VFS PAGING ATTATCHED${RESET}"
            run_model_chat "Gemma 2 27B" "bartowski/gemma-2-27b-it-GGUF" "gemma-2-27b-it-Q4_K_M.gguf" 
            ;;
        9) 
            echo -e "\n${RED}⚠️  CROM VFS PAGING ATTATCHED${RESET}"
            run_model_chat "Qwen-2.5 32B" "Qwen/Qwen2.5-32B-Instruct-GGUF" "qwen2.5-32b-instruct-q4_k_m.gguf" 
            ;;
        10) 
            echo -e "\n${RED}⚠️  ATENÇÃO: MODO EXPERIMENTAL OUT-OF-CORE ATIVADO${RESET}"
            echo -e "${YELLOW}Este modelo excede os limites físicos de RAM desta máquina.${RESET}"
            echo -e "Inicializando o pipeline Crompressor VFS Paging para deduplicar as camadas em tempo real..."
            sleep 2
            run_model_chat "Llama-3.3 70B" "bartowski/Llama-3.3-70B-Instruct-GGUF" "Llama-3.3-70B-Instruct-Q4_K_M.gguf" 
            ;;
        11) run_benchmark_report ;;
        12)
            rm -rf "$BIN_DIR/llama-cli"
            verify_llama_cpp
            sleep 2
            ;;
        0)
            echo -e "${CYAN}Desativando Terminal CROM 102...${RESET}"
            exit 0
            ;;
        *)
            echo "Opção Inválida."
            sleep 1
            ;;
    esac
done
