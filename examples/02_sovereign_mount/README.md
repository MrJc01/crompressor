# 02 - Montagem Transparente FUSE e Soberania

Este exemplo demonstra o Filesystem Virtual Soberano (VFS). Diferente de descompactar arquivos inteiros para o disco consumindo RAM e SSD, você monta o arquivo `.crom` e ele se comporta como uma pasta real!

E mais: mostramos o protocolo **Sovereignty Kill**.

## Testando

Inicie o script:
```bash
./run.sh
```

**Em outro terminal**, durante a pausa do script, apague o arquivo `sovereign.cromdb` digitando:
`rm examples/02_sovereign_mount/sovereign.cromdb`

Você verá o diretório montado explodir instantaneamente!
