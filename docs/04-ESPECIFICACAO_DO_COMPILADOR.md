# 04 — Especificação do Compilador (`crompressor-pack`)

> *"O compilador não inventa. Ele traduz o arquivo para a linguagem do Codebook."*

---

## Responsabilidade do `crompressor-pack`

O `crompressor-pack` é o binário mais complexo do sistema CROM. Sua função é:

1. **Ler** o arquivo original
2. **Fragmentar** em chunks otimizados
3. **Buscar** o padrão mais próximo no Codebook para cada chunk
4. **Calcular** o resíduo (Delta) exato
5. **Serializar** tudo em um arquivo `.crom` compacto

---

## Etapa 1: Chunking Adaptativo

O chunking é a divisão do arquivo original em blocos menores que serão buscados no Codebook. A estratégia de chunking impacta diretamente a taxa de compressão.

### Estratégia de Seleção

```go
// ChunkStrategy determina como o arquivo é fragmentado.
type ChunkStrategy uint8

const (
    // FixedChunk: blocos de tamanho fixo — simples e previsível.
    FixedChunk ChunkStrategy = iota

    // RabinChunk: Content-Defined Chunking baseado em Rabin fingerprint.
    // Detecta fronteiras naturais no fluxo de dados.
    RabinChunk

    // TypeAwareChunk: chunking inteligente baseado no tipo de arquivo.
    // Ex: um PNG é dividido por linhas de pixels; um PDF por objetos.
    TypeAwareChunk
)
```

### Chunking Fixo (MVP)

Para o MVP, usaremos chunks de tamanho fixo:

```go
const DefaultChunkSize = 128 // bytes

// FixedChunker divide o arquivo em blocos de tamanho fixo.
func FixedChunker(data []byte, chunkSize int) []Chunk {
    chunks := make([]Chunk, 0, len(data)/chunkSize+1)
    
    for offset := 0; offset < len(data); offset += chunkSize {
        end := offset + chunkSize
        if end > len(data) {
            end = len(data) // Último chunk pode ser menor
        }
        
        chunks = append(chunks, Chunk{
            Data:   data[offset:end],
            Offset: uint64(offset),
            Size:   uint32(end - offset),
            Hash:   xxhash.Sum64(data[offset:end]),
        })
    }
    
    return chunks
}
```

### Content-Defined Chunking com Rabin (Futuro)

```go
// RabinChunker usa fingerprints de Rabin para encontrar fronteiras naturais.
// Vantagem: se um byte é inserido no meio do arquivo, apenas 1-2 chunks mudam.
// Isso maximiza hits no Codebook para arquivos editados.
func RabinChunker(data []byte, avgSize int) []Chunk {
    const (
        minSize = 32
        maxSize = 1024
        mask    = (1 << 13) - 1 // Avg chunk size ~8KB
    )
    
    var (
        chunks []Chunk
        start  int
        fp     uint64 // Rabin fingerprint
    )
    
    for i := 0; i < len(data); i++ {
        fp = rabinUpdate(fp, data[i])
        size := i - start
        
        // Fronteira: quando o fingerprint "casa" com a máscara
        // OU quando atingimos o tamanho máximo
        if (size >= minSize && fp&mask == 0) || size >= maxSize {
            chunks = append(chunks, Chunk{
                Data:   data[start : i+1],
                Offset: uint64(start),
                Size:   uint32(i + 1 - start),
                Hash:   xxhash.Sum64(data[start : i+1]),
            })
            start = i + 1
        }
    }
    
    // Último chunk (se houver dados restantes)
    if start < len(data) {
        chunks = append(chunks, Chunk{
            Data:   data[start:],
            Offset: uint64(start),
            Size:   uint32(len(data) - start),
            Hash:   xxhash.Sum64(data[start:]),
        })
    }
    
    return chunks
}
```

---

## Etapa 2: Embedding e Busca de Vizinhos Mais Próximos

Para cada chunk, precisamos encontrar o **codeword mais similar** no Codebook. Isso envolve dois passos:

### 2.1: Geração do Embedding (Vetor de Características)

O embedding transforma os bytes brutos do chunk em um vetor numérico que captura sua "essência" — permitindo comparação rápida entre chunks.

```go
// Embed transforma um chunk em um vetor float32 para busca no HNSW.
// Método: SimHash (Locality-Sensitive Hash)
// - Determinístico (mesmo input → mesmo output)
// - Preserva similaridade (chunks parecidos → embeddings próximos)
func Embed(chunk []byte, dim int) []float32 {
    embedding := make([]float32, dim)
    
    for i := 0; i < len(chunk); i++ {
        // Projeta cada byte em múltiplas dimensões usando hash
        for d := 0; d < dim; d++ {
            h := murmur3(uint64(chunk[i]), uint64(d+i))
            if h&1 == 0 {
                embedding[d] += 1.0
            } else {
                embedding[d] -= 1.0
            }
        }
    }
    
    // Normalizar o vetor (L2 normalization)
    norm := float32(0)
    for _, v := range embedding {
        norm += v * v
    }
    norm = float32(math.Sqrt(float64(norm)))
    for i := range embedding {
        embedding[i] /= norm
    }
    
    return embedding
}
```

### 2.2: Busca HNSW (K Nearest Neighbors)

