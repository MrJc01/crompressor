# 08 — Casos de Uso Avançados

> *"O CROM não é apenas um compressor. É uma plataforma de representação digital."*

---

## 1. Clones Paramétricos

O conceito mais revolucionário do CROM: se o Codebook contém a "essência" de padrões visuais e binários, o mapa de IDs (`.crom`) se torna uma **receita editável**.

### Como Funciona

```
Foto Original (10MB) → crompressor-pack → foto.crom (50KB)

foto.crom contém:
  chunk[0] = ID:402991  (céu azul, gradiente suave)
  chunk[1] = ID:100234  (textura de areia)
  chunk[2] = ID:882011  (sombra diagonal)
  ...

Editando o .crom:
  chunk[0] = ID:402991 → ID:509123  (céu com nuvens)
  → Nova imagem! Mesma estrutura, céu diferente.
```

**Aplicações:**
- **Geração de variantes**: criar 100 versões de uma imagem alterando poucos IDs
- **A/B Testing visual**: testar layouts com substituição paramétrica
- **Anonimização controlada**: substituir padrões identificáveis por padrões genéricos

### Limitação Lossless

Com clones paramétricos, a garantia lossless se aplica **em relação ao .crom editado**, não ao arquivo original. O clone é uma **nova compilação**, fiel ao seu próprio mapa de IDs.

---

## 2. Compressão Massiva de Datasets

### Cenário: Data Center com 10TB de Logs

```
Sem CROM:
  10TB de logs → gzip → 2TB comprimidos → armazenados

Com CROM:
  10TB de logs → crompressor-pack (Codebook treinado em logs) → 200GB
  
  Por quê? Logs são 80% repetitivos:
  - Timestamps com padrões conhecidos
  - Stack traces recorrentes  
  - Headers HTTP idênticos
  - IPs e UUIDs com estrutura fixa
  
  O Codebook "já sabe" esses padrões → IDs sem Delta.
```

### Cenário: Backup de Imagens Médicas (DICOM)

```
Hospital gera 1TB/dia de imagens DICOM (raios-X, tomografias)

Com Codebook especializado (crom-medical.cromdb):
  - Texturas anatômicas comuns → IDs diretos
  - Metadados DICOM → padrões conhecidos
  - Apenas variações únicas do paciente → Delta
  
  Resultado: 1TB/dia → 30-50GB/dia (95-97% de compressão)
```

---

## 3. Versionamento Ultra-Compacto

Quando dois arquivos compartilham a maioria dos chunks (ex: versões de um documento), os `.crom` gerados serão quase idênticos:

```
documento_v1.docx → doc_v1.crom = [ID:100, ID:200, ID:300, ID:400]
documento_v2.docx → doc_v2.crom = [ID:100, ID:200, ID:350, ID:400]
                                                    ^^^
                                         Apenas 1 chunk mudou!

Diff entre .crom = 1 entrada alterada (~25 bytes)
vs. Diff entre .docx = potencialmente megabytes
```

**Aplicação:** Sistemas de versionamento onde o histórico inteiro pesa quase nada.

---

## 4. Streaming Comprimido

```
Servidor envia stream de dados em tempo real
Ambos (server/client) possuem o mesmo Codebook

Em vez de enviar:  [128 bytes de dados por chunk]
Envia:             [8 bytes de ID + ~2 bytes de delta]

Redução de bandwidth: ~12x em tempo real
Latência: lookup no Codebook < 1ms
```

---

## 5. Deduplicação Cross-File

O Codebook funciona como um **ponto de referência universal**. Se dois arquivos completamente diferentes compartilham padrões binários (ex: headers PNG, sequências JPEG):

```
foto_praia.jpg  → IDs: [402, 991, 338, 772, ...]
foto_monte.jpg  → IDs: [402, 119, 338, 445, ...]
                        ^^^       ^^^
                   Mesmo header PNG | Mesma tabela de cores

→ Os IDs compartilhados são armazenados UMA vez no Codebook.
→ Deduplicação implícita sem overhead de indexação.
```

---

## 6. Compressão de Modelos de ML

Modelos de ML (PyTorch, TensorFlow) são arquivos massivos com muita redundância interna (pesos repetitivos, zeros, padrões de quantização):

```
llama-7b.bin (14GB) → crompressor-pack → llama-7b.crom (~2-4GB)

Com Codebook treinado em pesos de modelos:
  - Padrões de quantização INT8/FP16 → IDs conhecidos
  - Blocos de zeros/near-zeros → Perfect Match
  - Estruturas repetitivas entre camadas → reutilização de IDs
```

---

> **Próximo:** [09 - Benchmarks e Métricas](09-BENCHMARKS_E_METRICAS.md)
