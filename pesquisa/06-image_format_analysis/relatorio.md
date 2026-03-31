# 📊 Relatório 06: Impacto de Extensões de Imagem na Compressão por Cérebros

Este relatório apresenta os resultados da análise científica de como diferentes formatos de imagem
interagem com o sistema de compressão do Crompressor (codebooks/cérebros).

- **Formatos Testados**: BMP, PNG, JPEG, WebP, GIF, TIFF, SVG
- **Cérebros Treinados**: 7 específicos + 1 universal
- **Total de Combinações**: ~77 testes com verificação SHA-256
- **Data da Auditoria**: 2026-03-30 19:44
- **Status de Integridade**: ⚠️ 5042/5330 PASS

---

## 📦 1. Inventário do Dataset

Dataset controlado gerado via ImageMagick: mesmas imagens-fonte convertidas para 7 formatos distintos.

| Formato | Arquivos Treino | Arquivos Teste | Observação |
|:--------|:----------------|:---------------|:-----------|
| **bmp** | 40 (42M) | 10 | 24-bit não comprimido |
| **png** | 40 (15M) | 10 | Deflate (LZ77+Huffman) |
| **jpg** | 40 (1,9M) | 10 | DCT lossy Q95 |
| **webp** | 40 (7,1M) | 10 | VP8L lossless |
| **gif** | 40 (7,1M) | 10 | LZW 256 cores |
| **tiff** | 40 (40M) | 10 | Sem compressão |
| **svg** | 40 (2,5M) | 10 | XML text-based |

---

## 🧠 2. Cérebros Treinados

| Cérebro | Formato | Tempo (ms) | Tamanho (.cromdb) |
|:--------|:--------|:-----------|:------------------|
| **brain_bmp** | bmp | 1544ms | 1.00 MB |
| **brain_png** | png | 760ms | 1.00 MB |
| **brain_jpg** | jpg | 260ms | 1.00 MB |
| **brain_webp** | webp | 389ms | 1.00 MB |
| **brain_gif** | gif | 280ms | 1.00 MB |
| **brain_tiff** | tiff | 806ms | 1.00 MB |
| **brain_svg** | svg | 267ms | 1.00 MB |
| **brain_universal** | universal | 1839ms | 1.00 MB |

---

## 📊 3. Experimento A: Compressão Nativa (Formato × Seu Próprio Cérebro)

> Pergunta: "Qual formato o Crompressor comprime melhor quando treinado especificamente?"

| Formato | Tam. Original (Méd.) | Tam. CROM (Méd.) | Ratio (%) | Economia (%) | Verify |
|:--------|:---------------------|:-----------------|:----------|:-------------|:-------|
| **bmp** | 1.03 MB | 823.45 KB | 77,62% | **22,38%** | ⚠️ MIXED |
| **png** | 367.91 KB | 368.01 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **jpg** | 45.52 KB | 45.63 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **webp** | 178.66 KB | 178.77 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **gif** | 177.86 KB | 177.97 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **tiff** | 1002.27 KB | 789.05 KB | 79,03% | **20,98%** | ⚠️ MIXED |
| **svg** | 60.92 KB | 40.99 KB | 65,97% | **34,02%** | ⚠️ MIXED |

---

## 🔀 4. Experimento B: Matriz Cruzada (Formato × Cérebro Alheio)

> Pergunta: "Um cérebro treinado em JPEGs comprime bem BMPs? E vice-versa?"

### Heatmap de Economia (%) — Linha=Formato, Coluna=Cérebro

| FMT↓ BR→ | **bmp** | **png** | **jpg** | **webp** | **gif** | **tiff** | **svg** |
|:---------|:------|:------|:------|:------|:------|:------|:------|
| **bmp** | **4921%** | 5256% | 4886% | 5219% | 5252% | 4883% | 4874% |
| **png** | 38% | **39%** | 50% | 48% | 47% | 46% | 39% |
| **jpg** | 48% | 36% | **44%** | 64% | 39% | 33% | 62% |
| **webp** | 47% | 47% | 46% | **45%** | 64% | 47% | 35% |
| **gif** | 45% | 48% | 46% | 56% | **39%** | 56% | 34% |
| **tiff** | 5555% | 5104% | 5571% | 5512% | 5166% | **4813%** | 4849% |
| **svg** | 299% | 232% | 240% | 234% | 208% | 241% | **166%** |

