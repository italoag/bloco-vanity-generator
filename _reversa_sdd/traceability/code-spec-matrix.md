# Code-Spec Matrix

> Artefato global gerado pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | n/a sem unit específica

## Matriz

| Arquivo do legado | Unit correspondente | Cobertura |
|---------|---------------------|-----------|
| `cmd/bloco-vgen/main.go` | `cmd/bloco-vgen/` | 🟢 |
| `internal/cli/commands.go` | `internal/cli/` | 🟢 |
| `internal/config/config.go` | `internal/config/` | 🟢 |
| `internal/crypto/ethereum.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/bitcoin.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/solana.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/checksum.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/keystore.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/password.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/pools.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/random.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/validation.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/generator.go` | `internal/crypto/` | 🟢 |
| `internal/crypto/kdf/service.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/crypto/kdf/scrypt.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/crypto/kdf/pbkdf2.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/crypto/kdf/analyzer.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/crypto/kdf/types.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/crypto/kdf/interfaces.go` | `internal/crypto/kdf/` | 🟢 |
| `internal/progress/manager.go` | `internal/progress/` | 🟢 |
| `internal/tui/manager.go` | `internal/tui/` | 🟢 |
| `internal/tui/progress.go` | `internal/tui/` | 🟢 |
| `internal/tui/stats.go` | `internal/tui/` | 🟢 |
| `internal/tui/benchmark.go` | `internal/tui/` | 🟢 |
| `internal/tui/styles.go` | `internal/tui/` | 🟢 |
| `internal/tui/logo.go` | `internal/tui/` | 🟢 |
| `internal/tui/utils.go` | `internal/tui/` | 🟢 |
| `internal/validation/strategy.go` | `internal/validation/` | 🟢 |
| `internal/worker/pool.go` | `internal/worker/` | 🟢 |
| `internal/worker/stats.go` | `internal/worker/` | 🟢 |
| `internal/worker/interface.go` | `internal/worker/` | 🟢 |
| `pkg/errors/types.go` | `pkg/errors/` | 🟢 |
| `pkg/logging/secure_logger.go` | `pkg/logging/` | 🟢 |
| `pkg/logging/types.go` | `pkg/logging/` | 🟢 |
| `pkg/utils/format.go` | `pkg/utils/` | 🟢 |
| `pkg/wallet/types.go` | `pkg/wallet/` | 🟢 |
| `pkg/wallet/logger.go` | `pkg/wallet/` | 🟢 |
| `cmd/bloco-vgen/main.go` | `cmd/bloco-vgen/`, `cmd/bloco-vgen/inicializacao-da-cli/` | 🟢 |
| `internal/cli/commands.go` | `internal/cli/`, `internal/cli/gerar-carteiras-vanity/`, `internal/cli/salvar-keystore/` | 🟢 |
| `.github/workflows/*.yaml` | `traceability/spec-impact-matrix.md` | 🟡 |
| `keystores/` | `internal/cli/salvar-keystore/`, `internal/crypto/gerar-keystore-v3/` | 🟢 |

## Observações

- Arquivos Go principais possuem cobertura por módulo e, quando aplicável, por caso de uso aninhado. 🟢
- Workflows CI/CD foram cobertos arquiteturalmente e por matriz de impacto, não como units executáveis do runtime CLI. 🟡
- Artefatos sensíveis gerados (`keystores/`, logs legados) são documentados como risco operacional, não como código-fonte. 🟢
