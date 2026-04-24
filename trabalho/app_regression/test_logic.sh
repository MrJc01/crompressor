echo "Executando Teste de Regressão: Variáveis de Ambiente e Mounts"
[ -d "./merged" ] && echo "Pasta Merged OK"
[ -w "./merged" ] && echo "Pasta Escrita OK"
df -h ./merged
