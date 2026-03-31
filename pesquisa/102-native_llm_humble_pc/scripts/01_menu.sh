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

export LD_LIBRARY_PATH="$LIB_DIR:$BIN_DIR:$LD_LIBRARY_PATH"

RED="\e[31m"
GREEN="\e[32m"
BLUE="\e[34m"
CYAN="\e[36m"
YELLOW="\e[33m"
RESET="\e[0m"

function get_available_ram_mb() {
    awk '/MemAvailable/ {printf "%d", $2/1024}' /proc/meminfo
}

function draw_header() {
    clear
    echo -e "${BLUE}╔═════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BLUE}║ ${CYAN} CROM 102 — Native Terminal LLM (Humble PC Pipeline)${BLUE}   ║${RESET}"
    echo -e "${BLUE}╚═════════════════════════════════════════════════════════╝${RESET}"
    local ram_mb=$(get_available_ram_mb)
    echo -e "   ${YELLOW}RAM Disponível: ${ram_mb} MB | CPU: $(nproc) threads${RESET}"
    echo ""
}

function verify_llama_cpp() {
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${YELLOW}⚙️  Baixando Engine Nativa pré-compilada (llama.cpp)...${RESET}"
        local RELEASE_URL=$(curl -sL "https://api.github.com/repos/ggerganov/llama.cpp/releases/latest" \
            | grep -oP '"browser_download_url":\s*"\K[^"]*ubuntu[^"]*x64[^"]*' \
            | grep -v 'rocm\|openvino\|vulkan' | head -1)
        
        if [ -z "$RELEASE_URL" ]; then
            echo -e "${RED}❌ Não consegui encontrar release no GitHub.${RESET}"
            return 1
        fi
        
        echo -e "   URL: $RELEASE_URL"
        local TGZ_PATH="$BIN_DIR/llama-release.tar.gz"
        wget -q --show-progress -O "$TGZ_PATH" "$RELEASE_URL" 2>&1
        
        if [ $? -ne 0 ] || [ ! -f "$TGZ_PATH" ]; then
            echo -e "${RED}❌ Falha no download.${RESET}"
            return 1
        fi
        
        cd "$BIN_DIR"
        tar xzf "$TGZ_PATH" 2>/dev/null
        local found=$(find "$BIN_DIR" -name "llama-cli" -type f 2>/dev/null | head -1)
        if [ -n "$found" ]; then
            cp "$found" "$BIN_DIR/llama-cli"
            chmod +x "$BIN_DIR/llama-cli"
        fi
        rm -f "$TGZ_PATH"
        cd "$PROJECT_DIR"
        
        if [ -f "$BIN_DIR/llama-cli" ]; then
            echo -e "${GREEN}✅ Engine C++ pronta!${RESET}"
        else
            echo -e "${RED}❌ Binário llama-cli não encontrado.${RESET}"
            return 1
        fi
    fi
}

# ==============================================================================
# Preparar o Brain CROM e compilar modelo para .crom
# ==============================================================================
function prepare_crom_brain() {
    local cb_path="$MODEL_DIR/global_llm_brain.cromdb"
    if [ ! -f "$cb_path" ]; then
        echo -e "${YELLOW}⚙️  Gerando Cérebro CROM provisório...${RESET}" >&2
        mkdir -p "/tmp/crom_dummy"
        head -c 4096 /dev/urandom > "/tmp/crom_dummy/seed.bin"
        "$CROM_BIN" train -i "/tmp/crom_dummy" -o "$cb_path" -s 1024 >&2
    fi
    echo "$cb_path"
}

