# O que falta fazer

## Decisão

- [ ] Definir o público principal: CDN, backups, modelos de IA ou outro caso.
- [ ] Decidir o futuro da branch `dev` e dos repositórios `crompressor-*`, que dependem
      do formato antigo.

## Prioridade alta

- [ ] Testar o cenário do provedor, do artigo: cérebro treinado com o histórico e versões
      consecutivas de dados reais (`node_modules`, imagens Docker). Medir quanto ele ainda
      serve conforme os dados mudam.
- [ ] Atualizar o cérebro: testar `train --update` sem invalidar os `.crom` existentes.
- [ ] Demonstração com duas VMs trocando dados pelo cérebro.
- [ ] Testar acima de 1,4 GB, com o cérebro final grande (ele precisa caber na RAM).

## Ideia: vídeo em cima de um projeto existente

- [ ] Avaliar, a partir de um repositório como `crom-crompressor-ar`, comprimir horas de vídeo
      sem perda perceptível, com codebook treinado, deduplicação e montadores do arquivo final.
      O bitstream já comprimido não deduplica: testar em quadros decodificados ou blocos do codec.

## Prioridade média

- [ ] Melhorar dados novos, hoje perto do zstd puro, e acelerar literais.
- [ ] Relay para dois lados atrás de NAT sem porta aberta (exige servidor público).
- [ ] Testar `train --update` e os `.crom` antigos quando o cérebro muda.
- [ ] Auditoria externa de segurança do `pull` e do `daemon`.
- [ ] `grep` completo, incluindo literais e texto dividido entre blocos.
- [ ] `cromfs`: listar e ler os arquivos pela montagem.
- [ ] Testes para os wrappers de `pkg/` e para `cmd/`.

## Prioridade baixa

- [ ] Implementar `llm-vfs` de verdade ou remover a demonstração.
- [ ] Reduzir o binário de 48 MB, separando o P2P do núcleo.
- [ ] Investigar por que restam cerca de 13 KB no binário aleatório com poucos bytes alterados.
