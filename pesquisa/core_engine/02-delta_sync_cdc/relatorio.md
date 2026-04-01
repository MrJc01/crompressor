# 📊 Relatório 02: Delta Sync & Content-Defined Chunking (CDC)

Este relatório detalha a eficácia do algoritmo CDC do Crompressor V5 na redução de payload de sincronização entre versões de Banco de Dados SQL.

- **Dataset**: `pesquisa/datasets/dump_v1.sql` (5.72 MB)
- **Codebook**: Treinado dinamicamente com BPE Neural (Universal Codebook)
- **Motor**: Crompressor V10 (Neural BPE Tokenizer + Merkle/CDC)
- **Status de Integridade**: ✅ PASS (SHA-256 bit-a-bit)

## 📈 Métricas de Granularidade

| Atributo | Crompressor V10 | Crompressor V5 (anterior) | Observação |
| :--- | :--- | :--- | :--- |
| **Tamanho Original** | 5,727,877 B | 5,727,877 B | Dump SQL Bruto |
| **Peso Compilado (.crom)** | 1,338,431 B | 1,351,706 B | Redução aprimorada pelo BPE |
| **Header Version** | V5 (112 bytes) | V5 (112 bytes) | — |
| **Economia de Espaço** | **76.63%** | **76.4%** | Refinamento Neural Codebook |
| **Total Chunks** | 44,750 | — | Mapeados na ChunkTable |
| **Entropy (Delta Pool)** | 7.89 bits/byte | — | Dados aleatórios Pós-XOR LSH |
| **CodebookIDs Únicos** | 74 | 414 | **-82%** (BPE encontra a essência) |

## 📐 Distribuição de Padrões (Top-5 CodebookIDs)
```
#01  CodebookID: 340         Count: 4609    (10.30%)
#02  CodebookID: 283         Count: 3141    (7.02%)
#03  CodebookID: 353         Count: 2861    (6.39%)
#04  CodebookID: 380         Count: 1961    (4.38%)
#05  CodebookID: 262         Count: 1795    (4.01%)
```

## 🧠 Análise Técnica: Evolução V10 BPE
A economia saltou no `.crom` novamente (de 1,3MiB para 1.27MiB efetivos) superando a V5. O destaque aqui é incisivo: fomos de 414 Codebooks únicos requeridos para apenas **74 Codebooks Únicos** que mapeiam todo o dump de um banco de dados SQL!

O algoritmo compreendeu e extraiu super-tokens globais como estruturas "INSERT INTO", quebras de linha estáticas e timestamps de transações, engolindo os resíduos com uma entropia otimizada (7.89).

## 🛡️ Conclusão de Auditoria
A transição BPE foi não apenas bem-sucedida, como cirúrgica no CDC. Com apenas 74 Super-Tokens usados, um Node Lite no P2P que receba o Dump SQL precisará de KBínfimos de RAM para alocar o dicionário "Universal" necessário para ler a timeline temporal de backups incrementais do banco.

## 🔍 V11 — Micro-Patching & Grep Neural
- **Micro-Patching**: O compilador V11 agora detecta chunks com similaridade ≥ 80% e gera edit scripts (Levenshtein Diff) como alternativa ao XOR, escolhendo o menor. Isso beneficia especialmente dumps SQL onde `INSERT INTO` varia apenas no valor.
- **Grep Neural**: `crompressor grep "INSERT" -i dump.crom -c dump.cromdb` localiza todas as ocorrências sem descomprimir o arquivo.
