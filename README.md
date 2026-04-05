<p align="center">
  <h1 align="center">🧬 Crompressor</h1>
  <p align="center"><strong>Motor de Compressão Semântica em Go</strong></p>
  <p align="center">
    <a href="https://pkg.go.dev/github.com/MrJc01/crompressor"><img src="https://pkg.go.dev/badge/github.com/MrJc01/crompressor.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/MrJc01/crompressor"><img src="https://goreportcard.com/badge/github.com/MrJc01/crompressor" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
  </p>
</p>

<p align="center">
  🇧🇷 <strong>Português</strong> | <a href="README_en.md">🇺🇸 Read in English</a>
</p>

---

**Crompressor** é uma biblioteca de compressão *lossless* de alta performance escrita em Go. Ele combina extração semântica via indexação LSH em B-Tree, busca por similaridade de cosseno (HNSW) e um *codebook* treinável para alcançar proporções de compressão agressivas — especialmente em dados estruturados como código-fonte, logs, JSON, arquivos de configuração e tensores de modelos de IA.

## Recursos

- 🧠 **Codebook Treinável** — Crie dicionários específicos de domínio com seus próprios dados
- ⚡ **Busca LSH B-Tree O(1)** — Correspondência de padrões em tempo quase constante
- 🔒 **Integridade Lossless** — Reconstrução perfeita bit a bit com verificação via árvore de Merkle
- 📦 **Formato `.crom`** — Formato binário compacto, transmissível por streaming, com metadados integrados
- 🖥️ **Suporte VFS / FUSE** — Monte arquivos comprimidos como sistemas de arquivos virtuais
- 🌐 **Sincronização P2P** — Malha Kademlia/LibP2P para compartilhamento distribuído de codebooks
- 🔐 **Criptografia Pós-Quântica** — ChaCha20-Poly1305 + assinaturas inspiradas em Dilithium
- 🏗️ **Build WASM** — Execute o compressor no navegador

## Instalação

```bash
go get github.com/MrJc01/crompressor@latest
```

### Compilando a partir do código

```bash
git clone https://github.com/MrJc01/crompressor.git
cd crompressor
make build
```

O executável será gerado em `./bin/crompressor`.

**Requisitos:** Go 1.22+ e Make.

## Início Rápido

### Entendendo a Arquitetura (Os Arquivos)

Antes de rodar os comandos, é crucial entender o que cada arquivo representa no ecossistema do Crompressor:

- `data.bin` ou `./meus-dados/`: Seus **dados originais** (JSON, logs, código, etc.).
- `codebook.cromdb`: O **Dicionário Semântico (Cérebro)**. Gerado durante a etapa de `train`, ele guarda os padrões extraídos dos seus dados. Sem ele, é impossível descompactar o arquivo. Pode ser reutilizado em milhares de arquivos semelhantes.
- `data.crom`: O arquivo **Comprimido Final**. Ele contém apenas as "coordenadas" (deltas) que apontam para os padrões guardados no Codebook.

### Uso da CLI

```bash
# Treine um codebook a partir dos seus dados
./bin/crompressor train --input ./meus-dados/ --output codebook.cromdb --size 8192

# Comprima um arquivo
./bin/crompressor pack --input data.bin --output data.crom --codebook codebook.cromdb

# Descomprima
./bin/crompressor unpack --input data.crom --output restored.bin --codebook codebook.cromdb

# Verifique a integridade (perfeição bit-a-bit)
./bin/crompressor verify --original data.bin --restored restored.bin
```

### API Go

```go
package main

import (
    "fmt"
    "github.com/MrJc01/crompressor/pkg/sdk"
)

func main() {
    c := sdk.NewCompressor()

    // Comprimir (Pack)
    err := c.Pack("input.bin", "output.crom", "codebook.cromdb")
    if err != nil {
        panic(err)
    }

    // Descomprimir (Unpack)
    err = c.Unpack("output.crom", "restored.bin", "codebook.cromdb")
    if err != nil {
        panic(err)
    }

    fmt.Println("Concluído — compressão lossless verificada.")
}
```

## Estrutura do Projeto

```
crompressor/
├── cmd/crompressor/     # Binário da CLI (pack, unpack, verify, train, daemon)
├── pkg/                 # API Pública
│   ├── cromdb/          # Motor de banco de dados do Codebook
│   ├── cromlib/         # Compilador e descompactador principal
│   ├── format/          # Formato binário .crom (leitura/escrita)
│   ├── sdk/             # SDK de alto nível (cofre, compressor, crypto)
│   ├── sync/            # Sincronização baseada em manifestos
│   └── wasm/            # Ponto de entrada do WebAssembly
├── internal/            # Pacotes internos
│   ├── chunker/         # Chunking definido por conteúdo (CDC)
│   ├── codebook/        # Construtor de Codebook e indexação LSH
│   ├── entropy/         # Análise de entropia de Shannon e bypass
│   ├── fractal/         # Gerador de padrões fractais
│   ├── merkle/          # Integridade via árvore de Merkle
│   ├── search/          # Motor de similaridade de cosseno HNSW
│   ├── vfs/             # Sistema de arquivos virtual e montagem FUSE
│   └── ...              # delta, crypto, metrics, network, etc.
├── docs/                # Documentação técnica (10 capítulos)
├── examples/            # Exemplos de uso
├── scripts/             # Scripts auxiliares
├── go.mod
└── LICENSE              # MIT
```

