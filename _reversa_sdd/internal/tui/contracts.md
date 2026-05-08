# Módulo internal/tui, Contratos

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Contratos Expostos

| Contrato | Entrada | Saída | Consumidores | Confiança |
|---|---|---|---|---:|
| `internal/tui` | Dados/estado conforme `requirements.md` | Resultado ou erro conforme `design.md` | Módulos do monólito CLI | 🟢 |

## Assinaturas Relevantes

| Arquivo | Símbolo | Retorno | Confiança |
|---|---|---|---:|
| `internal/tui/manager.go` | `DetectCapabilities` | 🟢 |
| `internal/tui/progress.go` | `Update` | 🟢 |
| `internal/tui/progress.go` | `View` | 🟢 |

## Garantias

- Entradas válidas devem produzir comportamento equivalente ao legado. 🟢
- Entradas inválidas devem retornar erro ou falha documentada sem pânico. 🟢
- Dados sensíveis não devem ser logados sem sanitização quando aplicável. 🟢
- Contratos marcados como 🟡 exigem validação antes de mudanças de comportamento. 🟡

## Compatibilidade

A compatibilidade é interna ao monólito CLI Go. Não há contrato HTTP/RPC confirmado para esta unit. 🟢
