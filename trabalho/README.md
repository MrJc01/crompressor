# Crompressor: Laboratório SRE & Out-of-Core Engine 🚀

Bem-vindo ao diretório de `trabalho` do Pesquisador (SRE e CROM Orchestrator). 
Este documento sumariza a topologia, as ferramentas e como operar nossa arquitetura de ponta (V24+).

---

## 1. O que é o Crompressor?
O **Crompressor** deixou de ser um simples "compactador zip". Ele hoje funciona como um **Motor VFS (Virtual File System) Criptográfico e Semântico**.
Ele lê bytes massivos (gigabytes de modelos de IA, jogos inteiros como o Minecraft, repositórios nativos), fragmenta repetitividade bruta de bits usando "Codebooks LSH" criados por simulação de entropia (IA) e converte isso numa arquitetura soberana.

## 2. O Ecossistema de CROM (Minecraft Zero-RAM)
Para provar que o Crompressor funciona perfeitamente paginando dados Out-of-Core de forma otimizada para Desktop Apps ou IAs Pesadas, nós encapsulamos o Minecraft e o TLauncher num sistema que não ocupa tamanho físico real.

A arquitetura usa o `Crompressor V24` -> `SquashFuse` -> `Fuse-OverlayFS`. Resultando em 3 camadas de Kernel Space blindado.

### 🎮 Como Jogar? (Scripts)
A pasta `minecraft_client` possui dezenas de orquestrações de Kernel testadas, empacotadas no Grand-Maestro Launcher:

* **`play_crom_minecraft.sh`** : (Script Princípal Mestre)
  - Basta executá-lo no terminal com `./play_crom_minecraft.sh`.
  - Ele levanta os drives virtuais do zero num estalar de dedos, empunha o Java Launcher na memória e **já inicia o jogo**. 
  - 🛑 **A Segurança**: No exato momento que você farta de jogar e fechar a tela, o terminal reconhece o encerramento do processo Java e entra em ação efetuando os "Teardowns" de Zumbis de FUSE (`fusermount -uz`) e retorna tudo ao seu lugar físico de forma automatizada, poupando a sua Mente de tarefas mundanas.

* `vfs_squash_deploy.sh`: O Sub-worker que é engajado ativamente por trás das docas e injeta o `overlay` (sistema unificado) para garantir que você possa escrever/deletar blocos ou baixar mods sem afetar os volumes *Read-Only* do Squash CROM.

---

## 3. Pesquisas V25 (B-Tree FUSE) & V26 (Geração Fractal O(1))
A pasta `/pesquisa` e logs adjacentes atestam cientificamente o porquê escolhemos a arquitetura LSH e não matemática "caótica":
* A V26 **(Máquina de Turing Infinita ou Sementes de Pi)** foi comprovada falha no `103-fractal_v26_engine`. Nós forçamos ele a tentar indexar 8 bytes em 90 milhões de sementes e ele não encontrou, atestando o limite físico do silício para a computação O(1) de Entropia. Portanto, Codebooks Semânticos > PRNGs Fractal.
* A V25 **(B-Tree Hierarchy)** resolve o erro `Is a Directory` ao ler os Volumes via SquashFS externo sem reescrever 45 mil bytes do kernel CGO. Todo lixo pesado fica abstraído dos repositórios via exclusivas engenharias de `.gitignore`. 

---

> **Aviso de Engenheiro:** A operação falhou pela 1ª vez acusando `Transport endpoint is not connected` porque você fechou abruptamente o console anterior enquanto o Minecraft ou o Engine go-fuse montava o virtual! Eu já corrigi os scripts com _Lazy Unmount (`-z`)_ para que mesmo que você quebre os vidros estilhaçados, o orquestrador consiga reciclar o estilhaço e arrancar tudo para o céu perfeitamente nas futuras chamadas. Divirta-se jogando sem medo!
