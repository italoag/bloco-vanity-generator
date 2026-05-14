# Módulo internal/crypto, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Perguntas já escaladas e resolvidas no arquivo transversal: placeholder de EncryptPrivateKeyWithKDF e persistência Solana. Os demais pontos permanecem como notas de implementação.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Há placeholder de endereço em fluxo específico de EncryptPrivateKeyWithKDF. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Persistência Solana foi marcada como simplificação/placeholder. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | `EncryptPrivateKeyWithKDF()` deve ser tratado como detalhe interno, não como contrato público a preservar. | Consumidores devem usar `GenerateKeyStore()` ou serviço equivalente de alto nível, que recebe/endossa o endereço real. O placeholder não deve aparecer em API pública nem em artefatos persistidos. | Respondida |
| Q-02 | Persistência Solana não deve permanecer como `.key` bruto ou placeholder inseguro. | Implementar formato criptografado/seguro para Solana ou, se ainda não implementado, documentar explicitamente como suporte parcial e bloquear gravação de chave privada bruta por padrão. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