function compile_to_crom() {
    local file_path="$1"
    local crom_path="${file_path}.crom"
    local cb_path=$(prepare_crom_brain)
    
    if [ ! -f "$crom_path" ]; then
        echo -e "${YELLOW}⚙️  Compilando para formato CROM (Passthrough Paging)...${RESET}" >&2
        "$CROM_BIN" pack -i "$file_path" -o "$crom_path" -c "$cb_path" >&2
        echo -e "${GREEN}✔ Compilação CROM finalizada.${RESET}" >&2
    else
        echo -e "${GREEN}✔ Arquivo .crom já existe (cache hit).${RESET}" >&2
    fi
    echo "$crom_path"
}

# ==============================================================================
# Montar VFS FUSE real do Crompressor
# ==============================================================================
function mount_crom_vfs() {
    local crom_path="$1"
    local file_name="$2"
    local cb_path="$MODEL_DIR/global_llm_brain.cromdb"
    
    mkdir -p /tmp/crom_llm
    fusermount -u /tmp/crom_llm 2>/dev/null
    sleep 1
    
    echo -e "${CYAN}🔌 Montando VFS FUSE real do Crompressor...${RESET}" >&2
    "$CROM_BIN" mount -i "$crom_path" -m /tmp/crom_llm -c "$cb_path" > /tmp/crom_vfs_daemon.log 2>&1 &
    VFS_PID=$!
    sleep 3
    
    # Descobrir o nome do arquivo exposto pelo FUSE
    local run_path="/tmp/crom_llm/$file_name"
    if [ ! -f "$run_path" ]; then
        run_path=$(find /tmp/crom_llm -type f 2>/dev/null | head -n 1)
    fi
    
    if [ -z "$run_path" ] || [ ! -f "$run_path" ]; then
        echo -e "${RED}❌ Falha na montagem VFS FUSE. Log:${RESET}" >&2
        cat /tmp/crom_vfs_daemon.log 2>/dev/null | tail -5 >&2
        kill -9 "$VFS_PID" 2>/dev/null
        VFS_PID=""
        return 1
    fi
    
    echo -e "${GREEN}✔ VFS Ativo (PID: $VFS_PID) → $run_path${RESET}" >&2
    echo "$run_path"
    return 0
}

function cleanup_vfs() {
    if [ -n "$VFS_PID" ]; then
        echo -e "${YELLOW}Desmontando VFS CROM...${RESET}"
        kill -9 "$VFS_PID" 2>/dev/null
        fusermount -u /tmp/crom_llm 2>/dev/null
        VFS_PID=""
    fi
}

