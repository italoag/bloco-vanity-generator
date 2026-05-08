# Módulo internal/crypto, Contratos

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Contratos Expostos

| Contrato | Entrada | Saída | Consumidores | Confiança |
|---|---|---|---|---:|
| `internal/crypto` | Dados/estado conforme `requirements.md` | Resultado ou erro conforme `design.md` | Módulos do monólito CLI | 🟢 |

## Assinaturas Relevantes

| Arquivo | Símbolo | Retorno | Confiança |
|---|---|---|---:|
| `internal/crypto/ethereum.go` | `GenerateAddressFromPrivateKey` | 🟢 |
| `internal/crypto/ethereum.go` | `OptimizedAddressGeneration` | 🟢 |
| `internal/crypto/checksum.go` | `ToChecksumAddress` | 🟢 |
| `internal/crypto/keystore.go` | `EncryptPrivateKeyWithKDF` | 🟢 detalhe interno; consumidores devem preferir `GenerateKeyStore()` |
| `internal/crypto/password.go` | `GenerateSecurePassword` | 🟢 |

## Garantias

- Entradas válidas devem produzir comportamento equivalente ao legado. 🟢
- Entradas inválidas devem retornar erro ou falha documentada sem pânico. 🟢
- Dados sensíveis não devem ser logados sem sanitização quando aplicável. 🟢
- Contratos marcados como 🟡 exigem validação antes de mudanças de comportamento. 🟡
- `EncryptPrivateKeyWithKDF()` não deve ser tratado como contrato público de integração; a API suportada para consumidores é o serviço de alto nível (`GenerateKeyStore()`/`KeyStoreService`). 🟢

## Compatibilidade

A compatibilidade é interna ao monólito CLI Go. Não há contrato HTTP/RPC confirmado para esta unit. 🟢
