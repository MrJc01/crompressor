# Arquitetura do Crompressor

## Ideia

Os dois lados têm o mesmo cérebro (`.cromdb`). O `.crom` guarda o que monta o arquivo
a partir dele, então o que o cérebro já conhece não precisa passar pela rede.

## Pipeline do `pack`

```
Arquivo → blocos de 16 MB → corte em torno das entradas do cérebro
        → entrada idêntica: só o ID   | resto: literal
        → literais em zstd, com o cérebro como dicionário
        → tabela de chunks em colunas + blocos comprimidos
```

- **Corte (`internal/search/aligned.go`):** com entradas de 1 KB ou mais, um hash rolante
  acha as entradas em qualquer posição. Entradas menores usam blocos fixos e, opcionalmente,
  `--cdc` (FastCDC) ou casamento aproximado com resíduo XOR.
- **Busca exata (`internal/search/lsh.go`):** O(1), por hash da entrada inteira. A busca
  aproximada só compara entradas com os mesmos 2 primeiros bytes: não é um LSH de verdade.
- **Dicionário (`internal/delta`):** usa até 32 MB finais do cérebro.
  `--max-compression` usa o encoder em C, nível 19; os frames continuam legíveis em Go puro.

## Cérebro (`.cromdb`)

Entradas de tamanho fixo, as mais frequentes do corpus, numeradas na ordem em que aparecem
nos dados. Essa ordem deixa os IDs sequenciais e a tabela quase sem custo. Não há K-Means:
é contagem de frequência. O `train` conta os blocos pelo hash e relê o corpus para copiar só os
escolhidos, então o cérebro final (`-s` × `-k`) precisa caber na RAM, e não o corpus inteiro.

## Formato `.crom`

| Seção | Conteúdo |
|---|---|
| Header | Versão, SHA-256 do original, tamanho, flags, Merkle, cérebro embutido opcional |
| Tabela de blocos | Tamanho comprimido de cada bloco (cada um tem 16 MB do original) |
| Tabela de chunks | `[tamanho][zstd]`, em colunas e com diferenças, cifrada com `--encrypt` |
| Blocos | Resíduos e literais em zstd, cifrados com `--encrypt` |

Se o resultado ficasse maior que o original, o arquivo é gravado inteiro (passthrough).

## Sincronização

1. **Autenticação:** cada lado prova ter o cérebro com HMAC sobre um desafio, sem revelar o hash.
2. **Manifesto:** o receptor recebe a lista de blocos e compara com a versão local.
3. **Diferença:** só os blocos que faltam passam, comprimidos com o cérebro como dicionário.
4. **Reconstrução:** o `.crom` é refeito e o `unpack` confere o SHA-256.

O tráfego é cifrado pelo libp2p. A descoberta usa `--peer`, mDNS na rede local, ou DHT.
