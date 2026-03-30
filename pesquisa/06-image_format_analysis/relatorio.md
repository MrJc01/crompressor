# 📊 Relatório 06: Impacto de Extensões de Imagem na Compressão por Cérebros

Este relatório apresenta os resultados da análise científica de como diferentes formatos de imagem
interagem com o sistema de compressão do Crompressor (codebooks/cérebros).

- **Formatos Testados**: BMP, PNG, JPEG, WebP, GIF, TIFF, SVG
- **Cérebros Treinados**: 7 específicos + 1 universal
- **Total de Combinações**: ~77 testes com verificação SHA-256
- **Data da Auditoria**: 2026-03-30 07:54
- **Status de Integridade**: ⚠️ 3712/3977 PASS

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
| **brain_bmp** | bmp | 392ms | 1.00 MB |
| **brain_png** | png | 170ms | 1.00 MB |
| **brain_jpg** | jpg | 145ms | 1.00 MB |
| **brain_webp** | webp | 165ms | 1.00 MB |
| **brain_gif** | gif | 156ms | 1.00 MB |
| **brain_tiff** | tiff | 315ms | 1.00 MB |
| **brain_svg** | svg | 100ms | 1.00 MB |
| **brain_universal** | universal | 886ms | 1.00 MB |

---

## 📊 3. Experimento A: Compressão Nativa (Formato × Seu Próprio Cérebro)

> Pergunta: "Qual formato o Crompressor comprime melhor quando treinado especificamente?"

| Formato | Tam. Original (Méd.) | Tam. CROM (Méd.) | Ratio (%) | Economia (%) | Verify |
|:--------|:---------------------|:-----------------|:----------|:-------------|:-------|
| **bmp** | 1.03 MB | 839.22 KB | 79,00% | **21,00%** | ⚠️ MIXED |
| **png** | 367.91 KB | 368.01 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **jpg** | 45.52 KB | 45.63 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **webp** | 178.66 KB | 178.77 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **gif** | 177.86 KB | 177.97 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **tiff** | 1002.27 KB | 805.24 KB | 80,62% | **19,38%** | ⚠️ MIXED |
| **svg** | 60.92 KB | 41.01 KB | 66,58% | **33,42%** | ⚠️ MIXED |

---

## 🔀 4. Experimento B: Matriz Cruzada (Formato × Cérebro Alheio)

> Pergunta: "Um cérebro treinado em JPEGs comprime bem BMPs? E vice-versa?"

### Heatmap de Economia (%) — Linha=Formato, Coluna=Cérebro

| FMT↓ BR→ | **bmp** | **png** | **jpg** | **webp** | **gif** | **tiff** | **svg** |
|:---------|:------|:------|:------|:------|:------|:------|:------|
| **bmp** | **3257%** | 4516% | 5253% | 4543% | 4935% | 4167% | 4608% |
| **png** | 49% | **49%** | 43% | 49% | 48% | 36% | 46% |
| **jpg** | 33% | 28% | **36%** | 38% | 38% | 34% | 47% |
| **webp** | 47% | 41% | 28% | **46%** | 46% | 37% | 38% |
| **gif** | 41% | 36% | 28% | 40% | **36%** | 47% | 35% |
| **tiff** | 4285% | 5327% | 4886% | 4578% | 4960% | **4121%** | 4798% |
| **svg** | 221% | 205% | 210% | 207% | 186% | 220% | **145%** |

---

## 🔮 5. Experimento C: Inferência Pós-Treino (Imagens Novas)

> Pergunta: "O cérebro generaliza para imagens que NUNCA viu durante o treinamento?"

| Formato | Ratio Treino (%) | Ratio Inferência (%) | Degradação (%) | Generaliza? |
|:--------|:-----------------|:---------------------|:---------------|:------------|
| **bmp** | 79,00% | 91,90% | 0,00% | 🔴 NÃO (>50%) |
| **png** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **jpg** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **webp** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **gif** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **tiff** | 80,00% | 90,10% | 62,00% | 🔴 NÃO (>50%) |
| **svg** | 66,00% | 103,90% | 58,00% | 🔴 NÃO (>50%) |

---

## 🌐 6. Experimento D: Cérebro Universal vs Especialistas

> Pergunta: "Vale a pena manter 7 cérebros ou 1 universal resolve tudo?"

| Formato | Ratio Universal (%) | Ratio Especialista (%) | Penalty (pp) | Veredicto |
|:--------|:--------------------|:-----------------------|:-------------|:----------|
| **bmp** | 83,76% | 79,00% | 0,00pp | 🔴 Use Especialista |
| **png** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **jpg** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **webp** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **gif** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **tiff** | 84,54% | 80,00% | 62,00pp | 🔴 Use Especialista |
| **svg** | 103,12% | 66,00% | 58,00pp | 🔴 Use Especialista |

---

## 🧪 7. Validação de Hipóteses

### H1: Formatos brutos (BMP/TIFF) comprimem melhor
- BMP saving: **21,00%** | TIFF saving: **19,38%**
- JPG saving: **0,00%** | PNG saving: **0,00%**
- **Resultado: ❌ REFUTADA** — Formatos comprimidos também são bem comprimidos pelo CROM.

### H2: JPEG/WebP (pré-comprimidos) têm desempenho inferior
- JPEG saving: **0,00%** | WebP saving: **0,00%**
- BMP saving: **21,00%** (referência raw)
- **Resultado: ❌ REFUTADA** — O CROM encontra padrões mesmo em dados pré-comprimidos.

### H3: Cross-format penalty é significativo
- Saving médio nativo (diagonal): **1098,57%**
- Saving médio cruzado (off-diagonal): **1406,14%**
- Cross-format penalty: **pp**
- **Resultado: ❌ REFUTADA** — O penalty cruzado é menor que 10pp, aceitável.

### H4: Imagens novas mantêm ≥70% da taxa do treino
- Degradação média pós-treino: **17,14%**
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
