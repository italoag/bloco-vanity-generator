# Módulo internal/crypto/kdf, Contratos

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Contratos Expostos

| Contrato | Entrada | Saída | Consumidores | Confiança |
|---|---|---|---|---:|
| `internal/crypto/kdf` | Dados/estado conforme `requirements.md` | Resultado ou erro conforme `design.md` | Módulos do monólito CLI | 🟢 |

## Assinaturas Relevantes

| Arquivo | Símbolo | Retorno | Confiança |
|---|---|---|---:|
| `internal/crypto/kdf/service.go` | `DeriveKey` | 🟢 |
| `internal/crypto/kdf/service.go` | `ValidateParams` | 🟢 |
| `internal/crypto/kdf/analyzer.go` | `AnalyzeKeystore` | 🟢 |

## Garantias

- Entradas válidas devem produzir comportamento equivalente ao legado. 🟢
- Entradas inválidas devem retornar erro ou falha documentada sem pânico. 🟢
- Dados sensíveis não devem ser logados sem sanitização quando aplicável. 🟢
- Contratos marcados como 🟡 exigem validação antes de mudanças de comportamento. 🟡

## Compatibilidade

A compatibilidade é interna ao monólito CLI Go. Não há contrato HTTP/RPC confirmado para esta unit. 🟢
