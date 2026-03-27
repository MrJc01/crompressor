# MANIFESTO CROM: Compressão de Realidade e Objetos Mapeados

> "Não comprimimos dados. Nós compilamos a realidade."

Bem-vindo ao CROM. Mais do que um projeto de compressão de software, o CROM é uma infraestrutura de soberania de dados ponta a ponta. Este manifesto foi concebido para guiar tanto os curiosos quanto os engenheiros de base através dos fundamentos do que estamos construindo.

---

## 1. A Visão: Por que o CROM existe?

Vivemos na era da cópia infinita. Cada vez que enviamos um arquivo, clonamos informações redundantes através dos oceanos, gastando energia, largura de banda e perdendo o controle arquitetural sobre nossos próprios dados.

O CROM nasceu para instaurar a **Soberania Digital**. Em vez de tratar a compressão como uma tarefa isolada de empacotar bytes e esquecê-los imediatamente após a descompressão, o CROM trata a compressão como *linguagem*.

Imagine a **Biblioteca de Babel** de Jorge Luis Borges — uma biblioteca contendo todos os livros possíveis. Em vez de carregar todos os livros do mundo nas costas, o CROM carrega apenas o *Dicionário* de todos os parágrafos possíveis (o **Codebook**). Quando criamos um arquivo `.crom`, não estamos guardando o livro; estamos guardando uma fina folha de papel com coordenadas exatas que dizem: *"Capítulo 1 está na estante 4, Capítulo 2 na estante 8"*. Sem a biblioteca (o Dicionário), o papel é inútil. Com a biblioteca, o universo inteiro é desdobrado instantaneamente.

---

## 2. Para Leigos: O CROM explicado com LEGO

Entender o CROM é fácil se você pensar em **LEGO**.

Imagine que você tem uma caixa gigantesca cheia de milhares de peças de LEGO variadas na sua casa. Esta caixa gigante é o nosso **Codebook** (Dicionário de Padrões).

Seu amigo constrói um castelo incrível e quer enviá-lo para você. Em vez de desmontar o castelo, colocar num pacote enorme e enviar pelo correio (como o ZIP tradicional faz), ele faz o seguinte:
1. Ele olha para as peças do castelo e percebe que você já tem 95% delas na sua caixa em casa.
2. Ele te envia apenas um manual pequenininho por email: *"Pegue a peça azul 4x4 e coloque aqui; pegue a peça vermelha e coloque ali."*
3. Se o castelo dele usar uma peça rara que estava arranhada (um dado único ou "sujo"), ele te envia a instrução padrão + um "adesivinho" com o arranhão (o **Delta**). 

O arquivo `.crom` resultante não é o castelo, é apenas o manual e os adesivinhos. 

### O Disco Virtual (VFS)
E quando você quiser ler esse arquivo enorme? É aqui que surge o **VFS** (Virtual Filesystem ou Sistema de Arquivos Virtual). É como ter uma biblioteca de 1 Terabyte guardada dentro de uma caixa de sapato. Você não precisa tirar e abrir todos os livros na mesa para ler um parágrafo. Você enfia a mão, puxa a página exata que quer ler e ela se materializa na sua mão instantaneamente. O computador pensa que o arquivo cru está lá, sem notar que é tudo uma mágica de projeção sob demanda.

---

## 3. Para Técnicos: Deep Dive na Engenharia

Para os engenheiros, o CROM não é magia, é matemática determinística e engenharia de sistemas rigorosa.

### Pipeline de Dados (Compilação)
1. **Chunking (128B)**: O dado de entrada é fatiado em blocos fixos de 128 bytes (modificáveis via rolling hash no futuro).
2. **LSH (Locality-Sensitive Hashing)**: O "Radar de Proximidade". Em vez de busca linear $O(N)$, usamos assinaturas digitais de similaridade. Padrões similares caem nos mesmos baldes, reduzindo a busca do espaço de 32.000 padrões potenciais para apenas um punhado de candidatos.
3. **XOR Distance**: Encontrado o melhor `CodebookID`, aplicamos um XOR bit-a-bit contra o chunk de entrada. O residual gerado (Delta) contém um mar de zeros, preservando apenas o "ruído de entropia".
4. **Zstd Compression**: Todos os deltas e resíduos são agrupados (Delta Pool) e devorados pelo Zstandard, explorando agressivamente a entropia artificialmente dizimada.

### Format V2: O Paradigma do Acesso Aleatório O(1)
Abondonamos o streaming linear v1. O verdadeiro poder do CROM está na sua **BlockTable**. Em V2, empacotamos os Chunks em "Blocos de 16MB". A `BlockTable` armazena os offsets no disco para cada bloco comprimido.
Assim, ao receber um `ReadAt(offset)`, a matemática diz em qual bloco o offset está. Carregamos do disco, passamos no AES-GCM, descomprimimos via Zstd e armazenamos em um cache LRU. A partir daí, fazemos o XOR lookup e entregamos os bytes exatos em **Latências O(1)** (P50 resolvido na casa de 140µs). 

