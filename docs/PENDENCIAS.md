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

Comprimir horas de vídeo com codebook treinado, deduplicação e montadores do arquivo final.

- MP4 comum não deduplica: H.264/H.265 já removeram a repetição. Só cópias idênticas ganham.
- É preciso trabalhar em quadros decodificados. Bit a bit, costuma dar mais que o MP4; sem perda
  perceptível, compete com AV1. Faz sentido em acervos repetitivos: câmeras fixas, aulas,
  telejornais, vinhetas (Crom TV).
- Base: `crompressor-video` já decodifica com FFmpeg em blocos de 768 bytes, mas o `encode` grava
  só o hash do bloco mais próximo, sem resíduo: é com perda, e o cérebro é um `.gob` em RAM.
  `crom-crompressor-ar` é um cofre por arquivo inteiro, útil para cópias e como interface.

- O `crompressor` é a origem: o motor fica aqui, e os outros repositórios testam em cima dele.
- Trecho parecido = hash + diferença que o transforma. Em MP4 os bytes não ficam parecidos; em
  quadros decodificados, sim, mas o ruído de câmera impede a diferença de zerar. A busca atual só
  compara blocos com os mesmos 2 primeiros bytes e precisa melhorar para achar blocos parecidos.

- [ ] Testar hash + diferença nos repositórios (`crompressor-video`, `crom-crompressor-ar`),
      usando o motor do `crompressor`, e medir o custo da diferença em vídeo com ruído.
- [ ] Primeiro teste barato: em 1 a 2 horas de vídeo real repetitivo, medir quantos blocos de
      imagem se repetem e comparar com o mesmo vídeo em AV1 (tamanho e qualidade, com VMAF ou SSIM).
- [ ] Só se compensar: criar o pipeline quadros → motor → montador, com resíduo para a qualidade.
- [ ] Conferir se o `crom-crompressor-ar` funciona com o formato novo do motor.

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
