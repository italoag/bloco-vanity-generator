# Módulo pkg/wallet, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Perguntas já escaladas e resolvidas no arquivo transversal: Wallet.IsValid por rede e migração do logger legado para logging seguro.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Wallet.IsValid conflita com Bitcoin/Solana e endereço Ethereum com 0x. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Logger legado pode persistir dados sensíveis se usado. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
