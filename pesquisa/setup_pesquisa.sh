#!/bin/bash

# ==============================================================================
# 🚀 Crompressor Research: High-Fidelity Indexer (V3.0 — Hierárquico)
# Gera o índice index.md baseado na nova estrutura de categorias.
# ==============================================================================

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INDEX_FILE="$PROJECT_DIR/index.md"

echo "📝 Gerando índice hierárquico (index.md)..."

cat <<EOF > "$INDEX_FILE"
# 🧪 Pesquisa e Viabilidade Técnica: Crompressor V3

Este arquivo é o ponto central de auditoria para todos os experimentos de compressão soberana.
Reorganizado em **8 categorias temáticas** para navegabilidade.

---

EOF

# Adicionar Manifesto primeiro
if [ -f "$PROJECT_DIR/Manifesto.md" ]; then
    echo "- 📄 **[Manifesto de Auditoria](./Manifesto.md)**" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"
fi

# Categorias na ordem lógica
CATEGORIES="core_engine rede_p2p seguranca performance ia_llm fronteira web_lab v26_lab"

CATEGORY_LABELS="core_engine:🔧 Core Engine (Fundação)
rede_p2p:🌐 Rede & P2P
seguranca:🔒 Segurança & Crypto
performance:⚡ Performance & Hardware
ia_llm:🧠 IA & LLM
fronteira:🚀 Astrofísica & Fronteira
web_lab:🌐 Web Simulations
v26_lab:🔬 V26 Fractal Lab"

for cat in $CATEGORIES; do
    CAT_DIR="$PROJECT_DIR/$cat"
    if [ ! -d "$CAT_DIR" ]; then
        continue
    fi

    # Buscar label da categoria
    LABEL=$(echo "$CATEGORY_LABELS" | grep "^$cat:" | cut -d: -f2-)
    if [ -z "$LABEL" ]; then
        LABEL="$cat"
    fi

    echo "## $LABEL" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"

    # Listar subpastas numeradas nesta categoria
    find "$CAT_DIR" -maxdepth 1 -type d -name "[0-9]*" | sort -V | while read DIR; do
        CLEAN_PATH="${DIR#$PROJECT_DIR/}"
        BASENAME=$(basename "$DIR")
        LABEL_ITEM=$(echo "$BASENAME" | sed 's/-/ /g' | sed 's/_/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)} 1')

        if [ -f "$DIR/index.html" ]; then
            echo "- 🌐 [$LABEL_ITEM](./$CLEAN_PATH/index.html)" >> "$INDEX_FILE"
        elif [ -f "$DIR/relatorio.md" ]; then
            echo "- 📄 [$LABEL_ITEM](./$CLEAN_PATH/relatorio.md)" >> "$INDEX_FILE"
        elif [ -f "$DIR/Relatorio_Gzip_Zstd_Crompressor.md" ]; then
            echo "- 📄 [$LABEL_ITEM](./$CLEAN_PATH/Relatorio_Gzip_Zstd_Crompressor.md)" >> "$INDEX_FILE"
        else
            echo "# Relatório Ausente (Terminal-Only Audit)
Este experimento foi rodado via terminal e ainda não gerou um artefato markdown consolidado." > "$DIR/relatorio.md"
            echo "- 📄 [$LABEL_ITEM](./$CLEAN_PATH/relatorio.md)" >> "$INDEX_FILE"
        fi
    done

    echo "" >> "$INDEX_FILE"
    echo "---" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"
done

cat <<EOF >> "$INDEX_FILE"
## 🌐 Plataforma de Testes Web (web_lab/)

Para auditar as simulações gráficas e laboratórios de IA in-browser, inicie o hub local:

\`\`\`bash
cd pesquisa
python3 -m http.server 8091
\`\`\`

Acesse: **\`http://localhost:8091/web_audit_lab.html\`**

---

## 🛠️ Comandos de Atualização
\`\`\`bash
cd pesquisa
./setup_pesquisa.sh
\`\`\`

> [!CAUTION]
> **Atenção**: Este repositório é auditado bit-a-bit. A modificação manual dos relatórios sem a execução dos benchmarks correspondentes invalida a conformidade SRE.
EOF

echo "✅ Sucesso! O índice hierárquico foi regenerado em: $INDEX_FILE"
