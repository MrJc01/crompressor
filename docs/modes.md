# Modos de Operação: Edge vs Archive

O Crompressor opera em dois modos mutuamente exclusivos, implementando a **Bifurcação de Shannon**
— uma separação formal entre compressão lossy e lossless.

---

## Edge (Lossy)

```bash
crompressor pack -i input.bin -o output.crom --mode edge
```

**O que faz:** Quantiza cada chunk para o padrão mais próximo no codebook e **descarta o resíduo XOR**.
O output é essencialmente uma sequência de índices do codebook.

| Propriedade | Valor |
|-------------|-------|
| Compressão | ~5-8x (dados densos) |
| Fidelidade | Lossy — MSE ~2.55 |
| SHA-256 match | ❌ Não garante |
| Uso ideal | Inferência de borda, LLMs em CPU, prototipagem rápida |

**Analogia:** É o que GPTQ e AWQ fazem — quantização com perda controlada.

### Por que funciona

Para dados com distribuição gaussiana (como pesos de LLMs), a quantização via codebook
captura a estrutura estatística dominante. O resíduo descartado é majoritariamente ruído
que não contribui significativamente para a qualidade da inferência.

---

## Archive (Lossless)

```bash
crompressor pack -i input.bin -o output.crom --mode archive
```

**O que faz:** Quantiza cada chunk e **armazena o resíduo XOR completo** para reconstrução exata.

| Propriedade | Valor |
|-------------|-------|
| Compressão | ~1.5-2.5x (tensores densos) |
| Fidelidade | Lossless — bit-exact |
| SHA-256 match | ✅ Garantido |
| Uso ideal | Backup, cold storage, distribuição P2P, arquivamento |

**Analogia:** É o equivalente de `zstd` ou `brotli`, mas com dicionário semântico treinado.

### Limites de Shannon

Em dados de alta entropia (>7.9 bits/byte), o resíduo XOR ocupa praticamente o mesmo
tamanho do original. Isso não é um bug — é o limite teórico da informação.
O codebook reduz a redundância estrutural, mas o ruído incompressível permanece no delta.

---

## Comparação Visual

```
Original (100 bytes)
├── Edge:    [idx₁, idx₂, ..., idxₙ]              → ~12 bytes (8x)
└── Archive: [idx₁, idx₂, ..., idxₙ] + [Δ₁...Δₙ]  → ~45 bytes (2.2x)
```

---

## Quando usar cada modo

| Cenário | Modo recomendado |
|---------|-----------------|
| Rodar LLM localmente em CPU com 8GB RAM | **Edge** |
| Backup de dataset de treinamento | **Archive** |
| Distribuir modelo comprimido via P2P | **Edge** (receptor retreina se necessário) |
| Arquivar pesos originais para reprodutibilidade | **Archive** |
| Prototipagem rápida de compressão | **Edge** |
| Produção com garantia de integridade | **Archive** |

---

## Referência Matemática

A formalização completa da Bifurcação de Shannon está documentada em:
→ [crompressor-matematica/papeis/papel6.md](https://github.com/MrJc01/crompressor-matematica)
