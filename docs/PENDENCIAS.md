# O que falta fazer

## Depende de decisão ou de recurso externo

- [ ] Definir o público principal: CDN, backups, modelos de IA ou outro caso.

## Prioridade alta

- [ ] Testes de escala com vários GB, medindo memória e tempo. O manifesto e a
      sincronização carregam tudo em memória.
- [ ] CI: rodar com e sem CGO e incluir o teste entre dois nós (hoje usa `-short`).
- [ ] Medir um cérebro compartilhado por vários projetos e usuários.

## Prioridade média

- [ ] Relay para dois lados atrás de NAT sem porta aberta (exige servidor público).
- [ ] Testar `train --update` e os `.crom` antigos quando o cérebro muda.
- [ ] Revisar a segurança do `pull`: autenticação, limites de tamanho e dados de peers.
- [ ] Testar um arquivo `.crom` real gerado antes destas correções.
- [ ] Testes para `pkg/*`, `cmd/`, `internal/entropy` e `internal/semantic`.

## Prioridade baixa

- [ ] Reduzir o binário de 48 MB, separando o P2P do núcleo.
- [ ] Atualizar `docs/modes.md` (tabela "8x"), `docs/architecture.md` e `MIGRATION_PLAN.md`.
- [ ] Testar `grep`, `cromfs` e `llm-vfs` com o formato novo.
- [ ] Investigar os ~13 KB que sobram no binário aleatório com poucos bytes alterados.
- [ ] Decidir o futuro da branch `dev` e dos repositórios `crompressor-*`, que dependem
      do formato antigo.
