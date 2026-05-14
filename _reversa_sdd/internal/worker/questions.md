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

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | Goroutine leak e deadlock não são comportamentos de paridade aceitáveis. | Workers devem respeitar `context.Context`, usar canais com dono claro, finalizar via `WaitGroup`, evitar sends bloqueantes após cancelamento e ter testes determinísticos de cancelamento, incluindo execução com `-race`. | Respondida |
| Q-02 | O benchmark não deve preservar o TODO como comportamento alvo. | Implementar benchmark executando o mesmo caminho de geração/validação usado pela CLI, coletando métricas reais do pool. Enquanto não implementado, documentar o benchmark como limitado e evitar claims de performance real. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