# ==============================================================================
# Chat Principal
# ==============================================================================
function run_model_chat() {
    local model_name="$1"
    local repo_id="$2"
    local file_name="$3"
    local use_vfs="$4"           # "true" para usar pipeline CROM VFS
    local ctx_size="${5:-2048}"  # Context size
    local min_size_mb="${6:-10}" # Tamanho mínimo esperado (integridade)
    
    local file_path="$MODEL_DIR/$file_name"
    local run_path="$file_path"
    VFS_PID=""
    
    draw_header
    echo -e "${CYAN}▶️  [MODO CHAT] Alvo: $model_name${RESET}"
    
    # Download do modelo
    if [ ! -f "$file_path" ]; then
        echo -e "${YELLOW}Baixando $model_name...${RESET}"
        wget --show-progress -O "$file_path" "https://huggingface.co/$repo_id/resolve/main/$file_name"
    fi
    
    # Validar integridade
    if [ -f "$file_path" ]; then
        local actual_mb=$(du -m "$file_path" | cut -f1)
        if [ "$actual_mb" -lt "$min_size_mb" ]; then
            echo -e "${RED}⚠️  Arquivo corrompido (${actual_mb}MB < ${min_size_mb}MB). Re-baixando...${RESET}"
            rm -f "$file_path" "${file_path}.crom"
            wget --show-progress -O "$file_path" "https://huggingface.co/$repo_id/resolve/main/$file_name"
        fi
    fi
    
    if [ ! -f "$file_path" ]; then
        echo -e "${RED}❌ Arquivo não encontrado.${RESET}"
        sleep 3
        return
    fi
    
    # ========================================================================
    # Pipeline CROM VFS (para modelos marcados com use_vfs=true)
    # ========================================================================
    if [ "$use_vfs" == "true" ]; then
        echo -e "${YELLOW}🚀 PIPELINE CROM VFS ATIVADA${RESET}"
        
        # 1. Compilar para .crom
        local crom_path=$(compile_to_crom "$file_path")
        
        # 2. Montar VFS FUSE real
        local vfs_result=$(mount_crom_vfs "$crom_path" "$file_name")
        if [ $? -eq 0 ] && [ -n "$vfs_result" ]; then
            run_path="$vfs_result"
        else
            echo -e "${YELLOW}Fallback: usando arquivo direto sem VFS.${RESET}"
            run_path="$file_path"
        fi
    fi
    
    # Validação GGUF
    echo -e "\n${BLUE}==================================================================${RESET}"
    echo -e "${YELLOW}🛡️  [SRE AUDIT] Validação GGUF...${RESET}"
    
    local magic=$(head -c 4 "$run_path" 2>/dev/null | xxd -p)
    local fsize=$(ls -lh "$run_path" 2>/dev/null | awk '{print $5}')
    echo -e "   Magic: 0x${magic^^} | Size: $fsize"
    if [ "$use_vfs" == "true" ]; then
        echo -e "   ${CYAN}Fonte: CROM VFS FUSE (Paging O(1))${RESET}"
    else
        echo -e "   ${CYAN}Fonte: Disco Físico Direto${RESET}"
    fi
    
    if [ "$magic" = "47475546" ]; then
        echo -e "${GREEN}✅ GGUF Válido${RESET}"
    else
        echo -e "${RED}❌ Arquivo inválido! O Magic Check falhou.${RESET}"
        echo -e "${YELLOW}--- Debug Info ---${RESET}"
        echo "Run path tested: $run_path"
        echo "ls -lah /tmp/crom_llm:"
        ls -lah /tmp/crom_llm 2>/dev/null
        echo "head test direct:"
        head -c 4 "$run_path" 2>/dev/null | xxd -p
        echo "cat /tmp/crom_vfs_daemon.log (tail 10):"
        cat /tmp/crom_vfs_daemon.log 2>/dev/null | tail -10
        echo -e "${YELLOW}------------------${RESET}"
        read -p "Pressione Enter para voltar ao menu..."
        cleanup_vfs
        return
    fi
    echo -e "${BLUE}==================================================================${RESET}\n"
    
    verify_llama_cpp
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${RED}Engine não disponível.${RESET}"
        cleanup_vfs
        sleep 3
        return
    fi
    
    echo -e "${GREEN}🧠 Inicializando Chat Interativo${RESET}"
    echo -e "${CYAN}(Use /exit ou Ctrl+C para voltar ao Menu)${RESET}\n"
    
    cd "$BIN_DIR"
    
    # Threads = núcleos físicos (evita throttle térmico em HyperThreading)
    local phys_cores=$(lscpu -p=CORE 2>/dev/null | grep -v '#' | sort -u | wc -l)
    local threads=${phys_cores:-2}
    
    # OOM Score alto = kernel mata llama-cli primeiro, protege o desktop
    bash -c "echo 800 > /proc/\$\$/oom_score_adj 2>/dev/null; exec env LD_LIBRARY_PATH=\".\" ./llama-cli -m \"$run_path\" -c $ctx_size -t $threads -b 256 -n 256 --conversation -p \"Responda sempre em PT-BR. Você é a IA do laboratório CROM.\""
    local exit_code=$?
    
    if [ $exit_code -eq 137 ] || [ $exit_code -eq 139 ]; then
        echo -e "\n\e[31m🚨 [SRE] Processo eliminado (OOM/Segfault). RAM insuficiente.\e[0m"
    fi
    
    cd "$PROJECT_DIR"
    cleanup_vfs
    
    echo -e "\n${YELLOW}Chat encerrado. Redirecionando...${RESET}"
    sleep 2
}

