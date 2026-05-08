# Dependências — bloco-wallet-generator

Data de geração: 2026-05-08T00:15:00Z

## Gerenciador e runtime

🟢 **CONFIRMADO** — O projeto usa Go Modules.

| Item | Valor | Fonte |
|---|---|---|
| Módulo | `bloco-eth` | `go.mod` |
| Versão Go | `1.24.3` | `go.mod` |
| Gerenciador | Go Modules | `go.mod`, `go.sum`, `Makefile` |
| Binário | `bloco-eth` | `Makefile`, `Dockerfile`, `cmd/bloco-eth/main.go` |

## Dependências diretas

🟢 **CONFIRMADO** — Dependências diretas declaradas em `go.mod`:

| Dependência | Versão | Papel aparente |
|---|---:|---|
| `github.com/btcsuite/btcd` | `v0.25.0` | Suporte Bitcoin/secp256k1 |
| `github.com/btcsuite/btcd/btcec/v2` | `v2.3.5` | Curvas criptográficas/secp256k1 |
| `github.com/btcsuite/btcd/btcutil` | `v1.1.5` | Utilitários Bitcoin |
| `github.com/charmbracelet/bubbles` | `v0.21.0` | Componentes TUI |
| `github.com/charmbracelet/bubbletea` | `v1.3.6` | Framework TUI |
| `github.com/charmbracelet/fang` | `v0.4.0` | Execução/experiência CLI |
| `github.com/charmbracelet/lipgloss` | `v1.1.0` | Estilização terminal |
| `github.com/ethereum/go-ethereum` | `v1.16.3` | Ethereum, chaves e keystore |
| `github.com/gagliardetto/solana-go` | `v1.14.0` | Suporte Solana |
| `github.com/google/uuid` | `v1.6.0` | UUIDs, provavelmente KeyStore |
| `github.com/spf13/cobra` | `v1.10.1` | CLI, comandos e flags |
| `github.com/tyler-smith/go-bip32` | `v1.0.0` | Derivação BIP-32 |
| `github.com/tyler-smith/go-bip39` | `v1.1.0` | Mnemonics BIP-39 |
| `golang.org/x/crypto` | `v0.45.0` | Criptografia e KDF |
| `golang.org/x/term` | `v0.37.0` | Terminal |

## Dependências indiretas relevantes

🟢 **CONFIRMADO** — Dependências indiretas de maior relevância técnica:

| Dependência | Versão | Observação |
|---|---:|---|
| `filippo.io/edwards25519` | `v1.0.0-rc.1` | Criptografia Ed25519 |
| `github.com/decred/dcrd/dcrec/secp256k1/v4` | `v4.4.0` | Curva secp256k1 |
| `github.com/holiman/uint256` | `v1.3.2` | Inteiros usados por ecossistema Ethereum |
| `github.com/json-iterator/go` | `v1.1.12` | JSON alternativo |
| `github.com/klauspost/compress` | `v1.16.0` | Compressão |
| `github.com/spf13/pflag` | `v1.0.9` | Flags usadas por Cobra |
| `go.mongodb.org/mongo-driver` | `v1.12.2` | Driver MongoDB indireto; sem uso direto confirmado |
| `go.uber.org/zap` | `v1.21.0` | Logging indireto |
| `golang.org/x/sync` | `v0.18.0` | Primitivas de sincronização |
| `golang.org/x/sys` | `v0.38.0` | Chamadas de sistema |
| `golang.org/x/text` | `v0.31.0` | Texto/unicode |

## Dependências de build, CI e segurança

🟢 **CONFIRMADO** — Ferramentas externas acionadas por automação:

| Ferramenta | Onde aparece | Papel |
|---|---|---|
| `go test` | `Makefile`, `.github/workflows/*.yaml` | Testes unitários, race e cobertura |
| `go vet` | `Makefile`, `ci.yaml` | Análise estática Go |
| `gofmt` | `Makefile`, `ci.yaml` | Formatação |
| `golangci-lint` | `Makefile`, `ci.yaml` | Lint |
| `gosec` | `Makefile`, `ci.yaml` | Security scan Go |
| `govulncheck` | `ci.yaml` | Vulnerability scan Go |
| `Semgrep` | `semgrep.yml` | SAST |
| `Trivy` | `docker.yaml` | Scan de imagem Docker |
| `Codecov` | `ci.yaml` | Upload de cobertura |
| `Docker Buildx` | `docker.yaml`, `release.yaml` | Build multi-arch |

## Dependências Docker

🟢 **CONFIRMADO** — Bases do `Dockerfile`:

| Estágio | Imagem | Versão parametrizada |
|---|---|---|
| Build | `golang:${GO_VERSION}-alpine${ALPINE_VERSION}` | `GO_VERSION=1.24`, `ALPINE_VERSION=3.20` |
| Runtime | `alpine:${ALPINE_VERSION}` | `ALPINE_VERSION=3.20` |

🟢 **CONFIRMADO** — Pacotes Alpine instalados:

| Estágio | Pacotes |
|---|---|
| Build | `git`, `ca-certificates`, `tzdata` |
| Runtime | `git`, `ca-certificates`, `tzdata` |

## Dependências críticas por capacidade

🟢 **CONFIRMADO** — Mapeamento preliminar por responsabilidade:

| Capacidade | Dependências principais |
|---|---|
| CLI | `cobra`, `fang`, `pflag` |
| TUI | `bubbletea`, `bubbles`, `lipgloss`, `x/term` |
| Ethereum/KeyStore | `go-ethereum`, `uuid`, `x/crypto` |
| Bitcoin/secp256k1 | `btcd`, `btcec/v2`, `btcutil`, `secp256k1/v4` |
| Solana | `solana-go`, `edwards25519` |
| Mnemonics/HD wallet | `go-bip39`, `go-bip32` |
| Concorrência/performance | Go runtime, `x/sync` indireto |
| Logging seguro | Código local em `pkg/logging`, dependências indiretas de logging |

## Lacunas e observações

🟡 **INFERIDO** — Não há arquivo de dependências separado para testes; os testes usam a toolchain Go e bibliotecas do próprio módulo.

🔴 **LACUNA** — Não foi executado `go list -m all`; esta extração se baseia no conteúdo estático de `go.mod` e nos workflows.

🟢 **CONFIRMADO** — Não foi identificado `package.json`, `requirements.txt`, `pom.xml`, `Cargo.toml`, `Gemfile`, `composer.json`, `docker-compose.yml` ou `docker-compose.yaml`.
