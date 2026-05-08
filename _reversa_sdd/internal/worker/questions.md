# Módulo internal/worker, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Itens triados como perguntas unitárias informativas do Writer. Não foram escalados para validação humana global porque não há lacuna crítica/vermelha neste arquivo.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Cancelamento e fechamento de canais exigem cuidado para evitar goroutine leak. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Benchmark CLI ainda não submete WorkItem real ao pool. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
