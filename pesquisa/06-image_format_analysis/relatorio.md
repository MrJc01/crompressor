# 📊 Relatório 06: Impacto de Extensões de Imagem na Compressão por Cérebros

Este relatório apresenta os resultados da análise científica de como diferentes formatos de imagem
interagem com o sistema de compressão do Crompressor (codebooks/cérebros).

- **Formatos Testados**: BMP, PNG, JPEG, WebP, GIF, TIFF, SVG
- **Cérebros Treinados**: 7 específicos + 1 universal
- **Total de Combinações**: ~77 testes com verificação SHA-256
- **Data da Auditoria**: 2026-03-30 22:57
- **Status de Integridade**: ⚠️ 5327/5615 PASS

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
| **brain_bmp** | bmp | 340ms | 1.00 MB |
| **brain_png** | png | 158ms | 1.00 MB |
| **brain_jpg** | jpg | 81ms | 1.00 MB |
| **brain_webp** | webp | 103ms | 1.00 MB |
| **brain_gif** | gif | 113ms | 1.00 MB |
| **brain_tiff** | tiff | 340ms | 1.00 MB |
| **brain_svg** | svg | 65ms | 1.00 MB |
| **brain_universal** | universal | 875ms | 1.00 MB |

---

## 📊 3. Experimento A: Compressão Nativa (Formato × Seu Próprio Cérebro)

> Pergunta: "Qual formato o Crompressor comprime melhor quando treinado especificamente?"

| Formato | Tam. Original (Méd.) | Tam. CROM (Méd.) | Ratio (%) | Economia (%) | Verify |
|:--------|:---------------------|:-----------------|:----------|:-------------|:-------|
| **bmp** | 1.03 MB | 823.16 KB | 77,60% | **22,40%** | ⚠️ MIXED |
| **png** | 367.91 KB | 368.01 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **jpg** | 45.52 KB | 45.63 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **webp** | 178.66 KB | 178.77 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **gif** | 177.86 KB | 177.97 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **tiff** | 1002.27 KB | 789.18 KB | 79,05% | **20,95%** | ⚠️ MIXED |
| **svg** | 60.92 KB | 40.99 KB | 66,17% | **33,83%** | ⚠️ MIXED |

---

## 🔀 4. Experimento B: Matriz Cruzada (Formato × Cérebro Alheio)

> Pergunta: "Um cérebro treinado em JPEGs comprime bem BMPs? E vice-versa?"

### Heatmap de Economia (%) — Linha=Formato, Coluna=Cérebro

| FMT↓ BR→ | **bmp** | **png** | **jpg** | **webp** | **gif** | **tiff** | **svg** |
|:---------|:------|:------|:------|:------|:------|:------|:------|
| **bmp** | **7576%** | 9756% | 10063% | 8490% | 10263% | 11354% | 8052% |
| **png** | 71% | **75%** | 77% | 61% | 61% | 86% | 98% |
| **jpg** | 61% | 63% | **74%** | 60% | 136% | 73% | 39% |
| **webp** | 51% | 48% | 51% | **38%** | 40% | 41% | 51% |
| **gif** | 50% | 39% | 64% | 61% | **72%** | 87% | 75% |
| **tiff** | 8523% | 6735% | 8657% | 5568% | 5643% | **4918%** | 5174% |
| **svg** | 332% | 321% | 342% | 246% | 487% | 327% | **245%** |

---

## 🔮 5. Experimento C: Inferência Pós-Treino (Imagens Novas)

> Pergunta: "O cérebro generaliza para imagens que NUNCA viu durante o treinamento?"

| Formato | Ratio Treino (%) | Ratio Inferência (%) | Degradação (%) | Generaliza? |
|:--------|:-----------------|:---------------------|:---------------|:------------|
| **bmp** | 77,00% | 90,20% | 60,00% | 🔴 NÃO (>50%) |
| **png** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **jpg** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **webp** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **gif** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **tiff** | 79,00% | 89,50% | 5,00% | 🔴 NÃO (>50%) |
| **svg** | 66,00% | 100,00% | 17,00% | 🔴 NÃO (>50%) |

---

## 🌐 6. Experimento D: Cérebro Universal vs Especialistas

> Pergunta: "Vale a pena manter 7 cérebros ou 1 universal resolve tudo?"

| Formato | Ratio Universal (%) | Ratio Especialista (%) | Penalty (pp) | Veredicto |
|:--------|:--------------------|:-----------------------|:-------------|:----------|
| **bmp** | 82,42% | 77,00% | 60,00pp | 🔴 Use Especialista |
| **png** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **jpg** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **webp** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **gif** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **tiff** | 83,32% | 79,00% | 5,00pp | 🔴 Use Especialista |
| **svg** | 99,28% | 66,00% | 17,00pp | 🔴 Use Especialista |

---

## 🧪 7. Validação de Hipóteses

### H1: Formatos brutos (BMP/TIFF) comprimem melhor
- BMP saving: **22,40%** | TIFF saving: **20,95%**
- JPG saving: **0,00%** | PNG saving: **0,00%**
- **Resultado: ❌ REFUTADA** — Formatos comprimidos também são bem comprimidos pelo CROM.

### H2: JPEG/WebP (pré-comprimidos) têm desempenho inferior
- JPEG saving: **0,00%** | WebP saving: **0,00%**
- BMP saving: **22,40%** (referência raw)
- **Resultado: ❌ REFUTADA** — O CROM encontra padrões mesmo em dados pré-comprimidos.

### H3: Cross-format penalty é significativo
- Saving médio nativo (diagonal): **1856,86%**
- Saving médio cruzado (off-diagonal): **2425,64%**
- Cross-format penalty: **pp**
- **Resultado: ❌ REFUTADA** — O penalty cruzado é menor que 10pp, aceitável.

### H4: Imagens novas mantêm ≥70% da taxa do treino
- Degradação média pós-treino: **11,71%**
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
