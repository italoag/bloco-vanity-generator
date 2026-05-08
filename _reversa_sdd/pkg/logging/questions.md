# Módulo pkg/logging, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Pergunta sobre logs sensíveis já foi escalada e resolvida no arquivo transversal. A questão de overflow de buffer permanece como detalhe de implementação não bloqueante.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Arquivos legados wallets-*.log podem conter termos sensíveis. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Overflow de buffer deve cair para escrita síncrona. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