function run_benchmark_report() {
    draw_header
    echo -e "${CYAN}▶️  Gerando Benchmark Automatizado...${RESET}"
    
    verify_llama_cpp
    if [ ! -f "$BIN_DIR/llama-cli" ]; then
        echo -e "${RED}Engine não disponível.${RESET}"
        sleep 3
        return
    fi
    
    echo "# Relatório de Inferência Nativa (Humble PC)" > "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "## Benchmark T/s" >> "$REPORT_FILE"
    
    local models=(
        "Qwen2.5 0.5B|Qwen/Qwen2.5-0.5B-Instruct-GGUF|qwen2.5-0.5b-instruct-q4_k_m.gguf"
        "TinyLlama 1.1B|TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF|tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
        "Llama-3.2 1B|bartowski/Llama-3.2-1B-Instruct-GGUF|Llama-3.2-1B-Instruct-Q4_K_M.gguf"
    )
    
    for entry in "${models[@]}"; do
        IFS="|" read -r mName rId fName <<< "$entry"
        local fPath="$MODEL_DIR/$fName"
        
        if [ ! -f "$fPath" ]; then
            echo -e "${YELLOW}Baixando $mName...${RESET}"
            wget -q --show-progress -O "$fPath" "https://huggingface.co/$rId/resolve/main/$fName"
        fi
        
        echo -e "${GREEN}Benchmarking $mName...${RESET}"
        echo "### $mName" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        cd "$BIN_DIR"
        LD_LIBRARY_PATH="." ./llama-cli -m "$fPath" -c 512 -t 2 -p "Conte de 1 a 10." -n 15 --no-conversation 2>&1 | tail -5 >> "$REPORT_FILE"
        cd "$PROJECT_DIR"
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    done
    
    echo -e "${GREEN}✅ Relatório gerado!${RESET}"
    sleep 4
}

