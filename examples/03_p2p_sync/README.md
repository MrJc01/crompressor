# 03 - Sincronização Zero-Data Transparente

Este incrível cenário demonstra a superioridade do modelo CROMpressor via P2P.
Dois diretórios independentes (simulando computadores separados) irão rodar instâncias P2P.

Ambos compartilham o mesmo `Codebook` treinado.
Quando o "Node A" anuncia (share) o arquivo pela DHT e o "Node B" pede... ele **NÃO BAIXA OS DADOS**.
Ele transaciona apenas o pequeno **Manifesto** (o DNA estrutural) e constrói o arquivo no Node B extraindo as peças puras direto do `Codebook` local.

## Executando (Simulação)
```bash
./run.sh
```
A visualização explicará Node A (Seeder) e Node B (Leecher).
