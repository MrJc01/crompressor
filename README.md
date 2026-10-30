# Crompressor

> Quem tem o mesmo cérebro não precisa receber o arquivo: pela rede passa só o que monta ele. Escrito em Go.

[![Build](https://github.com/MrJc01/crompressor/actions/workflows/ci.yml/badge.svg)](https://github.com/MrJc01/crompressor/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/MrJc01/crompressor)](https://goreportcard.com/report/github.com/MrJc01/crompressor)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## O que é o Crompressor?

Dois ou mais usuários instalam o mesmo **cérebro** (`.cromdb`). Depois disso, para enviar um arquivo que o cérebro já conhece, basta passar pela rede o que monta esse arquivo, e não o arquivo inteiro. O cérebro funciona como uma biblioteca compartilhada dos dados.

Serve para coleções e versões de dados parecidos. **Não** substitui o zstd para comprimir um arquivo isolado, e o cérebro costuma ter quase o tamanho dos dados que aprendeu: o ganho vem de reaproveitá-lo em muitas transferências.

| Caso (cérebro nos dois lados) | Original | Pela rede | Redução |
|---|---|---|---|
| Projeto que o cérebro já conhece | 27,55 MB | 0,0066 MB | 99,98% |
| Versão real mais nova do projeto | 27,56 MB | 0,0152 MB | 99,94% |
| 1 GB aleatório com 4 bytes inseridos | 1.000 MB | 0,88 MB | 99,91% |
| Logs novos do mesmo formato | 7,14 MB | 1,19 MB | 83,33% |

Um mesmo cérebro serve a vários projetos que ele já viu (1,38 GB de `node_modules` passaram como 0,85 MB). Em um projeto que ele nunca viu, o resultado fica perto do zstd puro (cerca de 85%).

Veja todos os números e como reproduzi-los em [Benchmarks](docs/benchmarks.md).

---

## Instalação

```bash
git clone https://github.com/MrJc01/crompressor
cd crompressor
make build
# Binário em ./bin/crompressor
```

---

## Quick Start

```bash
# Comprimir um arquivo sozinho: restaura sem cérebro
crompressor pack -i dados.bin -o dados.crom
crompressor unpack -i dados.crom -o restaurado.bin

# Com um cérebro treinado
crompressor train -i ./dados_de_exemplo/ -o cerebro.cromdb -k 4096 -s 40000
crompressor pack -i dados.bin -o dados.crom -c cerebro.cromdb
crompressor unpack -i dados.crom -o restaurado.bin -c cerebro.cromdb

# Conferir e inspecionar
crompressor verify --original dados.bin --restored restaurado.bin
crompressor info -i dados.crom
```

Opções do `pack`:

- `--max-compression`: zstd nível 19, mais lento e menor. Exige um binário com CGO (o padrão do `make build`, se houver compilador C); restaurar não exige.
- `--embed-brain`: leva o cérebro dentro do `.crom`, para quem ainda não o tem.
- `--encrypt <senha>`: AES-256-GCM.
- `--mode edge`: **com perda**. Guarda só as referências, e o restaurado não será igual ao original. Vídeo, binários e documentos ficam corrompidos.

---

## Transferência com cérebro compartilhado

O caso principal do projeto: os dois lados têm o mesmo `.cromdb` e só o que falta passa pela rede.
Não é um substituto do zstd para comprimir um arquivo isolado.

```bash
# Os dois lados: mesmo cérebro, com blocos de 4 KB
crompressor train -i ./dados/ -o cerebro.cromdb -k 4096 -s 40000

# Quem envia
crompressor pack -i projeto.tar -o ./crom-data/projeto.crom -c cerebro.cromdb
crompressor daemon -c cerebro.cromdb -d ./crom-data

# Quem recebe (o endereço aparece na saída do daemon)
crompressor pull --peer /ip4/IP/tcp/4001/p2p/ID -f projeto.crom -c cerebro.cromdb -d ./crom-data
# Ou, sem saber o endereço: --bootstrap <endereço de um nó> --private-dht (também no daemon)
crompressor unpack -i ./crom-data/projeto.crom -o projeto.tar -c cerebro.cromdb
```

Só recebe arquivos quem prova ter o mesmo cérebro, por desafio e resposta. O cérebro não é enviado pela rede: copie-o por um canal seu. Quem está atrás de NAT precisa de uma porta aberta, porque ainda não há relay.

Veja os resultados em [Benchmarks](docs/benchmarks.md), que separam dados já vistos pelo cérebro de dados novos.

---

## Como funciona

1. O `train` guarda blocos do corpus no cérebro (por exemplo, 4 KB cada).
2. O `pack` procura esses blocos em qualquer posição do arquivo, com hash rolante, então uma inserção não desalinha o resto.
3. Um bloco idêntico vira só o ID dele. O resto é comprimido com zstd usando o cérebro como dicionário, o que também aproveita dados apenas parecidos.
4. O `unpack` remonta o arquivo e confere o SHA-256. Se não bater, nenhum arquivo é gravado (exceto no modo edge ou com `--strict=false`).

Detalhes em [Arquitetura](docs/architecture.md).

---

## Maturidade

| Recurso | Situação |
|---|---|
| `train`, `pack`, `unpack`, `verify`, `info` | Testado |
| `daemon`, `pull` | Testado entre máquinas, com NAT e DHT; ainda sem auditoria externa de segurança |
| `mount` | Testado com FUSE |
| `grep` | Parcial: só vê texto inteiro dentro de uma entrada do cérebro |
| `cromfs` | Experimental: grava, mas a montagem não lista os arquivos |
| `llm-vfs` | Demonstração com arquivos fictícios |

Arquivos `.crom` novos não abrem em binários antigos, e nós de versões diferentes não sincronizam. Limites conhecidos em [Pendências](docs/PENDENCIAS.md).

---

## Estrutura do Repositório

```
crompressor/
├── cmd/crompressor/        ← CLI (pack, unpack, train, verify, info, grep, benchmark)
├── pkg/
│   ├── cromlib/            ← Motor de compressão (Pack, Unpack, AutoPack)
│   ├── format/             ← Serialização do formato .crom
│   └── cromdb/             ← Leitura de codebooks
├── internal/               ← Implementação interna (chunker, codebook, delta, entropy, etc.)
├── examples/               ← Exemplos Go (edge_mode, archive_mode, codebook_train)
├── docs/                   ← Documentação técnica (architecture, modes, benchmarks)
├── testdata/               ← Fixtures para testes
├── deployments/k8s/        ← Manifests Kubernetes
├── monitoring/             ← Prometheus + Grafana
├── Makefile
├── LICENSE                 ← MIT
└── README.md
```

---

## Makefile

```bash
make build      # Compila ./bin/crompressor
make test       # Roda todos os testes com -race
make bench      # Roda benchmarks de performance
make lint       # go vet
make clean      # Remove ./bin/
make demo       # Round-trip completo: gerar → comprimir → descomprimir → verificar
```

---

## Documentação

| Documento | Descrição |
|-----------|-----------|
| [Arquitetura](docs/architecture.md) | Como o motor funciona |
| [Modos de Operação](docs/modes.md) | Edge vs Archive: quando usar cada um |
| [Pendências](docs/PENDENCIAS.md) | O que ainda falta |
| [Benchmarks](docs/benchmarks.md) | Metodologia e resultados |
| [Pesquisa](docs/research/) | Links para estudo matemático |

---

## Ecossistema

Os outros repositórios `crompressor-*` (como [gui](https://github.com/MrJc01/crompressor-gui), [matematica](https://github.com/MrJc01/crompressor-matematica) e [video](https://github.com/MrJc01/crompressor-video)) são experimentos e não foram testados com o formato atual.

---

## Licença

MIT — veja [LICENSE](LICENSE).