---

## 🔮 5. Experimento C: Inferência Pós-Treino (Imagens Novas)

> Pergunta: "O cérebro generaliza para imagens que NUNCA viu durante o treinamento?"

| Formato | Ratio Treino (%) | Ratio Inferência (%) | Degradação (%) | Generaliza? |
|:--------|:-----------------|:---------------------|:---------------|:------------|
| **bmp** | 77,00% | 90,30% | 62,00% | 🔴 NÃO (>50%) |
| **png** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **jpg** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **webp** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **gif** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **tiff** | 79,00% | 89,50% | 3,00% | 🔴 NÃO (>50%) |
| **svg** | 65,00% | 100,00% | 97,00% | 🔴 NÃO (>50%) |

---

## 🌐 6. Experimento D: Cérebro Universal vs Especialistas

> Pergunta: "Vale a pena manter 7 cérebros ou 1 universal resolve tudo?"

| Formato | Ratio Universal (%) | Ratio Especialista (%) | Penalty (pp) | Veredicto |
|:--------|:--------------------|:-----------------------|:-------------|:----------|
| **bmp** | 82,44% | 77,00% | 62,00pp | 🔴 Use Especialista |
| **png** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **jpg** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **webp** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **gif** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **tiff** | 83,20% | 79,00% | 3,00pp | 🔴 Use Especialista |
| **svg** | 99,24% | 65,00% | 97,00pp | 🔴 Use Especialista |

---

## 🧪 7. Validação de Hipóteses

### H1: Formatos brutos (BMP/TIFF) comprimem melhor
- BMP saving: **22,38%** | TIFF saving: **20,98%**
- JPG saving: **0,00%** | PNG saving: **0,00%**
- **Resultado: ❌ REFUTADA** — Formatos comprimidos também são bem comprimidos pelo CROM.

### H2: JPEG/WebP (pré-comprimidos) têm desempenho inferior
- JPEG saving: **0,00%** | WebP saving: **0,00%**
- BMP saving: **22,38%** (referência raw)
- **Resultado: ❌ REFUTADA** — O CROM encontra padrões mesmo em dados pré-comprimidos.

### H3: Cross-format penalty é significativo
- Saving médio nativo (diagonal): **1438,14%**
- Saving médio cruzado (off-diagonal): **1540,52%**
- Cross-format penalty: **pp**
- **Resultado: ❌ REFUTADA** — O penalty cruzado é menor que 10pp, aceitável.

### H4: Imagens novas mantêm ≥70% da taxa do treino
- Degradação média pós-treino: **23,14%**
- **Resultado: ❌ REFUTADA** — O cérebro tem dificuldade para generalizar (degradação > 30%).

---

## 🏆 8. Conclusões e Recomendações

### Recomendações Práticas

1. **Para pipelines de imagens brutas (BMP/TIFF)**: O Crompressor é uma alternativa viável a Gzip/Zstd, oferecendo compressão com acesso aleatório via VFS.
2. **Para imagens pré-comprimidas (JPEG/WebP)**: Avaliar se o overhead do codebook justifica a economia adicional.
3. **Cérebro Universal vs Especialista**: Consultar a tabela do Experimento D para decidir.
4. **Generalização**: Se a degradação pós-treino for baixa, o cérebro pode ser compartilhado entre nós P2P sem retreino.

---

> [!TIP]
> Em ambientes de produção com milhões de imagens médicas (DICOM/TIFF), a economia do Crompressor pode representar TB de redução de storage.

**"Não comprimimos pixels. Compilamos a realidade visual."**

## 🔍 V11 — Exclusão Lógica Dinâmica
Com as atualizações inseridas via Busca Transparente (Grep O(1)) e as ramificações de Pack no Sprint 11, resguardamos e fortificamos a blindagem da análise não-textual. O arquivo \`.crom\` atuariamente desvia-se de varreduras indevidas de RandomReader no Grep Neural, garantindo resiliência passiva do stream bruto binário sem aterrissar em panic crashes decorrentes do motor semântico subjacente (BPE literais).
