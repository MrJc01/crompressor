# Benchmarks

O que importa no Crompressor é quantos bytes passam na rede quando os dois lados
já têm o mesmo cérebro. Por isso os dados que o cérebro já viu ficam separados dos
dados novos: os resultados são muito diferentes.

## Bytes na rede com o cérebro já instalado

Blocos de 4 KB, modo archive. Todos os casos restauraram bit a bit. Valores em bytes.

| Caso | Original | Cérebro | `.crom` | zstd -19 | zstd --patch-from |
|---|---|---|---|---|---|
| Visto: projeto idêntico | 27.545.600 | 9.089.536 | 17.017 | 244.461 | não medido |
| Visto: projeto + 1 byte no início | 27.545.601 | 9.089.536 | 15.587 | 244.454 | 2.340 |
| Parecido: versão real mais nova do projeto | 27.555.840 | 9.089.536 | 25.758 | 245.935 | 9.260 |
| Visto: binário aleatório idêntico | 20.000.000 | 19.997.184 | 13.235 | 20.000.473 | não medido |
| Parecido: binário aleatório + 4 bytes no meio | 20.000.004 | 19.997.184 | 12.556 | 20.000.477 | 1.716 |
| Novo: logs de outro dia, mesmo formato | 7.137.622 | 7.086.592 | 1.190.832 | 956.223 | 937.932 |
| Novo: código Go de outros arquivos | 15.237.120 | 13.255.168 | 2.248.321 | 2.187.974 | 2.002.373 |

Como ler esses números:

- **Dados que o cérebro já contém:** só as referências viajam, e elas são comprimidas.
  A redução passa de 99,9%, mesmo em dados incompressíveis.
- **Dados novos do mesmo tipo:** o cérebro entra como dicionário do zstd, mas ainda
  fica atrás do `zstd -19`, com perda de 3% a 25% nos casos medidos. Blocos inteiros
  iguais são raros, e o compressor Go usado aqui é mais fraco que o nível 19.
- **Cérebro:** neste modo ele tem quase o tamanho dos dados que aprendeu. O ganho
  vem de ele já estar instalado e ser reaproveitado em muitas transferências.
- **`zstd --patch-from`:** continua menor quando existe uma única versão base. O
  Crompressor serve para um cérebro comum a muitos arquivos e usuários.

## Sincronização entre nós

`daemon` e `pull` em contêineres Docker separados, com TCP entre IPs diferentes: arquivo
de 30 MB aleatórios, 4 bytes inseridos no meio, receptor com a versão antiga. Passaram
**17.083 bytes** pela rede e o SHA-256 da restauração bateu. O resultado foi o mesmo com
`--peer` e com descoberta por DHT privada (`--bootstrap --private-dht`), sem mDNS.

Não medido: máquinas em redes diferentes, NAT, perda de pacotes e DHT pública.

## Como reproduzir

```bash
make build
scripts/bench_shared_brain.sh ./bin/crompressor /tmp/crom-bench
```

O script precisa de `zstd`, `python3` e da branch `origin/dev`, e usa arquivos Go de
`~/go/pkg/mod`. Por isso os casos de código Go variam de máquina para máquina.
