# O que falta fazer

## Decisão

- [ ] Definir o público principal: CDN, backups, modelos de IA ou outro caso.
- [ ] Decidir o futuro da branch `dev` e dos repositórios `crompressor-*`, que dependem
      do formato antigo.

## Prioridade alta

- [ ] Medir um cérebro compartilhado por vários projetos e usuários.
- [ ] Testar com vários GB. O `train` guarda em memória cada bloco único do corpus.
- [ ] Deixar o `pack` mais rápido (cerca de 190 s por GB com o cérebro como dicionário).

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
