# Benchmarks

O que importa no Crompressor é quantos bytes passam na rede quando os dois lados
já têm o mesmo cérebro. Por isso os dados que o cérebro já viu ficam separados dos
dados novos: os resultados são muito diferentes.

## Bytes na rede com o cérebro já instalado

Blocos de 4 KB, modo archive. Todos os casos restauraram bit a bit. Valores em bytes.

| Caso | Original | Cérebro | `.crom` | `--max-compression` | zstd -19 | zstd --patch-from |
|---|---|---|---|---|---|---|
| Visto: projeto idêntico | 27.545.600 | 9.089.536 | 6.558 | 6.132 | 244.461 | não medido |
| Visto: projeto + 1 byte no início | 27.545.601 | 9.089.536 | 6.572 | 6.146 | 244.454 | 2.340 |
| Parecido: versão real mais nova do projeto | 27.555.840 | 9.089.536 | 15.187 | 13.238 | 245.935 | 9.260 |
| Visto: binário aleatório idêntico | 20.000.000 | 19.997.184 | 13.204 | 13.205 | 20.000.473 | não medido |
| Parecido: binário aleatório + 4 bytes no meio | 20.000.004 | 19.997.184 | 13.289 | 13.296 | 20.000.477 | 1.716 |
| Novo: logs de outro dia, mesmo formato | 7.137.622 | 7.086.592 | 1.189.575 | 933.060 | 956.223 | 937.932 |
| Novo: código Go de outros arquivos | 14.725.120 | 14.238.208 | 2.282.377 | 2.080.772 | 2.219.530 | 2.026.861 |

Como ler esses números:

- **Dados que o cérebro já contém:** só as referências viajam, com redução acima de 99,9%,
  mesmo em dados incompressíveis.
- **Dados novos do mesmo tipo:** o padrão fica atrás do `zstd -19` porque o compressor em
  Go é mais fraco. Com `--max-compression` (nível 19, encoder em C, binário com CGO), o
  Crompressor passa o `zstd -19` nos dois casos, a um custo de cerca de 25 s nos logs.
- **Cérebro:** neste modo ele tem quase o tamanho dos dados que aprendeu. O ganho vem de
  ele já estar instalado e ser reaproveitado em muitas transferências.
- **`zstd --patch-from`:** ainda é menor com uma única versão base na maioria dos casos. O
  Crompressor serve para um cérebro comum a muitos arquivos e usuários.

## Sincronização entre nós

Arquivo de 30 MB aleatórios com 4 bytes inseridos no meio; o SHA-256 da restauração bateu
em todos os casos que completaram. Nos contêineres, cada nó tinha 80 ms de atraso e 1% de perda.

| Cenário | Resultado |
|---|---|
| Receptor atrás de NAT, emissor público, com versão antiga | 18.952 bytes |
| Igual, sem versão local | 28.059 bytes |
| Dois lados atrás de NAT, porta aberta no emissor | 18.952 bytes |
| Dois lados atrás de NAT, sem porta aberta | **Falha: conexão impossível** |
| DHT privada entre contêineres, sem mDNS (sem atraso nem perda) | 17.083 bytes |
| DHT pública pela internet, dois processos na mesma máquina | 28.059 bytes, achou o peer em 95 s |

Entre um PC atrás de NAT e um servidor público com a porta aberta, conexão direta pela
internet: arquivo de 5 MB com 4 bytes inseridos, 2.973 bytes pela rede em 1 s, com 12 ms de
latência e SHA-256 igual.

Em localhost, com 1 GB aleatório e 4 bytes inseridos: 882.595 bytes pela rede e arquivo
idêntico. Pico de memória de 356 MB no `pull` e cerca de 1,5 GB no `pack`, que levou
cerca de 190 s por GB em 2 threads, com prioridade baixa.

Não medido: acima de 1 GB. Não há relay: quem está atrás de NAT
só recebe conexões com uma porta aberta.

## Como reproduzir

```bash
make build
scripts/bench_shared_brain.sh ./bin/crompressor /tmp/crom-bench
```

O script precisa de `zstd`, `python3` e da branch `origin/dev`, e usa arquivos Go de
`~/go/pkg/mod`. Por isso os casos de código Go variam de máquina para máquina.
