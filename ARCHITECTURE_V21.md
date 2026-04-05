# 🌌 ARQUITETURA V21: O MOTOR UNIVERSAL E A FRONTEIRA EXABYTE

## 1. A Ideia Primordial: O "Compilador de Realidade"
O Crompressor nunca foi criado para ser um mero análogo ao Gzip, Zstd ou XZ. Compressores tradicionais enxergam apenas bytes estatísticos isolados em um único arquivo. 

A fundação do Crompressor é atuar como um **Motor Mapeador da Realidade (Semantic Reality Compiler)**. Ele entende que a humanidade produz dados que são incrivelmente redundantes em sua essência primordial (bases de DNA, textos legais corporativos, requisições de rede diárias). O motor abstrai padrões repetitivos colossais da humanidade, armazenando-os em um **Codebook Universal**.
- **O Objetivo V21:** Sincronizar esse Codebook vivo entre dispositivos heterogêneos ao redor do globo sem interrupções – do maior Datacenter Cloud ao microlaboratório montado em um CubeSat ou celular Android no subsolo. O Crompressor transforma Petabytes de dados em "ponteiros", e envia apenas os ponteiros pela rede.

---

## 2. A Filosofia "Edge-Anywhere": Perfis Adaptativos (Adaptive Execution)
Para que o Motor rode de forma eficiente em ambientes cujas regras físicas divergem radicalmente (bateria limitadíssima vs. rede instável vs. I/O ilimitado), o Crompressor V21 descarta a ideia de ter versões separadas. O mesmo código puramente estático em Go muda de forma durante a inicialização (`boot`).

### 🪐 2.1 Perfil IoT / Aeroespacial (Satélites & RPi Zero)
*Ameaças: Radiação (Bit Flips), Baixíssima Banda, SD Card Wear.*
- **Processamento:** Travado em Single-Thread (1 Goroutine Master) garantindo previsibilidade.
- **Armazenamento:** Mmap desativado. Leitura/Escrita perfeitamente sequencial para não fritar o controlador de armazenamento do MicroSD/NAND.
- **Rede P2P:** Agressivo Forward Error Correction (FEC). Polinômios de Reed-Solomon protegem cada pedaço da comunicação P2P, garantindo que mesmo se o sinal de rádio corromper os pacotes 50% do tempo, o Codebook inteiro será recuperado sem retransmissão de TCP.

### 📱 2.2 Perfil Mobile (Android / Smartphones)
*Ameaças: Bateria, Thermal Throttling (Aquecimento Súbito).*
- **Processamento:** Multi-Thread contido (2 a 4 Threads max). Se o host esquentar, a DHT vai "dormir".
- **Memória:** Radioactive Decay (G.C.) rodando furiosamente. Chunks do Codebook na RAM decaem em minutos se não forem buscados. Limite do Cache estrito (ex: 64MB).
- **Transporte DHT:** Limite duro de max 15 Peers Kademlia simultâneos. Corta requisições não solicitadas violentamente.

### 🏢 2.3 Perfil Enterprise / Cloud (Servidores & Bare-Metal)
*Ameaças: Latência O(N) linear no Codebook, Gargalos de Socket.*
- **Processamento:** GPU HNSW Offload. Busca semântica acelerada pelos milhares de Cuda Cores presentes nas VRAMs do Datacenter (OpenCL/CUDA) // ou SIMD AVX512 na CPU.
- **Memória:** Cache e Mmap ilimitados cruzando NVMe IOPS.
- **Rede P2P:** Atua como **Master Router** do Swarm. Interliga com 500+ nós, absorve as requisições Mobile e Espaciais pesadas resolvendo os grafos HNSW no servidor.

### 🧠 2.4 Perfil Shared-Core (Daemon Multi-App IPC)
*Ameaças: Desperdício de L1/L2, Race Conditions e Replicação Inútil de Memória no SO.*
- Se múltiplos aplicativos no mesmo dispositivo (ex: Celular Android ou Servidor Linux) utilizarem o Crompressor, o Motor **NÃO** deve ser instanciado N vezes. Ele operará como um **Serviço de Sistema (Daemon)**.
- **Comunicação por IPC/UDS:** Os aplicativos clientes enviam "Bytes Brutos" via Unix Domain Sockets ou gRPC leve e recebem "Ponteiros CROM".
- **Economia Exponencial do Cérebro:** O Dicionário (ex: 500MB) fica alocado apenas **uma única vez na RAM central do Daemon**. Se 15 aplicativos requisitarem compressão simultaneamente, o custo de memória extra do Codebook para o Sistema Operacional será rigorosamente zero.

---

## 3. Checklist de Integração da Arquitetura V21 (Status Executivo)

Aqui está a planilha mestra com a validação oficial do que já existe, e o que precisa ser codificado para as ondas posteriores. Se bater com a Checklist, o SRE aprova. Se não bater, código reprovado.

### ONDA 1: O Engine Biológico ✅ [CONCLUÍDO]
- [X] **Pesquisa 18 (Swarm Federated Learning):** Implementado `ProposeChunkMsg` e `gossip.go` parsers. Os nós transmitem conhecimento (chunks) em rede distribuída descentralizada (Zero-Trust).
- [X] **Pesquisa 20 (Codebook Radioactive Decay):** Implementado `DecayEngine`. O código executa limpeza LFU da Memória Cache (OOM Defense SRE). Compatível com qualquer Perfil (Basta ajustar o `decayWindow` no init).

### ONDA 2: Resiliência Multi-Plataforma ⏳ [PRÓXIMA ONDA]
- [ ] **Adaptive Profiles (`autobrain`):** Scanner nativo da VM Go (`runtime.NumCPU()`) que elege as limitações e limites da engine (IoT, Mobile, Server) no start do binário crompressor.
- [ ] **Pesquisa 26 (Forward Error Correction - FEC P2P):** Matrizes Polinomiais no `network/bitswap.go`. Dividir o "Byte P2P" via `klauspost/reedsolomon` para tolerância extrema a falhas (Survival Mode do Raspberry/Satelite).

### ONDA 3: Velocidade Lado Servidor & Cripto Lado Borda
- [ ] **Pesquisa 22 (GPU HNSW Offload):** Descarregar as operações Cossenoidais / Hamming da B-Tree na placa de vídeo para nós configurados no Perfil "Cloud".
- [ ] **Pesquisa 27/25 (Post-Quantum & ZKP Poison Defense):** O Swarm P2P precisa do filtro anti-spam na entrada, utilizando Chaves Assimétricas seguras contra os supercomputadores da próxima década.

## 4. Compromisso de Engenharia (C-Level SRE)
Qualquer modificação no "Crompressor V21" precisará honrar 3 regras SRE antes do Merge:
1. **O binário não pode deixar de ser `Pure Go` nos perfis primários** (Se introduzirmos CUDA, será através de Build Tags condicional `//go:build cuda`, preservando a capacidade de compilar pra Android com `GOOS=android` sem quebrar cgo).
2. O **Radioactive Decay** não pode afetar o Mmap nativo do `vfs` montado pelo usuário.
3. O Codebook sempre se provará idêntico através do SHA-256 independentemente do hardware que fez o processamento do Bitswap P2P.
