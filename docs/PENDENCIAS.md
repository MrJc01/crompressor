# O que falta fazer

## Decisão

- [ ] Definir o público principal: CDN, backups, modelos de IA ou outro caso.
- [ ] Decidir o futuro da branch `dev` e dos repositórios `crompressor-*`, que dependem
      do formato antigo.

## Prioridade alta

- [ ] Melhorar dados novos: um cérebro entre projetos fica perto do zstd puro em projetos que
      não viu. Investigar arquivos menores que o bloco e o dicionário limitado a 32 MB.
- [ ] Testar acima de 1,4 GB, com o cérebro final grande (ele precisa caber na RAM).
- [ ] Acelerar literais: 500 MB de dados novos levam quase 3 min, e `--max-compression`, 22 min.

## Prioridade média

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
