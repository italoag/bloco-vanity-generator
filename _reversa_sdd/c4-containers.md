# C4 Containers — Architect

> Como a aplicação é um monólito CLI, os containers abaixo são containers lógicos e de deployment.

## Diagrama

```mermaid
flowchart TB
  User[Operador CLI]

  subgraph LocalMachine[Máquina local / container Docker]
    CLI["CLI Runtime
Go + Cobra + Fang
Comandos, flags, roteamento"]
    TUI["TUI Runtime
Bubble Tea/Bubbles/Lip Gloss
Progresso, stats, benchmark"]
    Config["Config Engine
Go structs/env
Defaults e validação"]
    Worker["Worker Engine
Goroutines/channels
Busca concorrente"]
    Crypto["Crypto Engine
go-ethereum/btcd/solana-go/x-crypto
Carteiras, checksum, KDF, keystore"]
    Validation["Validation Engine
Strategy pattern
Prefix/suffix/checksum"]
    Logging["Secure Logging Engine
Go local
Sanitização, rotação, buffer"]
    Domain["Domain Model
pkg/wallet/pkg/errors/pkg/utils
Entidades e cálculos"]
    Files[("Filesystem Artifacts
keystores/*.json, *.pwd, *.mnemonic, *.log")]
  end

  subgraph CICD[GitHub Actions]
    CI["CI/Test/Lint/Security"]
    Docker["Docker Buildx"]
    Release["Release Assets"]
  end

  User --> CLI
  CLI --> Config
  CLI --> Domain
  CLI --> Worker
  CLI --> TUI
  CLI --> Crypto
  CLI --> Logging
  Worker --> Domain
  Worker --> Crypto
  Worker --> Validation
  Worker --> Logging
  Crypto --> Domain
  Crypto --> Files
  Logging --> Files
  TUI --> Domain
  CI --> CLI
  Docker --> CLI
  Release --> CLI
```

## Containers lógicos

| Container | Tecnologia | Responsabilidades | Dados | Confiança |
|---|---|---|---|---:|
| CLI Runtime | Go, Cobra, Fang | Comandos `generate`, `stats`, `benchmark`, `version`; parsing de flags; orquestração. | `Config`, `GenerationCriteria`, `GenerationResult` | 🟢 |
| TUI Runtime | Bubble Tea, Bubbles, Lip Gloss | Exibição interativa de progresso, tabelas e benchmark. | `GenerationStats`, `BenchmarkResult` | 🟢 |
| Config Engine | Go structs/env | Defaults, variáveis `BLOCO_*`, validação de limites. | `Config` | 🟢 |
| Worker Engine | Goroutines, channels, sync | Execução concorrente, primeiro resultado vencedor, stats. | `WorkerStats`, `WorkResult` | 🟢 |
| Crypto Engine | go-ethereum, btcd, solana-go, x/crypto | Chaves, endereços, checksum, KDF, keystore. | `Wallet`, `KeyStoreV3`, `CryptoParams` | 🟢 |
| Validation Engine | Código local | Estratégias de match por padrão/checksum. | `GenerationCriteria` | 🟢 |
| Secure Logging | Código local | Logs sanitizados, rotação e async buffering. | `LogEntry` | 🟢 |
| Domain Model | `pkg/wallet`, `pkg/utils`, `pkg/errors` | Tipos de domínio, probabilidades, erros. | structs Go | 🟢 |
| Filesystem Artifacts | FS local | Persistência de artefatos gerados. | JSON/text/log | 🟢 |
| CI/CD | GitHub Actions, Docker Buildx | Build, teste, lint, scan e release. | binários, imagens, checksums | 🟢 |

## Ausências relevantes

| Item | Status | Confiança |
|---|---|---:|
| Banco de dados | Não identificado | 🟢 |
| API HTTP própria | Não identificada | 🟢 |
| Fila/cache | Não identificados | 🟢 |
| Serviço remoto runtime obrigatório | Não identificado | 🟢 |
