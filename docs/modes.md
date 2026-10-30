# Modos de Operação

## Archive (padrão, sem perda)

```bash
crompressor pack -i dados.bin -o dados.crom -c cerebro.cromdb
```

Guarda tudo o que é preciso para refazer o arquivo. O `unpack` confere o SHA-256 e, se
não bater, não grava nada. Use neste modo para qualquer dado que precise voltar igual.

## Edge (com perda)

```bash
crompressor pack -i dados.bin -o dados.crom -c cerebro.cromdb --mode edge
```

Troca cada bloco pela entrada mais parecida do cérebro e descarta a diferença. O arquivo
fica pequeno (cerca de 25 bytes por bloco antes de comprimir a tabela), mas **o restaurado
não é igual ao original**.

Só faz sentido em dados que toleram erro, como experimentos com pesos de redes neurais.
Vídeo, binários, documentos e bancos de dados ficam corrompidos. O impacto em modelos
reais não foi medido neste repositório.

## Quando usar cada um

| Cenário | Modo |
|---|---|
| Transferência entre usuários, backup, qualquer dado que precise voltar igual | Archive |
| Experimento em que erro é aceitável e foi medido | Edge |
