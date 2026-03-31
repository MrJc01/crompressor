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

# 2. Listar todas as pastas numeradas de pesquisas (01 a 101)
# Lógica: Usa o nome da pasta como Label para hierarquia clara.
find . -maxdepth 1 -type d -name "[0-9]*" | sort -V | while read DIR; do
    CLEAN_PATH="${DIR#./}"
    LABEL=$(echo "$CLEAN_PATH" | sed 's/-/ /g' | sed 's/_/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)} 1')
    
    # Verifica qual o arquivo principal da pasta para linkar (HTML ou MD)
    if [ -f "$DIR/index.html" ]; then
        echo "- 🌐 [$LABEL](./$CLEAN_PATH/index.html)" >> "$INDEX_FILE"
    elif [ -f "$DIR/relatorio.md" ]; then
        echo "- 📄 [$LABEL](./$CLEAN_PATH/relatorio.md)" >> "$INDEX_FILE"
    elif [ -f "$DIR/Relatorio_Gzip_Zstd_Crompressor.md" ]; then
        echo "- 📄 [$LABEL](./$CLEAN_PATH/Relatorio_Gzip_Zstd_Crompressor.md)" >> "$INDEX_FILE"
    else
        # Se não tem relatório md, cria um placeholder genérico para constatar no index
        echo "# Relatório Ausente (Terminal-Only Audit)
Este experimento foi rodado via terminal e ainda não gerou um artefato markdown consolidado." > "$DIR/relatorio.md"
        echo "- 📄 [$LABEL](./$CLEAN_PATH/relatorio.md)" >> "$INDEX_FILE"
    fi
done

cat <<EOF >> "$INDEX_FILE"

---

## 🌐 Plataforma de Testes Web (Testes 91-101)

Para auditar as simulações gráficas, aceleração 3D e laboratórios de Inteligência Artificial in-browser (LLMs / WebGPU), inicie o hub local:

\`\`\`bash
cd pesquisa
python3 -m http.server 8091
\`\`\`

Em seguida, acesse no navegador: **\`http://localhost:8091/web_audit_lab.html\`**

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
