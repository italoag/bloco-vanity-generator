# Módulo internal/progress, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 1 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Pergunta já escalada e resolvida no arquivo transversal: corrigir e reativar o progress manager textual.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | O fluxo CLI atual desabilita progress manager textual por deadlocks. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | O progress manager textual deve ser corrigido e reativado como fallback, não preservado como lacuna permanente. | Redesenhar o lifecycle com `context.Context`, fechamento controlado de canais, `WaitGroup`, testes de cancelamento e teste com `-race`. O fallback textual deve ser previsível quando TUI não estiver disponível. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
