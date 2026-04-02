echo "Iniciando Stress de Metadados: Criando 1000 Arquivos Pequenos"
for i in {1..1000}; do touch ./merged/stress_$i.txt; done
echo "Stress Concluído. Verificando Contagem:"
ls ./merged/stress_* | wc -l