function clean_corrupted() {
    draw_header
    echo -e "${CYAN}🔍 Verificando integridade...${RESET}\n"
    
    for f in "$MODEL_DIR"/*.gguf; do
        [ -f "$f" ] || continue
        local fname=$(basename "$f")
        local fsize_mb=$(du -m "$f" | cut -f1)
        local magic=$(head -c 4 "$f" | xxd -p)
        
        if [ "$magic" != "47475546" ]; then
            echo -e "  ${RED}✗ $fname (${fsize_mb}MB) — CORROMPIDO${RESET}"
            rm -f "$f" "${f}.crom"
        else
            echo -e "  ${GREEN}✓ $fname (${fsize_mb}MB) — OK${RESET}"
        fi
    done
    
    echo ""
    sleep 3
}

# ==============================================================================
# MENU PRINCIPAL
# ==============================================================================
while true; do
    draw_header
    echo "=== 🤖 MODO CHAT DIRETO (Disco → Llama) ==="
    echo "  1. 🟢 Qwen-2.5 0.5B       (~29MB)   Ultra-rápido"
    echo "  2. 🟢 TinyLlama 1.1B       (~638MB)  Rápido"
    echo "  3. 🟢 Llama-3.2 1B         (~770MB)  Bom"
    echo "  4. 🟡 DeepSeek-R1 1.5B     (~1.1GB)  Raciocínio"
    echo "  5. 🟡 Phi-3 Mini 3.8B      (~2.3GB)  Excelente"
    echo "  6. 🔴 DeepSeek-R1 7B       (~4.5GB)  Avançado"
    echo ""
    echo "=== 🧠 MODO CROM VFS (Pack → Mount → Llama) ==="
    echo "  7. 🌌 Mistral Small 24B Q4  (~14GB)  VFS Paging"
    echo "  8. 🌌 Gemma 2 27B Q4        (~18GB)  VFS Paging"
    echo "  9. 🌌 Qwen 2.5 32B Q4       (~22GB)  VFS Paging"
    echo " 10. 🌌 Llama-3.3 70B Q4      (~40GB)  EXPERIMENTAL"
    echo ""
    echo "=== 🔧 FERRAMENTAS ==="
    echo " 11. 📝 Benchmark Automatizado"
    echo " 12. ⚙️  Re-Download Engine"
    echo " 13. 🧹 Limpar Corrompidos"
    echo "  0. Sair"
    echo ""
    read -p "Escolha [0-13]: " op
    
    case $op in
        # Modelos diretos (sem VFS)
        1) run_model_chat "Qwen-2.5 0.5B" \
            "Qwen/Qwen2.5-0.5B-Instruct-GGUF" \
            "qwen2.5-0.5b-instruct-q4_k_m.gguf" \
            "true" 2048 20 ;;
        2) run_model_chat "TinyLlama 1.1B" \
            "TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF" \
            "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf" \
            "true" 2048 500 ;;
        3) run_model_chat "Llama-3.2 1B" \
            "bartowski/Llama-3.2-1B-Instruct-GGUF" \
            "Llama-3.2-1B-Instruct-Q4_K_M.gguf" \
            "true" 2048 600 ;;
        4) run_model_chat "DeepSeek-R1 1.5B" \
            "bartowski/DeepSeek-R1-Distill-Qwen-1.5B-GGUF" \
            "DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf" \
            "true" 1024 800 ;;
        5) run_model_chat "Phi-3 Mini 3.8B" \
            "bartowski/Phi-3-mini-4k-instruct-GGUF" \
            "Phi-3-mini-4k-instruct-Q4_K_M.gguf" \
            "true" 512 1800 ;;
        6) run_model_chat "DeepSeek-R1 7B" \
            "bartowski/DeepSeek-R1-Distill-Qwen-7B-GGUF" \
            "DeepSeek-R1-Distill-Qwen-7B-Q4_K_M.gguf" \
            "true" 512 3500 ;;
        
        # Modelos CROM VFS (Pack + Mount + Llama)
        7) 
            echo -e "\n${RED}⚠️  PIPELINE CROM VFS: Pack → Mount → Inferência${RESET}"
            run_model_chat "Mistral Small 24B" \
                "bartowski/Mistral-Small-24B-Instruct-2501-GGUF" \
                "Mistral-Small-24B-Instruct-2501-Q4_K_M.gguf" \
                "true" 1024 12000
            ;;
        8)
            echo -e "\n${RED}⚠️  PIPELINE CROM VFS${RESET}"
            run_model_chat "Gemma 2 27B" \
                "bartowski/gemma-2-27b-it-GGUF" \
                "gemma-2-27b-it-Q4_K_M.gguf" \
                "true" 1024 15000
            ;;
        9) 
            echo -e "\n${RED}⚠️  PIPELINE CROM VFS${RESET}"
            run_model_chat "Qwen-2.5 32B" \
                "Qwen/Qwen2.5-32B-Instruct-GGUF" \
                "qwen2.5-32b-instruct-q4_k_m.gguf" \
                "true" 512 18000
            ;;
        10) 
            echo -e "\n${RED}⚠️  MODO EXPERIMENTAL (70B)${RESET}"
            run_model_chat "Llama-3.3 70B" \
                "bartowski/Llama-3.3-70B-Instruct-GGUF" \
                "Llama-3.3-70B-Instruct-Q4_K_M.gguf" \
                "true" 256 35000
            ;;
        
        11) run_benchmark_report ;;
        12)
            rm -rf "$BIN_DIR/llama-cli"
            verify_llama_cpp
            sleep 2
            ;;
        13) clean_corrupted ;;
        0)
            echo -e "${CYAN}Desativando Terminal CROM 102...${RESET}"
            exit 0
            ;;
        *)
            echo "Opção inválida."
            sleep 1
            ;;
    esac
done
