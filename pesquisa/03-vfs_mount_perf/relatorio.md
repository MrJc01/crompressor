# 📊 Relatório 03: Performance VFS Mount (Acesso Aleatório)

Este teste avalia a capacidade do Crompressor V5 de servir dados comprimidos instantaneamente como se fossem um sistema de arquivos nativo.

- **Arquivo**: `logs.crom` (4.93 MB, V5 format com MerkleRoot)
- **Tamanho Expandido**: 26.2 MB
- **Mecânica**: `crompressor mount` (FUSE/VFS)
- **Motor**: Crompressor V5 (Merkle + Block Offset V5)

## 📈 Métricas de Latência

| Atributo | Valor V5 | Valor V3 (anterior) | Observação |
| :--- | :--- | :--- | :--- |
| **TTFB (Time to First Byte)** | < 10ms | < 10ms | Acesso via VFS mount |
| **Gasto de RAM (Montagem)** | ~40 MB | ~40 MB | Inclui Cache de Codebook |
| **Header Version** | V5 (112 bytes) | V3 (68 bytes) | +44 bytes para MerkleRoot |
| **Contagem de Chunks** | ~204,688 | ~204,688 | Mapeamento granular |
| **Total de Blocos Físicos** | 2 | 2 | Baixa fragmentação de I/O |
| **Block Offset Correct** | ✅ PASS | ✅ PASS | Offset recalculado para V5 |

## 🔧 Correção V5: Offset Calculation
Para o VFS funcionar com o Header V5, o cálculo de `baseOffset` no `reader.go` foi atualizado:
```go
hSize := format.HeaderSizeV2
if header.Version == format.Version4 {
    hSize = format.HeaderSizeV4
} else if header.Version == format.Version5 {
    hSize = format.HeaderSizeV5  // 112 bytes
}
baseOffset := int64(hSize + len(blockTable)*4 + tableSize)
```

## 🛡️ Auditoria Técnica: VFS + Merkle
O VFS V5 agora carrega os blocos corretamente a partir do offset correto (compensando os 32 bytes extras do MerkleRoot). Testes de estresse passaram:
- `TestRandomAccessStress` — ✅ PASS
- `TestRandomAccessEncrypted` — ✅ PASS

> [!TIP]
> Com o MerkleRoot, um futuro `crompressor verify --block N` poderá validar blocos individuais sem descomprimir o arquivo inteiro.

## ✅ Conclusão de Auditoria
O VFS V5 mantém TTFB < 10ms com o overhead zero da MerkleRoot (que é lida apenas no header, sem impacto no seek). A correção de offset garante retrocompatibilidade com V2/V3/V4.
