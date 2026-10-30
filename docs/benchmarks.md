# Benchmarks

O que importa no Crompressor é quantos bytes passam na rede quando os dois lados
já têm o mesmo cérebro. Por isso os dados que o cérebro já viu ficam separados dos
dados novos: os resultados são muito diferentes.

## Bytes na rede com o cérebro já instalado

Blocos de 4 KB, modo archive. Todos os casos restauraram bit a bit. Valores em bytes.

| Caso | Original | Cérebro | `.crom` | zstd -19 | zstd --patch-from |
|---|---|---|---|---|---|
| Visto: projeto idêntico | 27.545.600 | 9.089.536 | 177.392 | 244.461 | não medido |
| Visto: projeto + 1 byte no início | 27.545.601 | 9.089.536 | 177.442 | 244.454 | 2.340 |
| Parecido: versão real mais nova do projeto | 27.555.840 | 9.089.536 | 211.932 | 245.935 | 9.260 |
| Visto: binário aleatório idêntico | 20.000.000 | 19.997.184 | 125.562 | 20.000.473 | não medido |
| Parecido: binário aleatório + 4 bytes no meio | 20.000.004 | 19.997.184 | 133.822 | 20.000.477 | 1.716 |
| Novo: logs de outro dia, mesmo formato | 7.137.622 | 7.086.592 | 1.233.261 | 956.223 | 937.932 |
| Novo: código Go de outros arquivos | 15.237.120 | 13.255.168 | 2.521.534 | 2.187.974 | 2.002.373 |

Como ler esses números:

- **Dados que o cérebro já contém:** só as referências viajam, cerca de 25 bytes por
  bloco de 4 KB. É o que permite 99% de redução, mesmo em dados incompressíveis.
- **Dados novos:** blocos inteiros raramente se repetem, então o ganho vem do zstd
  sobre os literais. Hoje o resultado fica pior que o zstd puro.
- **Cérebro:** neste modo ele tem quase o tamanho dos dados que aprendeu. O ganho
  vem de ele já estar instalado e ser reaproveitado em muitas transferências.
- **`zstd --patch-from`:** é melhor quando existe uma única base conhecida, porque
  a tabela do `.crom` ainda não é comprimida.

## Sincronização entre dois nós

`daemon` e `pull` em processos separados, por libp2p em localhost: arquivo de 30 MB
aleatórios, 4 bytes inseridos no meio, receptor com a versão antiga. Passaram 162.147
bytes pela rede e o SHA-256 da restauração bateu. Quase tudo é o manifesto (21 bytes
por bloco), que ainda não é comprimido.

## Como reproduzir

```bash
make build
scripts/bench_shared_brain.sh ./bin/crompressor /tmp/crom-bench
```

O script precisa de `zstd`, `python3` e da branch `origin/dev`, e usa arquivos Go de
`~/go/pkg/mod`. Por isso os casos de código Go variam de máquina para máquina.