```go
// FindBestMatch busca o codeword mais próximo no Codebook.
// Retorna o ID do codeword e a distância (quanto menor, melhor).
func (compiler *Compiler) FindBestMatch(chunk Chunk) (MatchResult, error) {
    // 1. Gerar embedding do chunk
    embedding := Embed(chunk.Data, compiler.codebook.EmbedDim())
    
    // 2. Buscar K=5 vizinhos mais próximos no HNSW
    results := compiler.index.Search(embedding, 5)
    
    // 3. Para cada candidato, calcular similaridade EXATA
    //    (HNSW é aproximado; aqui refinamos com comparação byte-a-byte)
    bestMatch := MatchResult{Distance: math.MaxFloat64}
    
    for _, candidate := range results {
        pattern := compiler.codebook.Lookup(candidate.ID)
        
        // Distância exata: Hamming distance (contagem de bits diferentes)
        dist := hammingDistance(chunk.Data, pattern)
        
        if dist < bestMatch.Distance {
            bestMatch = MatchResult{
                CodebookID: candidate.ID,
                Pattern:    pattern,
                Distance:   dist,
            }
        }
    }
    
    return bestMatch, nil
}
```

### Fluxo de Decisão por Chunk

```
┌─────────────────────────────────────────────────────┐
│                  POR CHUNK                           │
│                                                      │
│  Chunk [i]                                           │
│    │                                                 │
│    ├── Hash rápido (xxhash) ─► Cache local hit?      │
│    │   └── SIM → Reutilizar resultado anterior       │
│    │   └── NÃO → Continuar para busca HNSW           │
│    │                                                 │
│    ├── Embedding (SimHash) ─► Vetor float32[32]      │
│    │                                                 │
│    ├── HNSW Search ─► Top-5 candidatos               │
│    │                                                 │
│    ├── Refinamento Exato ─► Melhor match (Hamming)   │
│    │                                                 │
│    ├── Match Perfeito? (distância == 0)               │
│    │   └── SIM → Armazenar apenas o ID (Delta vazio) │
│    │   └── NÃO → Calcular Delta (XOR)                │
│    │                                                 │
│    └── Output: { codebook_id, delta, original_size } │
└─────────────────────────────────────────────────────┘
```

---

## Etapa 3: Pipeline de Compilação Paralelo

O compilador utiliza **goroutines** para processar múltiplos chunks em paralelo:

```go
func (compiler *Compiler) Compile(input string, output string) error {
    // 1. Ler arquivo original
    data, err := os.ReadFile(input)
    if err != nil {
        return err
    }
    originalHash := sha256.Sum256(data)
    
    // 2. Chunking
    chunks := FixedChunker(data, DefaultChunkSize)
    
    // 3. Processar chunks em paralelo (fan-out / fan-in)
    results := make([]ChunkResult, len(chunks))
    
    var wg sync.WaitGroup
    sem := make(chan struct{}, runtime.NumCPU()) // Limitar concorrência
    
    for i, chunk := range chunks {
        wg.Add(1)
        sem <- struct{}{}
        
        go func(idx int, c Chunk) {
            defer wg.Done()
            defer func() { <-sem }()
            
            // Buscar melhor match no Codebook
            match, _ := compiler.FindBestMatch(c)
            
            // Calcular Delta
            delta := XOR(c.Data, match.Pattern)
            
            results[idx] = ChunkResult{
                CodebookID:   match.CodebookID,
                Delta:        delta,
                OriginalSize: c.Size,
            }
        }(i, chunk)
    }
    
    wg.Wait()
    
    // 4. Serializar para .crom
    return compiler.writeCromFile(output, originalHash, results)
}
```

---

## Formato do Mapa de IDs (Chunk Table)

Cada entrada na Chunk Table do arquivo `.crom` tem a seguinte estrutura:

```go
// ChunkEntry representa uma entrada na tabela de chunks do .crom.
type ChunkEntry struct {
    CodebookID   uint64 // ID do codeword no Codebook (8 bytes)
    DeltaOffset  uint64 // Offset no Delta Pool (8 bytes)
    DeltaSize    uint32 // Tamanho do delta comprimido (4 bytes)
    OriginalSize uint32 // Tamanho original do chunk (4 bytes)
    Flags        uint8  // Flags: 0x01=perfect_match, 0x02=last_chunk
}

// Tamanho por entrada: 25 bytes
// Para 1GB de dados com chunks de 128 bytes:
//   ~8M chunks × 25 bytes = ~200MB de Chunk Table
//   (Comprimível para ~20-40MB)
```

### Otimização: Perfect Match Flag

Quando `Distance == 0` (o chunk é idêntico a um codeword do Codebook), o Delta é **vazio** e a flag `perfect_match` é setada. Isso elimina completamente o armazenamento do Delta para esse chunk.

```
Para datasets bem cobertos pelo Codebook:
- 60-90% dos chunks → perfect match → Delta = 0 bytes
- 10-40% dos chunks → Delta pequeno (maioria zeros) → comprime bem
```

---

## Estatísticas de Compilação

Após a compilação, o `crompressor-pack` exibe um relatório:

```
╔═══════════════════════════════════════════════════╗
║              CROM COMPILATION REPORT              ║
╠═══════════════════════════════════════════════════╣
║  Input:           dados.tar (1.2 GB)              ║
║  Output:          dados.crom (45 MB)              ║
║  Compression:     26.7:1                          ║
║                                                   ║
║  Chunks Total:    9,830,400                       ║
║  Perfect Matches: 7,372,800 (75%)                 ║
║  Partial Matches: 2,457,600 (25%)                 ║
║  Avg Delta Size:  12.3 bytes (de 128 originais)   ║
║                                                   ║
║  Codebook Used:   crom-universal-v1.cromdb        ║
║  Codebook Hash:   0xABCD...1234                   ║
║  Original SHA256: 0x9F8E...5D6C                   ║
║                                                   ║
║  Time:            47.3s (compile)                 ║
║  Speed:           25.4 MB/s                       ║
╚═══════════════════════════════════════════════════╝
```

---

> **Próximo:** [05 - Camada de Refinamento](05-CAMADA_DE_REFINAMENTO.md) — Como garantimos fidelidade lossless com o cálculo de Delta.
