# 🏗️ Arquitetura do Crompressor — Como Funciona

## Visão Geral

O Crompressor é um sistema de compressão baseado em **codebook learning** — ele aprende os padrões dos seus dados e depois usa esse conhecimento para representá-los de forma compacta.

```mermaid
graph LR
    subgraph "Fase 1: Treinamento"
        D[Dados de Treino] --> T[Trainer Engine]
        T --> CB["🧠 Codebook (.cromdb)"]
    end
    subgraph "Fase 2: Compressão"
        I[Arquivo Original] --> C[Chunker]
        C --> S[Search Engine]
        S --> CB
        S --> X[XOR Delta]
        X --> Z[ZSTD Compress]
        Z --> O["📦 Arquivo .crom"]
    end
    subgraph "Fase 3: Restauração"
        O2["📦 .crom"] --> R[Reader]
        R --> XR[XOR Reconstruct]
        CB2["🧠 .cromdb"] --> XR
        XR --> F["📄 Arquivo Original"]
    end
```

---

## Pipeline Detalhado

### 1. Chunking — Dividir para Conquistar

O arquivo de entrada é dividido em **pedaços (chunks)** de tamanho configurável. Existem 2 estratégias:

```mermaid
graph TD
    A[Arquivo de Entrada] --> B{Estratégia?}
    B -->|"--chunk-size 128"| C["Fixed Chunker<br>Blocos de tamanho fixo"]
    B -->|"--cdc"| D["CDC Chunker<br>Rabin Fingerprint"]
    C --> E["Chunk 1 (128B) | Chunk 2 (128B) | Chunk 3 (128B) | ..."]
    D --> F["Chunk 1 (97B) | Chunk 2 (153B) | Chunk 3 (112B) | ..."]
```

| Estratégia | Descrição | Melhor Para |
|:-----------|:----------|:------------|
| **Fixed** | Blocos de tamanho exato (default: 128B) | Dados binários estruturados |
| **CDC** (Content-Defined) | Fronteiras baseadas em hash rolling (Rabin) | Arquivos que sofrem inserções/deleções |

### 2. Search — Encontrar o Padrão Mais Próximo

Para cada chunk, o motor de busca encontra o **padrão mais similar** no codebook usando 3 estratégias em cascata:

```mermaid
graph LR
    CH[Chunk] --> H["1. Hash Exato<br>(O(1) lookup)"]
    H -->|Miss| L["2. LSH<br>(Locality Sensitive Hash)"]
    L -->|Miss| LN["3. Linear Scan<br>(Hamming Distance)"]
    H -->|Hit| R[CodebookID + Delta]
    L -->|Hit| R
    LN -->|Hit| R
```

| Fase | Complexidade | Descrição |
|:-----|:------------|:----------|
| Hash Exato | O(1) | Match perfeito — delta será zero |
| LSH | O(1) amortizado | Match aproximado via hash de localidade |
| Linear | O(n) | Busca exaustiva por menor distância Hamming |

### 3. Delta — XOR Bit-a-Bit

Após encontrar o padrão mais próximo, calculamos a **diferença (delta)** via operação XOR:

```
Chunk Original: 01101001 10110100 11001010
Padrão Codebook: 01101001 10110000 11001010
─────────────────────────────────────────
Delta (XOR):     00000000 00000100 00000000  ← Quase tudo zero!
```

> Quanto mais similar o chunk é ao padrão, **mais zeros no delta** → melhor compressão ZSTD.

### 4. Formato .crom (V2)

```mermaid
graph TD
    subgraph "Arquivo .crom"
        H["Header (32 bytes)<br>Version, OrigSize, Hash, ChunkCount"]
        BT["Block Table<br>Tamanho de cada bloco comprimido"]
        CT["Chunk Table<br>CodebookID + DeltaOffset por chunk"]
        DP["Delta Pool (ZSTD)<br>Dados XOR comprimidos"]
    end
    H --> BT --> CT --> DP
```

| Seção | Conteúdo | Tamanho |
|:------|:---------|:--------|
| Header | Versão, tamanho original, SHA-256, flags | 32 bytes fixos |
| Block Table | Tamanho comprimido de cada bloco de 16MB | 4 bytes × N blocos |
| Chunk Table | CodebookID (2B) + DeltaOffset (4B) por chunk | 6 bytes × N chunks |
| Delta Pool | Deltas XOR comprimidos com ZSTD | Variável |

---

## Fluxo de Descompressão

```mermaid
sequenceDiagram
    participant R as Reader
    participant CB as Codebook
    participant D as Delta Pool
    participant O as Output

    R->>R: Lê Header (.crom)
    R->>R: Lê Chunk Table
    
    loop Para cada chunk
        R->>CB: Busca padrão por CodebookID
        CB-->>R: Retorna padrão (128 bytes)
        R->>D: Lê delta comprimido
        D-->>R: Retorna delta XOR
        R->>R: Reconstrói: padrão XOR delta
        R->>O: Escreve bytes reconstruídos
    end
    
    R->>R: Verifica SHA-256
    R-->>O: ✅ Arquivo restaurado!
```

---

## CLI — Comandos Disponíveis

```bash
# Treinar codebook
crompressor train -i <pasta_dados> -o codebook.cromdb -s 16384

# Comprimir
crompressor pack -i arquivo.txt -o arquivo.crom -c codebook.cromdb

# Comprimir com CDC e chunk customizado
crompressor pack -i dados.csv -o dados.crom -c codebook.cromdb --cdc --chunk-size 64

# Descomprimir
crompressor unpack -i arquivo.crom -o restaurado.txt -c codebook.cromdb

# Verificar integridade
crompressor verify --original arquivo.txt --restored restaurado.txt

# Analisar arquivo .crom
crompressor info -i arquivo.crom -c codebook.cromdb
```
