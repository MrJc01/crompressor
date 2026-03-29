#!/bin/bash

# ==============================================================================
# 🚀 Crompressor Research: High-Fidelity Indexer (V2.0)
# Este script gera o índice index.md baseado na estrutura real de testes.
# ==============================================================================

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INDEX_FILE="$PROJECT_DIR/index.md"

echo "📝 Gerando índice dinâmico (index.md)..."

cat <<EOF > "$INDEX_FILE"
# 🧪 Pesquisa e Viabilidade Técnica: Crompressor V3

Este arquivo é o ponto central de auditoria para todos os experimentos de compressão soberana.

---

## 📂 Mapa de Documentação Técnica

EOF

# 1. Adicionar Manifesto primeiro
if [ -f "$PROJECT_DIR/Manifesto.md" ]; then
    echo "- 📄 **[Manifesto de Auditoria](./Manifesto.md)**" >> "$INDEX_FILE"
fi

# 2. Listar relatórios reais (Excluir Manifesto, Index, e pastas auxiliares)
# Lógica: Usa o nome da pasta como Label para hierarquia clara.
find . -name "*.md" \
    -not -name "index.md" \
    -not -name "Manifesto.md" \
    -not -path "./datasets/*" \
    -not -path "./scripts/*" | sort | while read FILE; do
    
    CLEAN_PATH="${FILE#./}"
    DIR_NAME=$(dirname "$CLEAN_PATH")
    
    if [ "$DIR_NAME" == "." ]; then
        # Arquivos na raiz (se houver outros além do manifesto)
        FILE_NAME=$(basename "$CLEAN_PATH" .md)
        LABEL=$(echo "$FILE_NAME" | sed 's/_/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)} 1')
    else
        # Usar o Nome da Pasta como Label (ex: 01-logs_redundancia -> 01 Logs Redundancia)
        LABEL=$(echo "$DIR_NAME" | sed 's/-/ /g' | sed 's/_/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)} 1')
    fi
    
    echo "- 📄 [$LABEL](./$CLEAN_PATH)" >> "$INDEX_FILE"
done

cat <<EOF >> "$INDEX_FILE"

---

## 🛠️ Comandos de Atualização
Para regenerar este dossiê, execute:
\`\`\`bash
cd pesquisa
./setup_pesquisa.sh
\`\`\`

> [!CAUTION]
> **Atenção**: Este repositório é auditado bit-a-bit. A modificação manual dos relatórios sem a execução dos benchmarks correspondentes invalida a conformidade SRE.
EOF

echo "✅ Sucesso! O índice foi regenerado em: $INDEX_FILE"
chmod +x "$PROJECT_DIR/setup_pesquisa.sh"