## Comandos Make

| Comando | Descrição |
|---|---|
| `make build` | Compila o executável da CLI para `./bin/crompressor` |
| `make test` | Roda todos os testes com detecção de *race conditions* |
| `make bench` | Executa benchmarks |
| `make lint` | Executa o `go vet` |
| `make clean` | Remove arquivos gerados pelo build |

## Documentação

Documentação técnica detalhada está disponível na pasta [`docs/`](docs/) (em inglês):

1. [Concept & Vision](docs/01-CONCEITO_E_VISAO.md)
2. [System Architecture](docs/02-ARQUITETURA_DO_SISTEMA.md)
3. [Dictionary Structure](docs/03-ESTRUTURA_DO_DICIONARIO.md)
4. [Compiler Specification](docs/04-ESPECIFICACAO_DO_COMPILADOR.md)
5. [Refinement Layer](docs/05-CAMADA_DE_REFINAMENTO.md)
6. [Tech Stack](docs/06-TECH_STACK.md)
7. [Security & Sovereignty](docs/07-SEGURANCA_E_SOBERANIA.md)
8. [Advanced Use Cases](docs/08-CASOS_DE_USO_AVANCADOS.md)
9. [Benchmarks & Metrics](docs/09-BENCHMARKS_E_METRICAS.md)
10. [MVP Strategy](docs/10-ESTRATEGIA_MVP.md)

## Resultados de Benchmark

