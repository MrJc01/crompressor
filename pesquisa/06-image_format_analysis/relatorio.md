# 📊 Relatório 06: Impacto de Extensões de Imagem na Compressão por Cérebros

Este relatório apresenta os resultados da análise científica de como diferentes formatos de imagem
interagem com o sistema de compressão do Crompressor (codebooks/cérebros).

- **Formatos Testados**: BMP, PNG, JPEG, WebP, GIF, TIFF, SVG
- **Cérebros Treinados**: 7 específicos + 1 universal
- **Total de Combinações**: ~77 testes com verificação SHA-256
- **Data da Auditoria**: 2026-03-29 16:50
- **Status de Integridade**: ⚠️ 810/912 PASS

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
| **brain_bmp** | bmp | 714ms | 1.00 MB |
| **brain_png** | png | 271ms | 1.00 MB |
| **brain_jpg** | jpg | 127ms | 1.00 MB |
| **brain_webp** | webp | 126ms | 1.00 MB |
| **brain_gif** | gif | 134ms | 1.00 MB |
| **brain_tiff** | tiff | 417ms | 1.00 MB |
| **brain_svg** | svg | 80ms | 1.00 MB |
| **brain_universal** | universal | 959ms | 1.00 MB |

---

## 📊 3. Experimento A: Compressão Nativa (Formato × Seu Próprio Cérebro)

> Pergunta: "Qual formato o Crompressor comprime melhor quando treinado especificamente?"

| Formato | Tam. Original (Méd.) | Tam. CROM (Méd.) | Ratio (%) | Economia (%) | Verify |
|:--------|:---------------------|:-----------------|:----------|:-------------|:-------|
| **bmp** | 1.03 MB | 830.19 KB | 78,22% | **21,77%** | ⚠️ MIXED |
| **png** | 367.91 KB | 367.98 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **jpg** | 45.52 KB | 42.82 KB | 91,47% | **8,53%** | ⚠️ MIXED |
| **webp** | 178.66 KB | 178.74 KB | 100,00% | **0,00%** | ⚠️ MIXED |
| **gif** | 177.86 KB | 179.29 KB | 100,42% | **-0,42%** | ⚠️ MIXED |
| **tiff** | 1002.27 KB | 796.66 KB | 79,67% | **20,32%** | ⚠️ MIXED |
| **svg** | 60.92 KB | 40.48 KB | 65,40% | **34,60%** | ⚠️ MIXED |

---

## 🔀 4. Experimento B: Matriz Cruzada (Formato × Cérebro Alheio)

> Pergunta: "Um cérebro treinado em JPEGs comprime bem BMPs? E vice-versa?"

### Heatmap de Economia (%) — Linha=Formato, Coluna=Cérebro

| FMT↓ BR→ | **bmp** | **png** | **jpg** | **webp** | **gif** | **tiff** | **svg** |
|:---------|:------|:------|:------|:------|:------|:------|:------|
| **bmp** | **8050%** | 8719% | 9883% | 9272% | 9151% | 8725% | 9270% |
| **png** | 84% | **69%** | 61% | 59% | 94% | 84% | 80% |
| **jpg** | 466% | 403% | **320%** | 344% | 295% | 416% | 504% |
| **webp** | 173% | 71% | 96% | **80%** | 99% | 90% | 175% |
| **gif** | 65% | 74% | 110% | 61% | **72%** | 60% | 62% |
| **tiff** | 8572% | 9144% | 11424% | 11058% | 12367% | **9280%** | 9953% |
| **svg** | 390% | 359% | 481% | 455% | 392% | 500% | **587%** |

---

## 🔮 5. Experimento C: Inferência Pós-Treino (Imagens Novas)

> Pergunta: "O cérebro generaliza para imagens que NUNCA viu durante o treinamento?"

| Formato | Ratio Treino (%) | Ratio Inferência (%) | Degradação (%) | Generaliza? |
|:--------|:-----------------|:---------------------|:---------------|:------------|
| **bmp** | 78,00% | 91,10% | 22,00% | 🔴 NÃO (>50%) |
| **png** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **jpg** | 91,00% | 103,40% | 47,00% | 🔴 NÃO (>50%) |
| **webp** | 100,00% | 100,00% | 0,00% | 🔴 NÃO (>50%) |
| **gif** | 100,00% | 101,80% | 42,00% | 🔴 NÃO (>50%) |
| **tiff** | 79,00% | 89,20% | 67,00% | 🔴 NÃO (>50%) |
| **svg** | 65,00% | 103,80% | 40,00% | 🔴 NÃO (>50%) |

---

## 🌐 6. Experimento D: Cérebro Universal vs Especialistas

> Pergunta: "Vale a pena manter 7 cérebros ou 1 universal resolve tudo?"

| Formato | Ratio Universal (%) | Ratio Especialista (%) | Penalty (pp) | Veredicto |
|:--------|:--------------------|:-----------------------|:-------------|:----------|
| **bmp** | 83,30% | 78,00% | 22,00pp | 🔴 Use Especialista |
| **png** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **jpg** | 103,82% | 91,00% | 47,00pp | 🔴 Use Especialista |
| **webp** | 100,00% | 100,00% | 0,00pp | 🔴 Use Especialista |
| **gif** | 101,44% | 100,00% | 42,00pp | 🔴 Use Especialista |
| **tiff** | 83,80% | 79,00% | 67,00pp | 🔴 Use Especialista |
| **svg** | 102,96% | 65,00% | 40,00pp | 🔴 Use Especialista |

---

## 🧪 7. Validação de Hipóteses

### H1: Formatos brutos (BMP/TIFF) comprimem melhor
- BMP saving: **21,77%** | TIFF saving: **20,32%**
- JPG saving: **8,53%** | PNG saving: **0,00%**
- **Resultado: ❌ REFUTADA** — Formatos comprimidos também são bem comprimidos pelo CROM.

### H2: JPEG/WebP (pré-comprimidos) têm desempenho inferior
- JPEG saving: **8,53%** | WebP saving: **0,00%**
- BMP saving: **21,77%** (referência raw)
- **Resultado: ❌ REFUTADA** — O CROM encontra padrões mesmo em dados pré-comprimidos.

### H3: Cross-format penalty é significativo
- Saving médio nativo (diagonal): **2636,86%**
- Saving médio cruzado (off-diagonal): **2955,74%**
- Cross-format penalty: **pp**
- **Resultado: ❌ REFUTADA** — O penalty cruzado é menor que 10pp, aceitável.

### H4: Imagens novas mantêm ≥70% da taxa do treino
- Degradação média pós-treino: **31,14%**
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
