# 🌌 Roadmap Estratégico Crompressor V14: "The Metamorphic Brain & Neural Telemetry"

**Objetivo Central**: Ultrapassar o limite de Dicionários Estáticos transferindo a Geração de Novos Cérebros (Aprendizado de Máquina) para um aspecto puramente Matemático, Determinístico e In-Band. O Crompressor V14 gera **Cérebros Mutantes Filhos (Epigenéticos)** a partir das falhas locais e os embala criptograficamente dentro do próprio `Header` do arquivo `.crom`, eliminando Risco P2P de contaminação cruzada cibernética.

## 🧬 Fases de Engenharia e Checklist (V14)

### Fase 1: O "Termômetro" de Entropia (Telemetria O(1))
Antes de adotar um cérebro dinâmico, o sistema compila métricas deterministas de Miss-Rate e Ineficiências Locais:
- `[ ]` **Tarefa 1.1 (Tracking O(1) de Bloqueios LSH)**: Mapear via Hashes FNV-1a de 64-bits todo chunk rotulado como *Literal Bypass* durante a compressão P1.
- `[ ]` **Tarefa 1.2 (Sinalizador Inteligente de Termodinâmica)**: Ao fim da rotina `Pack()`, a nova propriedade interna das métricas (`SuggestedMicroBrain`) avisa ao compilador se a repetição de padrões rejeitados atingiu a massa crítica que justificaria a Forja Sub-Crom.

### Fase 2: Estruturação do Header V8 (Micro-Dict In-Band Embedded)
- `[ ]` **Tarefa 2.1 (Suporte à Mutação)**: No `pkg/format/format.go`, implementar a `Version8`, criando extensões para além dos 137 bytes retrocompatíveis. 
- `[ ]` **Tarefa 2.2 (Data-Structure V8)**: Embutir campos como `MicroDictSize (uint32)` e o Byte Array cru contendo o Modelo Neural BPE Gerado, de forma que o Decodificador carregue-o localmente a partir da própria base de dados encenada. Todo atacante que tentar fraudar os cérebros seria blindado pelas próprias chaves AES de Convergent Encryption acopladas a eles.

### Fase 3: Spawning Epigenético (Geração Pós-Termômetro)
- `[ ]` **Tarefa 3.1 (A Fábrica Automática)**: Conectar uma flag da CLI/Interface (`ALLOW_EPIGENESIS`). Caso um P2P deseje compactar dados alienígenas massivos que ativam o alarme da Fase 1, o crompressor ativa o loop de treino determinístico *Multi-Pass* e constrói instantaneamente o Micro Dicionário Local. 
- `[ ]` **Tarefa 3.2 (Neural Binding)**: O Cérebro Mutante será amarrado estritamente à Indexação da Chunk Table (`CodebookIndex = 254`), deixando o espaço 255 intacto como fallback de Bypass Global Literal. 

### Fase 4: O Isolamento VFS contra Overflows
- `[ ]` **Tarefa 4.1 (LoadRawMemory)**: Criar Construtor Descentralizado (`internal/codebook`) para que o `vfs/reader.go` (FUSE System Mount Local) inicie o leitor carregando os padrões O(1) diretamente da varredura paralela do Array de memória instanciado do header V8.
- `[ ]` **Tarefa 4.2 (Defesa OOM - Out of Memory Killing)**: Limitar matematicamente o `MicroDictSize` ao Hardcap de 32MB no Reader remoto para evicção total de injeções forjadas hostis (Zero Trust Boundary).

### Fase 5: Validação Global & Metamorphic Research
- `[ ]` **Tarefa 5.1 (Pesquisa 11 - Spawning Efficiency)**: Testar na AWS ou local o tempo de geração do Micro-Cérebro contra um arquivo JSON massivo de logs.
- `[ ]` **Tarefa 5.2 (Pesquisa 12 - Security Fuzzing)**: Ferramentas gerando ruído branco no Header V8 e quebrando checksums intencionalmente para assegurar que a biblioteca FUSE recusa o arquivo instantaneamente.
- `[ ]` **Tarefa 5.3 (Pesquisa 13 - DHT Epigenetic Sync)**: Confirmar que a matriz Kademlia Bitswap não se confunde com arrays dinâmicos V8 e roteia os blocos mantendo *Convergent Encryption* imune a espionagem In-band.

---

### Principais Riscos Levantados
1. **Memória de Picos VFS**: Inicializar o Cérebro nativo `internal/codebook.Reader` em memória RAM, sem uso de MapFiles (`.cromdb` físico), exingindo Otimização Go Garbage Collector.
2. **Corrupção Silenciosa das Flags V7**: Mexer no Deslocamento da Tabela FUSE devido ao array in-band precisará de cautela. Todas as tabelas de Chunk/Blocks são contadas no Global Offset. Isso vai exigir refazer as somatórias de `HeaderSize`.