Resultados reais da suíte de testes automatizada (branch [`benchmark`](https://github.com/MrJc01/crompressor/tree/benchmark)). Comando: `git checkout benchmark && go run ./benchmark/`

### Taxa de Compressão (Ratio)

| Dataset | Tipo | Original | Packed | Ratio | Pack | Unpack | SHA-256 |
|---|---|---|---|---|---|---|---|
| go_source | Código Go repetitivo | 10 MB | 2.2 MB | **4.62x** | 7.3 MB/s | 12.8 MB/s | ✅ |
| json_api | JSON estruturado | 10 MB | 3.2 MB | **3.14x** | 3.2 MB/s | 13.7 MB/s | ✅ |
| binary_headers | Headers ELF + padding | 10 MB | 2.4 MB | **4.25x** | 4.3 MB/s | 31.1 MB/s | ✅ |
| mixed_config | Configurações YAML/TOML | 5 MB | 1.3 MB | **3.87x** | 6.9 MB/s | 16.9 MB/s | ✅ |
| server_logs | Linhas de log de servidor | 10 MB | 3.4 MB | **2.91x** | 3.9 MB/s | 16.8 MB/s | ✅ |
| high_entropy | Pseudoaleatório (pior caso) | 10 MB | 10 MB | 1.00x | 64 MB/s | 57 MB/s | ✅ |

### Escalabilidade (1MB → 500MB)

| Tamanho | Ratio | Vel. Pack | Vel. Unpack |
|---|---|---|---|
| 1 MB | **2.81x** | 3.0 MB/s | 13.4 MB/s |
| 10 MB | **2.91x** | 6.6 MB/s | 33.3 MB/s |
| 100 MB | **2.92x** | 9.0 MB/s | 37.2 MB/s |
| 500 MB | **2.93x** | 7.6 MB/s | 31.9 MB/s |

### Comparativo de Chunkers (melhor por dataset)

| Dataset | Fixed-128B | FastCDC | ACAC |
|---|---|---|---|
| json_api | 3.13x | **4.10x** 🏆 | 1.74x |
| server_logs | 2.88x | **3.66x** 🏆 | 2.50x |
| go_source | **4.60x** 🏆 | 4.30x | 1.86x |

### Montagem FUSE (VFS)

| Métrica | Valor |
|---|---|
| Leitura Sequencial VFS | **84.5 MB/s** |
| Leitura Direta no Disco | 319.4 MB/s |
| Latência do Primeiro Byte | 197ms |
| Integridade via VFS | ✅ SHA-256 MATCH |

### Docker FUSE Cascade

✅ **SUCESSO** — O Docker compilou e executou um contêiner lendo de uma cascata FUSE de 3 camadas:
`.crom` → Montagem CROM VFS → OverlayFS → `docker build` → `docker run`

> **Garantia Lossless (sem perda):** Todos os testes passam pela verificação SHA-256. Dados de alta entropia (aleatórios) são detectados automaticamente e passam direto (bypass) sem expansão.

### 🔄 Comparativo com Ferramentas de Mercado

O Crompressor é um compilador semântico baseado em dicionário. Ele **não** foi desenhado para superar o `gzip` ou `zstd` na compressão bruta de bytes em arquivos aleatórios, mas sim para possibilitar streaming VFS tipo *zero-copy* para dados altamente estruturados.

| Dataset | Crompressor | gzip -9 | zstd -19 | Melhor Ratio Bruto |
|---|---|---|---|---|
| go_source | 4.62x | 38.26x | 69.09x | zstd |
| json_api | 3.14x | 8.09x | 9.96x | zstd |
| binary_headers | 4.25x | 24.70x | 29.04x | zstd |
| server_logs | 2.91x | 6.48x | 8.45x | zstd |
| high_entropy | 1.00x (Bypass) | 1.00x | 1.00x | 🏆 CROM |

*Nota: `gzip` e `zstd` conseguem *ratios* muito maiores porque usam codificação de entropia generalizada (LZ77, FSE) escritos em C otimizado. O Crompressor sacrifica compressão bruta extrema para permitir leitura randômica em tempo O(1) (montagem via FUSE), sem precisar carregar todo o arquivo para a memória.*

## 🎯 Casos de Uso

### ✅ Onde o Crompressor Brilha
- **Sistema de Arquivos para Logs ou DB:** Quando você quer um banco de dados PostgreSQL ou logs JSON super compactados, mas que continuem ativamente legíveis em tempo real.
- **Execução Out-of-Core:** Rodar softwares pesados (como camadas Docker, Imagens de VM, ou mods de Minecraft) diretamente montados via FUSE num disco apertado.
- **Sincronização P2P:** A arquitetura de Codebook-Delta permite dedobrar semanticamente os dados de forma massiva entre nós de uma rede distribuída (Mesh/GossipSub).
- **Dados Estruturados:** APIs JSON, XML, YAML, CSV e repositórios massivos e repetitivos em Go/Python.

### ❌ Quando NÃO usar o Crompressor
- **Dados Pré-comprimidos:** Imagens (JPG, PNG), Vídeos (MP4) ou Arquivos (.zip, .tar.gz). O escudo de entropia detecta que são caóticos e apenas faz o "bypass" para poupar CPU — ou seja, você não ganha espaço extra.
- **Armazenamento Morto (Cold Storage):** Se você quer apenas fechar uma pasta num arquivo para becape e não liga para consultar aquilo instantaneamente depois (VFS), o `zstd -19` ou `7z` reduzem muito mais o arquivo.
- **Arquivos Genéricos Pequenos:** Se estiver comprimindo apenas 50KB de texto corrido, fazer um .zip é mais jogo do que gastar tempo treinando um Codebook de Machine Learning.

## ⚙️ Como Funciona (Debaixo do Capô)

1. **Train:** Cria um dicionário semântico (Codebook) a partir do seu dataset base, usando BPE Neural ou Locality-Sensitive Hashing.
2. **Pack:** Divide os seus arquivos de entrada em blocos baseados no contexto (usando FastCDC ou Delimitadores Semânticos), busca o bloco mais parecido (Cosine Similarity) no seu codebook previamente salvo, e grava apenas a diferença bruta (XOR Delta).
3. **Mount:** Monta o arquivo criptografado `.crom` e o seu dicionário respectivo `.cromdb` enraizado de volta dentro do kernel Linux (FUSE).
4. **Unpack (On-the-fly):** Quando o Docker, um Player de Mídia ou qualquer outra aplicação chama um byte, o seu motor reconstrói APENAS aquele pedaço exigido em microssegundos sem precisar descongelar um arquivo de 10GB na memória.

## Branches

| Branch | Objetivo | Comando |
|---|---|---|
| [`main`](https://github.com/MrJc01/crompressor) | Biblioteca pública — limpa, documentada, compatível com `go get` | `git checkout main` |
| [`dev`](https://github.com/MrJc01/crompressor/tree/dev) | Laboratório de P&D — CROM-IA, testes de SRE, experimentos malucos e UI. | `git checkout dev` |
| [`benchmark`](https://github.com/MrJc01/crompressor/tree/benchmark) | Suíte de benchmark com relatórios completos reais. | `git checkout benchmark` |

## Como Contribuir

Contribuições são bem-vindas! Crie um *issue* ou mande um *Pull Request*.

O grosso do desenvolvimento sempre acontece na branch [`dev`](https://github.com/MrJc01/crompressor/tree/dev) primeiro.

## Licença

[MIT](LICENSE) © 2026 MrJc01

---

<p align="center">
  <em>"Nós não comprimimos dados. Nós indexamos o universo."</em>
</p>
