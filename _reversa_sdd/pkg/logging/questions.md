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

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | Logs legados com segredos não devem ser preservados como comportamento alvo. | Migrar para logging seguro/sanitizado com allowlist de campos, redaction de valores sensíveis, permissões restritas quando houver arquivo e testes negativos garantindo ausência de private key, mnemonic, seed e password. | Respondida |
| Q-02 | Para eventos importantes, overflow de buffer deve cair para escrita síncrona em vez de perder logs. | Preservar WARN/ERROR/auditoria por fallback síncrono; se houver modo de alta performance no futuro, permitir descarte apenas para DEBUG/INFO com contador de drops e documentação explícita. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
