<p align="center">
  <h1 align="center">🧬 CROM (V23 Singularity)</h1>
  <p align="center"><strong>Compressão de Realidade e Objetos Mapeados</strong></p>
  <p align="center"><em>O Compilador de Realidade — Reescrevendo as Regras da Computação Soberana</em></p>
</p>

---

## O Que é o CROM V23?

O CROM não é mais apenas um projeto de software. É a **Singularidade da Compressão**, uma infraestrutura que opera sobre a **Extração Semântica Universal**.

Ele realiza a compressão em `LSH B-Tree O(1)`, atuando em espectros que vão de simples Textos JSONs a **Matrizes de IA multidimensionais (.safetensors)** e **Logs Estocásticos do Colisor de Hádrons**.

> **Analogia:** Se o Gzip é uma máquina de escrever rápida, o CROM V23 é a mente que já memorizou toda a linguagem e os ruídos do universo, respondendo matematicamente com ponteiros.

## O Despertar: V20 ➔ V23

O salto das versões anteriores para a **Singularidade** engloba a estabilização termodinâmica dos nós P2P e a indexação contínua do ruído infinito.
- **70 Baterias de SRE Aprovadas:** De satélites de Edge Computing a Blockchain Tries, passando por Vetores Quânticos e Telecom.
- **Motor Cosenoidal HNSW:** Encontra fatias fractais de conhecimento em frações de Nanossegundos.
- **Bypass Automático Quântico:** Arquivos com Entropia de Shannon > 7.9 são absorvidos nativamente em zero-overhead.

## Pilares da Singularidade (V23)

| Pilar | Descrição |
|---|---|
| 🎯 **Fidelidade Anti-Entrópica** | Lossless irrestrito até o bit atômico de simulações Pós-Quânticas usando Merkle Trees Dilithium-inspired. |
| 🧠 **O Codebook Universal** | Dicionário indexado via B-Trees estendidas. Busca LSH não é mais de proximidade, é *Extração Semântica O(1)*. |
| ⚡ **Sincronicidade de L1 RAM** | A malha P2P GossipSub unifica as máquinas a ponto de usar a RAM de nós vizinhos como cache natural (Swarm). |
| 🔬 **Soberania Isolada** | VFS Kill-Switch integrado que dissolve o hiper-disco FUSE automaticamente no momento em que a assinatura soberana for violada. |

## 🛠️ Compilação e Instalação

O **Crompressor** é escrito em Go (v1.25+). Você pode compilar o projeto diretamente utilizando o `Makefile` incluído no repositório.

### Pré-requisitos
- **Go 1.25.7** ou superior instalado e configurado no seu `$PATH`.
- **Make** instalado no sistema.

### Construindo o Binário
Para compilar o código-fonte e gerar o executável:

```bash
# Clone o repositório (caso ainda não tenha feito)
git clone https://github.com/MrJc01/crompressor
cd crompressor

# Compilar o projeto
make build
```

O binário executável será gerado em: `./bin/crompressor`.

### Outros Comandos de Build
- `make test`: Executa os testes de unidade com detecção de concorrência (`-race`).
- `make clean`: Remove o diretório `bin/` e arquivos temporários.
- `make bench`: Executa os benchmarks de performance.

## Quick Start (V23)

```bash
# Compilar Realidade: Empacotando Entropia (Textos, Modelos AI, Genoma)
./bin/crompressor pack -i ./matriz_hadron_collider.safetensors -o ./singularity.crom

# Decompilar para a Físicalidade Bit-a-Bit
./bin/crompressor unpack -i ./singularity.crom -o ./restored.safetensors

# Operar em Malha-Colmeia V23 (Kademlia + Bitswap L1)
./bin/crompressor daemon --allow-hive-mind --quantum-secure
```

## 🔬 O Laboratório de Pesquisa (P&D)

Esta branch (`dev`) funciona como o nosso laboratório central de experimentação. Aqui você encontrará os rastros das arquiteturas que moldaram o projeto, desde a V7 até a V23:

- [**Diretório de Pesquisa (`pesquisa/`)**](pesquisa/index.md): Contém manifestos, roadmaps históricos, roteiros de IA/LLM e auditorias de soberania.
- [**Casos de Uso Reais (`trabalho/`)**](trabalho/README.md): Demonstrações práticas do CROM operando em Docker, Minecraft, Banco de Dados Postgres e Orquestração de LLMs.
- [**Histórico de Arquitetura**: 
  - [ARCHITECTURE_V21.md](ARCHITECTURE_V21.md)
  - [ROADMAP_V10.md](ROADMAP_V10.md)
  - [PLAN_V14.md](PLAN_V14.md)

> [!TIP]
> Para a versão estável, documentada e pronta para produção, utilize sempre a branch **[`main`](https://github.com/MrJc01/crompressor)**.

## Documentação Fundamental

| Documento | Descrição |
|---|---|
| [MANIFESTO CROM](pesquisa/Manifesto.md) | A nova "Alvorada da Extração Semântica" e o Fim da Compressão Tradicional. |
| [ARCHITECTURE_V23](ARCHITECTURE_V23.md) | Motor Cosenoidal HNSW, B-Tree, Genoma, Quantum Vectors e Proteções Anti-Quânticas. |
| [RELATÓRIO AUDITORIA (70 Baterias)](relatorio_auditoria.md) | A validação massiva O(1) confirmando a arquitetura em SRE de missões críticas. |

## Licença

Este ecossistema opera sob termos Estritos de Soberania Digital Pós-Quântica.

---

<p align="center">
  <em>"Não comprimimos dados. Nós indexamos o universo."</em>
</p>