### Busca Variacional (Fuzziness) no Espaço Latente
A grande sacada é que **não procuramos matches exatos**. Pela Natureza da Teoria da Informação, procuramos no espaço latente de padrões o vetor de "menor distância de Hamming". O CROM acha a representação mais aproximada, e corrige o delta via XOR. Isso transforma a entropia aleatória da humanidade num dataset contínuo que encolhe a cada passo.

> [!NOTE] 
> **Nota Técnica de Segurança (AES-256-GCM / PBKDF2)**
> Antes de o Zstd despejar os blocos no disco, caso a flag de encriptação esteja ativa, todo o pool de deltas passa por um envelope **AES-256-GCM**. A chave de bloco simétrica nunca é exposta; ela é derivada do seu password via **PBKDF2-HMAC-SHA256** combinada com o Salt hiper-seguro de 32-bytes randomizados no cabeçalho do arquivo `.crom`.

---

## 4. O Diferencial: Por que não usar ZIP ou GZIP?

Não viemos substituir algoritmos genéricos. Viemos reescrever a topologia da persistência de dados. O CROM é um ecossistema infraestrutural com propriedades de rede.

| Característica | CROM | ZIP / GZIP / Zstd Bruto |
|---|---|---|
| **Persistência de Conhecimento** | O Dicionário (Codebook) **aprende** e fica estático na máquina local. Funciona como modelo fundacional contínuo para qualquer conjunto de dados futuro. | O dicionário morre ao final de cada arquivo. Repetitivo, isolado e míope. |
| **Segurança / "Kill-Switch"** | Soberania de Arquivo: O `.crom` não é decodificável sem o Codebook autorizado e a chave matriz. Se a máquina excluir o Codebook, a VM, o VFS e todos os arquivos auto-evaporam permanentemente. | Isolamento tradicional. O container de arquivos expõe os dados brutos ou a proteção da senha. |
| **Performance Ponto a Ponto** | Operações `mmap` e paginação controlada pelo Kernel. Projeta um volume de 50GB em milissegundos transparentes (FUSE). | Exige cópia em dobro na memória, travando disco e desempacotando byte por byte até o fim linearmente. |

---

## 5. Casos de Uso Reais

Onde o CROM deixa de ser teoria e passa a dominar a camada de abstração? 

### Cenário A: Sincronização P2P de Backups Massivos
Duas corporações (Nó A e Nó B) compartilham o mesmo Codebook de Treinamento. Uma atualização de banco de dados de 50GB ocorreu. Em vez de transmitir 50GB — ou rodar rsync que onera I/O —, o Nó A gera um **ChunkManifest**. Os nós comparam binariamente o Diff de hash em frações de milissegundo e transferem exclusivamente as chaves-delta.

### Cenário B: Armazenamento Soberano e Células Fantasmas
Você carrega informações táticas em um notebook numa operação. O CROM VFS (Virtual Filesystem) está ativo e exibindo 1TB de HD em pastas navegáveis. Uma ameaça é detectada: uma request ou comando retira o `trained.cromdb` (O Codebook) do sistema.
O *Sovereignty Watcher* capta a remoção em tempo real. Os pontos de montagem do FUSE são explodidos, o Cache LRU e os buffers são esvaziados à força da RAM, e os arquivos originais tornam-se ininteligíveis. A "morte" pela separação da linguagem.

### Cenário C: Criação de Máquinas e "Arquivos Clone"
Desenvolvedores sêniores que disparam 50 instâncias idênticas para microsserviços. Os ambientes não são cópias completas: são apenas Manifestos de Chunk CROM consumindo 4Kb no disco de estado. As dependências OS rodam instantaneamente sobre o FUSE local que cruza com o Dicionário mmap global do servidor. Custo de I/O zero, custo de storage massivamente deflacionado.

---

## 6. Roadmap e Futuro: O Enxame de Dados

A Fase 7 já é uma realidade: O **CROM Network Layer** fundiu nossa estrutura ao `libp2p`. 
O binário converte máquinas num **Enxame de Dados (Data Swarm)**. Descobrimos nós em rede WAN via *Kademlia DHT* e pareamos hosts LAN por *mDNS*. 

A autenticação é selada pelo `BuildHash` do Codebook. Dois nós com percepções da realidade (Codebooks) diferentes não "falam a mesma língua" e são rejeitados no handshake pelo Noise TCP/QUIC. No futuro, implementaremos:
- Descompressão Paralela Distribuída no núcleo P2P.
- Roteamento Bitswap inter-níveis (Swarm Routing).
- Rolling Hashes para fronteiras elásticas de chunking.

O CROM não reduz o que você cria para caber numa caixa menor. Ele reprograma a infraestrutura local para que seus dados sejam reconstruídos pelo tecido do Sistema.

Bem-vindos ao salto da **Compilação de Realidade**.
