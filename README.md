<p align="center">
  <h1 align="center">🧬 CROM</h1>
  <p align="center"><strong>Compressão de Realidade e Objetos Mapeados</strong></p>
  <p align="center"><em>O Compilador de Realidade — Reescrevendo as Regras da Compressão Digital</em></p>
</p>

---

## O Que é o CROM?

O CROM é um sistema de compressão **lossless** de nova geração, construído em **Go**, que opera sobre um paradigma fundamentalmente diferente dos compressores tradicionais.

Enquanto algoritmos como Gzip, Zstd e LZ4 tratam cada arquivo como uma sequência estatística de bytes sem contexto, o CROM trata cada arquivo como um **objeto reconhecível** dentro de um **Espaço Latente de Padrões** — um **Codebook Universal** de 50GB+.

> **Analogia:** O Gzip é um taquígrafo que inventa abreviações enquanto lê. O CROM é um **compilador** que já conhece todas as palavras do dicionário e apenas aponta para elas.

## O Conceito: "Compilador de Realidade"

O CROM não comprime — ele **compila**. Transforma dados brutos em um **mapa de referências** (IDs) que apontam para fragmentos já conhecidos no Codebook. A fidelidade bit-a-bit é garantida por uma **Camada de Refinamento (Delta Lossless)** que captura o resíduo exato entre o padrão encontrado e o dado original.

```
┌──────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Arquivo      │────▶│   crompressor-pack      │────▶│  .crom (mapa    │
│ Original     │     │  (Compilador)    │     │   de IDs +      │
│ (1TB)        │     │                  │     │   Resíduos)     │
└──────────────┘     └──────────────────┘     │  (~10GB)        │
                            │                 └─────────────────┘
                     ┌──────┴──────┐
                     │  Codebook   │
                     │  Universal  │
                     │  (50GB)     │
                     └─────────────┘
```

## Pilares do Projeto (Crompressor V16)

| Pilar | Descrição |
|---|---|
| 🎯 **Fidelidade Absoluta** | Compressão 100% lossless garantida por SHA-256 e resíduos exatos. |
| 🧠 **Codebook & Auto-Training** | Dicionários universais P2P com inferência de domínio e BPE instantâneo (Zero-Config). |
| ⚡ **Performance e Entropia** | Cache LSH O(1), Semantic Chunking e bypass de arquivos aleatórios via Shannon Entropy > 7.5. |
| 🛡️ **Tolerância a Expansão** | Smart Passthrough nativo: Se comprimir piorar o tamanho, converte zero-overhead. |
| 🔬 **Soberania e P2P Sync** | Daemon Kademlia DHT com Bitswap de blocos e validação criptográfica GossipSub (Ed25519). |

## Arquitetura Resumida

```
crompressor-pack (Compilador)                    crompressor-unpack (Decompilador)
┌─────────────────────┐                   ┌──────────────────────────┐
│ 1. Chunking         │                   │ 1. Leitura do .crom      │
│ 2. Busca HNSW       │                   │ 2. Lookup no Codebook    │
│ 3. Match de Padrão  │                   │ 3. Aplicação do Delta    │
│ 4. Cálculo de Delta │                   │ 4. Reconstrução          │
│ 5. Geração do .crom │                   │ 5. Validação SHA-256     │
└─────────────────────┘                   └──────────────────────────┘
```

## Quick Start (V16)

```bash
# Compilar (Modo Zero-Config com Auto-Training nativo)
crompressor pack -i ./meus_dados.json -o ./backup.crom

# Decompilar (restauração bit-a-bit)
crompressor unpack -i ./backup.crom -o ./restaurado.json

# Validar integridade
crompressor verify --original ./meus_dados.json --restored ./restaurado.json

# Modo Servidor P2P (Daemon GossipSub)
crompressor daemon --allow-hive-mind
```

## Documentação

| Documento | Descrição |
|---|---|
| [01 - Conceito e Visão](docs/01-CONCEITO_E_VISAO.md) | Compressão estatística vs. baseada em conhecimento |
| [02 - Arquitetura do Sistema](docs/02-ARQUITETURA_DO_SISTEMA.md) | Fluxo completo crompressor-pack / crompressor-unpack |
| [03 - Estrutura do Dicionário](docs/03-ESTRUTURA_DO_DICIONARIO.md) | Codebook de 50GB, mmap e indexação HNSW |
| [04 - Especificação do Compilador](docs/04-ESPECIFICACAO_DO_COMPILADOR.md) | Chunking, KNN e mapa de IDs |
| [05 - Camada de Refinamento](docs/05-CAMADA_DE_REFINAMENTO.md) | Delta Lossless e garantia bit-a-bit |
| [06 - Tech Stack](docs/06-TECH_STACK.md) | Go, CGO, HNSW e memória |
| [07 - Segurança e Soberania](docs/07-SEGURANCA_E_SOBERANIA.md) | Privacidade e soberania digital |
| [08 - Casos de Uso Avançados](docs/08-CASOS_DE_USO_AVANCADOS.md) | Clones paramétricos e compressão massiva |
| [09 - Benchmarks e Métricas](docs/09-BENCHMARKS_E_METRICAS.md) | Metas de performance vs. Zstd e Gzip |
| [10 - Estratégia MVP](docs/10-ESTRATEGIA_MVP.md) | Roadmap de 4 semanas e checklist técnico |

## Licença

Este projeto é **proprietário** e protegido sob os termos de soberania digital do CROM.

---

<p align="center">
  <em>"Não comprimimos dados. Compilamos realidade."</em>
</p>
