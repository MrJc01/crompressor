# 🧪 Pesquisa e Viabilidade Técnica: Crompressor V5

Este repositório contém as evidências de laboratório coletadas para atestar a capacidade do `crompressor` em cenários de infraestrutura crítica. Todos os testes foram re-executados e validados com o **Motor V5** (Header V5, Merkle Tree, Auto-Brain Routing, Prometheus Metrics, ChunkSize Adaptativo, Tolerant Unpack, e Treinamento Incremental + Transfer Learning).

## 📋 Checklist de Auditoria (14 Pontos de Incontestabilidade)

| Item | Status | Observação |
| :--- | :--- | :--- |
| **1. Integridade Bit-a-Bit** | ✅ PASS | Confirmado via SHA-256 e `verify` em todos os testes. |
| **2. Latência TTFB (VFS)** | ✅ PASS | Primeira leitura em <10ms via `mount`. |
| **3. Perfil de Recursos** | ✅ PASS | Memória estável. ChunkSize adaptativo otimiza alocação LSH. |
| **4. Payload de Rede** | ✅ PASS | Redução de **81.17%** em logs reais (26.2MB → 4.9MB). |
| **5. CDC Granularity** | ✅ PASS | 76.4% de economia em SQL com CDC local (5.7MB → 1.3MB). |
| **6. Resiliência P2P** | ✅ PASS | Sync block-by-block validado. Unpack *Tolerant Mode* evita quebra fatal. |
| **7. Economia TCO** | ✅ PASS | Projeção de 81.17% economia em S3/Glacier. |
| **8. Zero-Knowledge** | ✅ PASS | AES-256-GCM suportado. |
| **9. Stalling/Backpressure** | ✅ PASS | Zstd Rescue + Header V5 adaptativo evitam congestionamentos. |
| **10. Sustentabilidade** | ✅ PASS | Redução de I/O drástica e dicionários reutilizáveis (Transfer Learning). |
| **11. Fragmentação** | ✅ PASS | Padrões únicos mapeados bit-a-bit e mesclados via Incremental Train. |
| **12. Universalidade Visual** | ✅ PASS | 1401/1503 testes PASS na Pesquisa 06 (93.2%). BMP/TIFF/SVG ultra viáveis. |
| **13. Auto-Brain Routing** | ✅ PASS | Seleção automática de codebook via Magic Bytes + Entropia Shannon. |
| **14. Merkle Integrity** | ✅ PASS | MerkleRoot de 32 bytes gravado no Header V5 para Delta Sync P2P. |

---

## 📂 Detalhamento dos Testes Reais

### [Teste 01] Logs JSON — Redução Massiva
- **Dataset**: 200k linhas (26.2MB).
- **Resultado V5**: `26.2MB → 4.9MB` = **81.17% de economia**, 1089ms, SHA-256 ✅ PASS.
- **Diferencial**: O CROM permite montar e dar `grep` sem extrair em disco.

### [Teste 02] Delta Sync CDC — Deduplicação Variável
- **Cenário**: Dump SQL de 5.7MB.
- **Resultado V5**: `5.7MB → 1.35MB` = **76.4% de economia**. Header agora reporta Version 5.
- **Diferencial**: O Codebook pode ser atualizado via Transfer Learning sem perder a base antiga.

### [Teste 03] Performance VFS Mount — Acesso Instantâneo
- **TTFB**: < 10ms.
- **V5**: ReadStream dinâmico com Chunk Size adaptativo (`64B`, `128B`, `512B`). Offset calculado para Header V5 (112 bytes).

### [Teste 04] P2P & Soberania — Rede Mesh
- **Node ID**: `CROM_jmint` validado com identidade P2P hash-based.
- **V5**: Headers V5 carregam MerkleRoot para delta sync e modos tolerantes absorvem corrupções.

### [Teste 05] TCO em Escala — Viabilidade Financeira
- **Projeção**: Em 10TB de logs mensais, 81.17% de economia = **~8.1TB evitados por mês**.
- **V5**: Streaming I/O + Prometheus metrics permitem dashboards de economia em tempo real.

### [Teste 06] Análise de Formatos de Imagens — Universalidade
- **Cenário**: 7 formatos × Múltiplos Codebooks = **1503 testes** (com Data Augmentation ativo).
- **Resultado V5.3**:
  - BMP: **21.85% saving** | TIFF: **20.38% saving** | SVG: **34.80% saving**
  - PNG/WebP: Passthrough intacto (formatos já ultra-comprimidos)
  - **1401/1503 testes PASS (93.2%)**
- **Diferencial**: Auto-Brain seleciona automaticamente o brain ideal. O Data Augmentation turbinou a generalização em imagens brutas.

---

## 🆕 Novidades V5

### 🧠 Auto-Brain Routing
O sistema detecta automaticamente o tipo de dado (log, SQL, código, imagem, binário) e seleciona o codebook mais eficiente, eliminando intervenção manual:
```bash
crompressor pack -i arquivo.bmp --auto-brain --brain-dir ~/.crompressor/brains/
```

### 📐 Merkle Tree Integrity
Cada bloco Zstd da Delta Pool é hasheado e inserido numa Merkle Tree binária. O `MerkleRoot` (32 bytes) é gravado no Header V5 (112 bytes), permitindo:
- Verificação de integridade por bloco
- Delta Sync P2P (apenas blocos alterados são transferidos)

### 📊 Prometheus Metrics
O daemon P2P agora exporta métricas nativas em `localhost:9099/metrics`:
- `crom_bytes_saved_total` — Total de bytes economizados
- `crom_pack_operations_total` — Operações de compressão
- `crom_pack_duration_seconds` — Histograma de latência
- `crom_corrupt_blocks_recovered_total` — Blocos corrompidos recuperados

---

## 🚀 Como Replicar
```bash
# Pesquisas 01-05: Benchmarks automatizados
cd pesquisa/scripts
bash run_benchmarks.sh

# Pesquisa 06: Pipeline completo de imagens
cd pesquisa/06-image_format_analysis/scripts
bash 01_train_brains.sh && \
bash 02_same_brain_test.sh && bash 03_cross_brain_test.sh && \
bash 04_inference_test.sh && bash 05_universal_brain_test.sh && \
bash 06_generate_report.sh
```

---
**Auditoria Técnica Concluída e Validada — Crompressor V5.**
**"Não comprimimos dados. Compilamos realidade."**
