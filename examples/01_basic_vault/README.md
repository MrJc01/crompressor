# 01 - O Cofre de Segurança Básico

Neste exemplo, você transformará uma pasta contendo documentos sensíveis num arquivo `.crom` compactado e protegido.
Ele só poderá ser lido se houver a combinação exata de Chave Soberana e o respectivo Codebook treinado.

## Passos para testar

Basta rodar o script interativo:
```bash
./run.sh
```

## O que acontece nos bastidores?
1. O CROM lê uma pasta com múltiplos arquivos confidenciais.
2. É gerado um dicionário LSH (Codebook) otimizado exatamente para os padrões e fragmentos daquela pasta.
3. O comando `pack --encrypt` cifra os deltas e empacota, criando um arquivo hermético.
