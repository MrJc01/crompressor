# 🔒 Relatório de Pesquisa 12: V8 Security Fuzzing

## 🎯 Objetivo
Avaliar a robustez do parser do arquivo `.crom` contra manipulação maliciosa (arquivos corrompidos criados para derrubar o sistema via Memory Overflow ou OOM - Out of Memory).

## 🧪 Metodologia
1. **Fuzzing de Dicionário V8:** Modificação proposital dos inteiros de tamanho (Tamanho do Dicionário V8 para `4GB`) em cabeçalhos interceptados, com objetivo de forçar o motor a alocar 4GB de RAM em uma tacada no `Unpack`.
2. **Truncation Attacks:** Cortes prematuros no `.crom` forçando fim de arquivo durante a leitura de chaves criptográficas GCM, Tabela de Blocos e Tabela de Chunks.
3. **Mutações Inválidas:** Verificação rigorosa do número mágico `CMUT` e rejeições silenciosas.

## 📊 Resultados da Execução

*   **OOM Defense (32 MB Cap):** O parser do V8 rejeitou imediatamente a alocação do dicionário falso com tamanho superior a 32MiB.
*   **Safety Limits:** O motor devolve erro claro (e.g. `exceeds safety cap 33554432`) bloqueando o loop de fuzzing em nanosegundos (0.00s de processamento real).
*   **Status de Testes:** ✅ ALL PASS.

## 🧠 Conclusão
A estrutura P2P e local de dados de formato proprietário deve operar num modelo *Zero-Trust Web*. Como qualquer node pode alimentar a rede com dados modificados, o Crompressor blinda nativamente todos os canais de alocação de heap (`make([]byte, size)`) adotando constrições rigorosas e checagens parciais no header V8. Operacional para Cloud Serverless e GossipSub sem falhas de esgotamento OOM.
