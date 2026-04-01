# Pesquisa 103 — Motor Fractal V26 Multiestratégia

## Objetivo
Validar a viabilidade de comprimir Data Chunks (16 bytes) usando 4 estratégias de regressão matemática, medindo CPU-time e taxa de match para cada uma.

## Estratégias Testadas
| # | Estratégia | Descrição |
|---|---|---|
| 1 | PRNG (Go rand) | Brute-force linear de seeds no gerador pseudo-aleatório de Go |
| 2 | XOR Fractal Iterado | Iteração caótica XOR com acumulador de estado |
| 3 | Polinomial (ax²+bx+c) | Regressão polinomial de 2º grau mod 256 |
| 4 | Fibonacci Modulado | Sequência de Fibonacci com seeds arbitrárias |

## Análise Teórica
- **Shannon**: Chunks de entropia < 7.5 bits/byte possuem padrões exploráveis.
- **Kolmogorov-Chaitin**: Dados verdadeiramente aleatórios NÃO podem ser representados por fórmula menor. O fallback para literal é obrigatório.
- **Viabilidade**: Estratégias determinísticas (Poly, Fib) são O(1) na reconstrução JIT. PRNG exige apenas a seed.

## Como Executar
```bash
bash scripts/01_test.sh
```
