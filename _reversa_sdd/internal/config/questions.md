# Módulo internal/config, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 2 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Itens triados como perguntas unitárias informativas do Writer. Não foram escalados para validação humana global porque não há lacuna crítica/vermelha neste arquivo.


## Perguntas para Validação Humana

| ID | Pergunta / Lacuna | Impacto | Confiança |
|---|---|---|---:|
| Q-01 | BLOCO_* inválido pode ser ignorado silenciosamente quando parsing falha. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |
| Q-02 | Fallbacks e defaults devem ser mantidos para compatibilidade. | Pode alterar paridade comportamental se resolvido sem decisão explícita. | 🟡 |

## Respostas Consolidadas — 2026-05-14

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-01 | Variáveis `BLOCO_*` inválidas não devem derrubar a aplicação por padrão, para preservar compatibilidade operacional com ambientes existentes, mas também não devem ser completamente silenciosas. | Manter fallback para o default quando o valor vem de ambiente e não puder ser parseado; emitir warning sanitizado em stderr/log operacional. Para flags CLI explícitas, manter erro de validação e exit não-zero, porque o usuário acabou de fornecer o valor inválido. | Respondida |
| Q-02 | Defaults e fallbacks devem ser preservados como contrato de compatibilidade, mas centralizados e documentados. | Usar `DefaultConfig()` como única fonte de defaults; aplicar precedência `defaults < ambiente < flags`; validar a configuração final. Mudanças de default devem ser tratadas como alteração de produto, não como refatoração interna. | Respondida |

## Recomendação

Preservar o comportamento legado confirmado em 🟢 e tratar mudanças sobre itens 🟡/🔴 como decisões explícitas de produto/arquitetura. 🟢
