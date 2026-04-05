# 07 — Segurança e Soberania Digital

> *"O arquivo .crom é inútil sem o Codebook. Essa é a segurança mais elegante: a que nasce da arquitetura."*

---

## O Modelo de Segurança do CROM

O CROM implementa **segurança por design**, não por adição. A separação estrutural entre o arquivo comprimido (`.crom`) e o Codebook (`.cromdb`) cria uma camada de proteção intrínseca.

### Anatomia de Segurança

```
┌─────────────────────┐     ┌──────────────────────┐
│   Arquivo .crom     │     │   Codebook .cromdb    │
│                     │     │                       │
│  • IDs numéricos    │     │  • 50GB de padrões    │
│  • Deltas (XOR'd)   │     │  • Indexação HNSW     │
│  • Sem dados brutos │     │  • LOCAL apenas       │
│                     │     │                       │
│  ❌ Sem Codebook:   │     │  ✅ Com .crom:        │
│     Sequência de    │     │     Reconstrução      │
│     números sem     │     │     perfeita          │
│     significado     │     │                       │
└─────────────────────┘     └──────────────────────┘
```

### O Que um Atacante Vê ao Interceptar um `.crom`

```json
{
  "version": 1,
  "codebook_hash": "0xABCD...1234",
  "chunks": [
    {"id": 48293012, "delta": "AAIAAAAAAAg="},
    {"id": 91003847, "delta": "AAAAAAAAAA=="},
    {"id": 12847562, "delta": "ABAAAAAEgA=="}
  ]
}
```

**Sem o Codebook, esses IDs são coordenadas em um mapa que o atacante não possui.** Ele sabe que o ID `48293012` aponta para *algum* padrão de 128 bytes, mas não sabe qual. E o delta está XOR'd contra esse padrão desconhecido.

---

## Soberania Digital: O Modelo Local

### Princípio Fundamental

> **O Codebook NUNCA sai da máquina do usuário.**

| Aspecto | Compressão Tradicional | CROM |
|---|---|---|
| **Dependência de rede** | Nenhuma | Nenhuma |
| **Dependência de API** | Nenhuma | Nenhuma |
| **Dados enviados para nuvem** | Nenhum | Nenhum |
| **Modelo/Dicionário** | Embarcado (~KB) | Local (~50GB) |
| **Quem controla a "chave"** | Qualquer um (algoritmo público) | O proprietário do Codebook |

### Cenários de Soberania

**1. Backup Corporativo Seguro**
```
Empresa comprime 10TB de dados → 100GB de .crom
Envia .crom para cloud storage (Google/AWS/Azure)
Mantém Codebook local (50GB, air-gapped)
→ Cloud vê apenas IDs sem sentido. Zero exposição de dados.
```

**2. Transferência entre Filiais**
```
Filial A (São Paulo): Codebook local + dados originais
Filial B (Lisboa): Mesmo Codebook local (transferido uma vez por pendriver)
Transferência diária: apenas .crom via internet (1% do tamanho)
→ Wire-level encryption implícita pela arquitetura.
```

**3. Compliance LGPD / GDPR**
```
Dados pessoais comprimidos com Codebook proprietário
.crom armazenado externamente (backup)
"Direito ao esquecimento": destrua o Codebook → dados irrecuperáveis
→ Compliance por destruição do decodificador.
```

---

## Camadas de Segurança

### Camada 1: Separação Estrutural (Arquitetural)

O `.crom` é **semanticamente vazio** sem o Codebook. Não é criptografia — é **incompletude informacional**.

### Camada 2: Hash de Vinculação (Integridade)

```go
// O .crom só pode ser aberto com o Codebook EXATO que o gerou
if cromHeader.CodebookHash != codebook.BuildHash {
    return ErrCodebookMismatch
}
```

### Camada 3: Criptografia Opcional (Confidencialidade)

Para cenários que exigem segurança criptográfica adicional:

```go
// Criptografia AES-256-GCM do arquivo .crom
func EncryptCrom(cromData []byte, key [32]byte) ([]byte, error) {
    block, _ := aes.NewCipher(key[:])
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return gcm.Seal(nonce, nonce, cromData, nil), nil
}
```

### Camada 4: Codebook Personalizado (Ofuscação)

Organizações podem treinar Codebooks **proprietários** com seus próprios dados:

```
Codebook Universal (público):  crom-universal-v1.cromdb
Codebook Empresa X (privado):  empresa-x-v3.cromdb
→ Mesmo dado → IDs completamente diferentes
→ Engenharia reversa impraticável sem o Codebook correto
```

---

## Matriz de Ameaças

| Ameaça | Impacto | Mitigação CROM |
|---|---|---|
| Interceptação do `.crom` | Sem acesso ao Codebook: inútil | Separação arquitetural |
| Roubo do Codebook | Sem `.crom`: apenas padrões genéricos | Codebook não revela dados individuais |
| Comprometimento de ambos | Reconstrução possível | AES-256-GCM adicional |
| Supply chain (Codebook adulterado) | Dados corrompidos | SHA-256 de vinculação |
| Brute force de IDs | Impraticável: 500M+ codewords | Espaço de busca imenso |

---

> **Próximo:** [08 - Casos de Uso Avançados](08-CASOS_DE_USO_AVANCADOS.md)
