# Módulo internal/validation, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 1 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Itens triados como perguntas unitárias informativas do Writer. Não foram escalados para validação humana global porque não há lacuna crítica/vermelha neste arquivo.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Testes documentam bug/risco em validação de sufixo com checksum. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | O bug de validação de sufixo com checksum não deve ser preservado como paridade desejada. | Para Ethereum com `--checksum`, calcular o endereço EIP-55 e validar prefixo/sufixo contra ele. O matching deve ser case-insensitive por padrão e exato somente quando `--case-sensitive` estiver habilitado e validado. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
