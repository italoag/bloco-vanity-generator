# Módulo internal/crypto/kdf, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Itens triados como perguntas unitárias informativas do Writer. Não foram escalados para validação humana global porque não há lacuna crítica/vermelha neste arquivo.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | Ranges da CLI e do KDF podem divergir em alguns parâmetros. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Compatibilidade depende dos clientes Ethereum alvo. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | A divergência entre ranges da CLI e do módulo KDF não deve ser preservada como contrato. | Centralizar validação e ranges no módulo `internal/crypto/kdf`; a CLI deve delegar para esse validador e apenas traduzir erro para mensagem de usuário. Isso evita aceitar na CLI parâmetros rejeitados pelo KDF ou rejeitar parâmetros seguros aceitos pelo serviço. | Respondida |
| Q-02 | A compatibilidade alvo deve cobrir Ethereum KeyStore V3 e os clientes já modelados/testados: geth, Besu, Reth, Anvil e Hyperledger FireFly. | Manter `scrypt` como default compatível e seguro, suportar PBKDF2 com PRF explícita, e classificar incompatibilidades como warnings/relatório de análise quando o keystore ainda for válido para o padrão Ethereum. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
