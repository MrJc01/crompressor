# Plano de Migração do Ecossistema Crompressor

> **Data de Criação:** 2026-04-24
> **Status:** Em Andamento
> **Branch de trabalho:** `main` (produção) e novos repositórios

---

## 1. Visão Geral — O que estamos fazendo e por quê

O repositório `crompressor` cresceu organicamente durante a fase de pesquisa e acumulou:
- **~170 MB** de binários e `node_modules` commitados no Git
- Código de GUI, P2P, WASM, e laboratório de pesquisa misturados com o core engine
- README em tom experimental, não adequado para open-source público
- API pública expondo pacotes internos que não estão prontos para consumo

**Objetivo:** Separar responsabilidades em repositórios independentes, limpar a `main` para padrão profissional, e manter a `dev` como laboratório de pesquisa intocado.

---

## 2. Arquitetura do Ecossistema (Decisão Final)

| Repositório | Papel | Status |
|-------------|-------|--------|
| [crompressor](https://github.com/MrJc01/crompressor) | **Core engine** — CLI + biblioteca Go (Pack/Unpack/Verify) | 🔧 Reestruturando |
| [crompressor-gui](https://github.com/MrJc01/crompressor-gui) | **Interface gráfica** — Backend Go (Lorca) + Frontend React/Vite | 🆕 Criando agora |
| [crompressor-matematica](https://github.com/MrJc01/crompressor-matematica) | **Estudo matemático** — Provas, validações, Rate-Distortion | ✅ Existe |
| [crompressor-neuronio](https://github.com/MrJc01/crompressor-neuronio) | **Pesquisa neural** — CromGPT, PTQ, treinamento | ✅ Existe |
| [crompressor-security](https://github.com/MrJc01/crompressor-security) | **Camada de segurança** — AES-GCM, Ed25519, Kill-Switch | ✅ Existe |
| [crompressor-sinapse](https://github.com/MrJc01/crompressor-sinapse) | **Transporte P2P** — GossipSub, Bitswap, Kademlia | ✅ Existe |
| [crompressor-video](https://github.com/MrJc01/crompressor-video) | **Codec de vídeo** | ✅ Existe |

### Repositórios que NÃO serão criados agora (e por quê)

| Candidato | Decisão | Justificativa |
|-----------|---------|---------------|
| `crompressor-sync` | ❌ Não agora | O código P2P (`internal/network/`, `pkg/sync/`) ainda é experimental. Quando amadurecer, migra para `crompressor-sinapse` que já existe para esse propósito |
| `crompressor-wasm` | ❌ Não agora | É apenas 1 arquivo Go (`pkg/wasm/main.go`) + 1 HTML demo. É um build target, não um produto. Mantém na branch `dev` do core |

---

## 3. O que vai para cada lugar

### 3A. `crompressor` (core — branch `main`)

**Fica:**
- `cmd/crompressor/` — CLI (refatorado em arquivos separados)
- `pkg/crom/` — API pública nova (fachada limpa)
- `pkg/cromlib/` — Engine de compressão (implementação)
- `pkg/format/` — Serialização `.crom`
- `pkg/cromdb/` — Leitura de codebooks
- `internal/` — Tudo que é interno ao motor (chunker, codebook, delta, entropy, fractal, merkle, search, trainer)
- `docs/` — Reestruturado (architecture.md, modes.md, benchmarks.md)
- `examples/` — 3 exemplos Go (edge_mode, archive_mode, codebook_train)
- `testdata/` — Fixtures mínimas
- `deployments/`, `monitoring/` — K8s e Prometheus (mantém)

**Sai (migra ou é deletado):**
- `cmd/gui/` → migra para `crompressor-gui`
- `pkg/sdk/` → migra para `crompressor-gui`
- `pkg/sync/` → mantém na `dev`, futuro `crompressor-sinapse`
- `pkg/wasm/` → mantém na `dev`
- `ui/` → migra para `crompressor-gui`
- Binários raiz (`crompressor`, `crompressor-novo`, `basic_usage`, `crompressor_gui`) → deletados
- `default.cromdb` → deletado (artefato runtime)
- `README_en.md` → deletado (novo README principal em PT-BR)
- `relatorio_auditoria.md` → conteúdo migra para `docs/benchmarks.md`
- `examples/www/` (WASM demo) → mantém na `dev`
- `scripts/tests/` → mantém na `dev` (testes de laboratório)

### 3B. `crompressor-gui` (novo repositório)

**Recebe:**
- `cmd/gui/main.go` — Backend Go (handlers REST, WebSocket, Lorca)
- `pkg/sdk/` — SDK que wrapa o `cromlib` para a GUI
- `ui/` — Frontend React/Vite/Tailwind (sem `node_modules`, sem `dist`)
- Makefile próprio
- README próprio (PT-BR + EN)

**Depende de:**
- `github.com/MrJc01/crompressor` (importa `pkg/cromlib`, `pkg/format`, `internal/trainer`)

---

## 4. Renomeações Importantes

| Antes (dev) | Depois (main) | Motivo |
|-------------|---------------|--------|
| `--mode vault` | `--mode archive` | "Vault" confunde com HashiCorp Vault. "Archive" é o termo da indústria para lossless storage |
| `--mode edge` | `--mode edge` | Mantém — é claro e correto |
| `pkg/cromlib/` | `pkg/crom/` (fachada) + `pkg/cromlib/` (impl) | API pública limpa, implementação interna separada |

---

## 5. Checklist de Migração do `crompressor` (main)

> Referência completa em: task.md do Antigravity (Fase 0 a Fase 13)

### Resumo das Fases:
1. **Fase 0** — Backup (tag `pre-migration`)
2. **Fase 1** — Limpar binários e node_modules (~170 MB)
3. **Fase 2** — Atualizar `.gitignore`
4. **Fase 3** — Reestruturar `docs/`
5. **Fase 4** — Reestruturar `examples/`
6. **Fase 5** — Refatorar CLI (1 arquivo monolítico → 6 arquivos)
7. **Fase 6** — Criar `pkg/crom/` (API pública)
8. **Fase 7** — Atualizar `go.mod`
9. **Fase 8** — Simplificar `Makefile`
10. **Fase 9** — Substituir README
11. **Fase 10** — Criar CONTRIBUTING.md e CI/CD
12. **Fase 11** — Limpar `scripts/`
13. **Fase 12** — Validação (build, test, bench, lint, round-trip)
14. **Fase 13** — Commit e push

---

## 6. Checklist de Criação do `crompressor-gui`

- [ ] 6.1 Criar diretório e copiar arquivos do `dev`
- [ ] 6.2 Criar `go.mod` com módulo `github.com/MrJc01/crompressor-gui`
- [ ] 6.3 Criar `.gitignore` (inclui `ui/node_modules/`, `ui/dist/`, binários)
- [ ] 6.4 Criar `Makefile` (build backend, build frontend, dev server)
- [ ] 6.5 Criar `README.md` (PT-BR)
- [ ] 6.6 Criar `README_en.md` (EN)
- [ ] 6.7 Criar repositório no GitHub via `gh repo create`
- [ ] 6.8 Primeiro commit e push
- [ ] 6.9 Verificar que renderiza corretamente no GitHub

---

## 7. Dependências entre as tarefas

```
crompressor-gui (criar primeiro, para não perder código)
       ↓
crompressor main (limpar, reestruturar)
       ↓
Validação final (build, test, round-trip em ambos)
```

> **IMPORTANTE:** Criar o `crompressor-gui` ANTES de limpar a `main` do `crompressor`,
> para garantir que nenhum código é perdido durante a migração.

---

## 8. Conceitos Técnicos — Bifurcação de Shannon

A engine opera em dois modos mutuamente exclusivos:

```
CROM(X) = Σᵢ C[q(chunkᵢ(X))] ⊕ Δᵢ
```

- **Modo Edge (Lossy):** `Δᵢ` é descartado. Output = sequência de índices do codebook.
  - Compressão: ~5-8x mesmo em dados de alta entropia
  - MSE: ~2.55 (sem QAT)
  - Uso: Inferência de borda, rodar LLMs em CPU

- **Modo Archive (Lossless):** `Δᵢ` é armazenado via XOR.
  - Compressão: ~1.5-2.5x em tensores densos
  - SHA-256 do output = SHA-256 do input
  - Uso: Cold storage, backup, P2P distribuído

Referência completa: [crompressor-matematica/papeis/papel6.md](https://github.com/MrJc01/crompressor-matematica)

---

## 9. Riscos e Mitigações

| Risco | Probabilidade | Mitigação |
|-------|--------------|-----------|
| Perda de código durante migração | Baixa | Tag `pre-migration` + branch `dev` intocada |
| `go.mod` do GUI quebrar imports | Média | Testar build do GUI contra versão publicada do core |
| Testes quebrarem após renomeação vault→archive | Média | Atualizar todos os testes antes de renomear |
| GitHub CI falhar na primeira execução | Alta | Testar localmente antes, CI é iterativo |

---

## 10. Estado Atual (Atualizar conforme progresso)

- [x] Bifurcação de Shannon implementada e testada
- [x] Branch `dev` sincronizada com `origin/dev`
- [x] Branch `main` sincronizada com `origin/main`
- [x] Branch `main` limpa de pesquisa/trabalho
- [ ] `crompressor-gui` criado e publicado
- [ ] `crompressor` main reestruturado conforme checklist
- [ ] Validação final executada
