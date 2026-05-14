# Inventário — bloco-wallet-generator

Data de geração: 2026-05-08T00:15:00Z

## Resumo executivo

🟢 **CONFIRMADO** — O projeto é uma aplicação CLI em Go chamada `bloco-vgen`, voltada à geração de carteiras/endereço vanity para Ethereum e redes relacionadas, com suporte a padrões por prefixo/sufixo, EIP-55, KeyStore V3, KDF configurável, logging seguro, TUI e processamento concorrente.

## Estrutura de diretórios

🟢 **CONFIRMADO** — Diretórios principais identificados na raiz:

```text
.github/
.specify/
.vscode/
cmd/
docs/
internal/
keystores/
pkg/
specs/
```

Observação: `.git`, `.reversa`, `_reversa_sdd`, `node_modules`, `dist`, `build`, `coverage`, `__pycache__` e `.cache` foram excluídos da varredura conforme regra do Scout.

## Árvore funcional do produto

🟢 **CONFIRMADO** — Estrutura relevante para análise do código:

```text
cmd/
  bloco-vgen/
    main.go
internal/
  cli/
  config/
  crypto/
    kdf/
  progress/
  tui/
  validation/
  worker/
pkg/
  errors/
  logging/
  utils/
  wallet/
docs/
specs/
  001-animated-banner-create/
.github/
  workflows/
```

## Linguagens e tipos de arquivo

🟢 **CONFIRMADO** — Contagem de arquivos do produto, excluindo artefatos do Reversa, agentes e `keystores/`:

| Extensão | Quantidade |
|---|---:|
| `.go` | 61 |
| `.md` | 28 |
| `.sh` | 5 |
| `.yaml` | 4 |
| `.yml` | 3 |
| `.mod` | 1 |
| `.sum` | 1 |
| `.json` | 1 |
| `.log` | 4 |
| sem extensão | 6 |

🟢 **CONFIRMADO** — Linguagem principal: **Go**.

## Módulos identificados

🟢 **CONFIRMADO** — Módulos por estrutura de diretórios:

| Módulo | Caminho | Papel aparente |
|---|---|---|
| CLI principal | `cmd/bloco-vgen/` | Entry point executável |
| CLI | `internal/cli/` | Definição de comandos, flags e handlers Cobra |
| Configuração | `internal/config/` | Defaults, ambiente, validação e overrides |
| Criptografia | `internal/crypto/` | Geração de chaves, wallets, checksum, redes e keystore |
| KDF | `internal/crypto/kdf/` | Scrypt, PBKDF2, análise e compatibilidade de derivação de chave |
| Progresso | `internal/progress/` | Gerência de progresso |
| TUI | `internal/tui/` | Interface terminal, estilos, benchmark e estatísticas |
| Validação | `internal/validation/` | Estratégias de validação de endereço/padrão |
| Worker | `internal/worker/` | Pool concorrente e estatísticas de geração |
| Erros | `pkg/errors/` | Tipos/utilitários de erro compartilhados |
| Logging | `pkg/logging/` | Logger seguro, rotação e tipos |
| Utils | `pkg/utils/` | Formatação/utilidades gerais |
| Wallet | `pkg/wallet/` | Tipos de domínio e logging de wallet |

## Pontos de entrada

🟢 **CONFIRMADO** — Entry point principal:

| Caminho | Tipo | Evidência |
|---|---|---|
| `cmd/bloco-vgen/main.go` | CLI app entry | Função `main`, configuração padrão e execução do Cobra/Fang |

🟢 **CONFIRMADO** — Comandos CLI identificados em `internal/cli/commands.go`:

| Comando | Papel |
|---|---|
| `bloco-vgen` | Geração principal de wallets |
| `stats` | Análise de dificuldade/probabilidade de padrões |
| `benchmark` | Medição de performance de geração |
| `version` | Exibição de versão |

## Arquivos de configuração

🟢 **CONFIRMADO** — Arquivos de configuração e automação encontrados:

| Caminho | Finalidade |
|---|---|
| `go.mod` | Módulo Go e dependências |
| `go.sum` | Checksums de dependências Go |
| `Makefile` | Build, testes, lint, release e tarefas de desenvolvimento |
| `Dockerfile` | Build multi-stage e imagem runtime Alpine |
| `.dockerignore` | Exclusões de contexto Docker |
| `.github/dependabot.yml` | Automação de atualização de dependências |
| `.github/workflows/ci.yaml` | Pipeline CI principal |
| `.github/workflows/docker.yaml` | Build/teste/push de imagem Docker |
| `.github/workflows/go.yaml` | Workflow Go simples de build/test |
| `.github/workflows/release.yaml` | Release com binários, checksums e Docker |
| `.github/workflows/semgrep.yml` | Segurança estática Semgrep |
| `.github/workflows/version-bump.yml` | Versionamento automático por merge de PR |
| `.whitesource` | Configuração WhiteSource/Mend |

## Scripts e tarefas de build

🟢 **CONFIRMADO** — Principais targets do `Makefile`:

| Target | Função |
|---|---|
| `deps`, `tidy` | Dependências Go |
| `build`, `build-all` | Build local e multiplataforma |
| `run`, `run-demo`, `demo`, `examples` | Execução demonstrativa |
| `test`, `test-race`, `test-coverage`, `test-unit`, `bench` | Testes e benchmark |
| `lint`, `fmt`, `vet`, `check`, `ci` | Qualidade e validação |
| `security` | Scan com gosec quando disponível |
| `docs` | Geração de documentação CLI |
| `release` | Preparação de release |

## CI/CD

🟢 **CONFIRMADO** — GitHub Actions detectado em `.github/workflows/`:

| Workflow | Responsabilidade |
|---|---|
| `ci.yaml` | Testes em Ubuntu/macOS, Go 1.24/1.25, vet, fmt, race, coverage, lint, gosec, govulncheck, build e validação |
| `docker.yaml` | Build multi-arch Docker, push para GHCR, Trivy e limpeza de imagens antigas |
| `go.yaml` | Build e testes Go básicos |
| `release.yaml` | Release por tag, binários linux/darwin, checksums e imagem Docker |
| `semgrep.yml` | Semgrep CI agendado/por PR/push |
| `version-bump.yml` | Criação automática de tag por merge em `main` |

## Docker

🟢 **CONFIRMADO** — `Dockerfile` usa build multi-stage:

| Estágio | Base | Função |
|---|---|---|
| `builder` | `golang:1.24-alpine3.20` | Download de módulos, build estático de `./cmd/bloco-vgen` |
| runtime | `alpine:3.20` | Usuário não-root, healthcheck e `ENTRYPOINT ["bloco-vgen"]` |

🟢 **CONFIRMADO** — Não foi identificado `docker-compose.yml`/`docker-compose.yaml`.

## Banco de dados

🟢 **CONFIRMADO** — Não foram encontrados DDL, migrations, schema ORM ou diretório de banco de dados no produto.

🟡 **INFERIDO** — `go.mongodb.org/mongo-driver` aparece apenas como dependência indireta em `go.mod`; não há evidência nesta varredura de uso direto de MongoDB como persistência da aplicação.

## Artefatos sensíveis ou gerados

🟢 **CONFIRMADO** — O diretório `keystores/` contém artefatos gerados de wallet:

| Tipo | Quantidade |
|---|---:|
| `.json` | 170 |
| `.pwd` | 170 |
| `.mnemonic` | 22 |

🔴 **LACUNA** — O conteúdo desses arquivos não foi inspecionado por segurança. Eles devem ser tratados como material sensível ou derivado de material sensível.

🟢 **CONFIRMADO** — Também existem logs em `debug.log` e `internal/worker/wallets-*.log`.

## Cobertura de testes

🟢 **CONFIRMADO** — Framework de teste: `go test` da biblioteca padrão Go, acionado via `Makefile` e GitHub Actions.

🟢 **CONFIRMADO** — Arquivos de teste do produto: **24** arquivos `_test.go`.

Principais áreas cobertas por testes:

- `internal/cli/`
- `internal/config/`
- `internal/crypto/`
- `internal/crypto/kdf/`
- `internal/tui/`
- `internal/worker/`
- `pkg/logging/`

## Integrações externas e bibliotecas relevantes

🟢 **CONFIRMADO** — Integrações/bibliotecas de ecossistema blockchain e terminal:

| Biblioteca | Papel aparente |
|---|---|
| `github.com/ethereum/go-ethereum` | Criptografia/endereço Ethereum e KeyStore |
| `github.com/btcsuite/btcd` / `btcutil` | Suporte Bitcoin/secp256k1 |
| `github.com/gagliardetto/solana-go` | Suporte Solana |
| `github.com/spf13/cobra` | CLI e comandos |
| `github.com/charmbracelet/fang` | Execução CLI com experiência terminal |
| `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` | TUI |
| `golang.org/x/crypto` | Criptografia/KDF |
| `github.com/google/uuid` | Identificadores de KeyStore |

## Sugestão de organização das specs

🟢 **CONFIRMADO** — O projeto possui pastas top-level de domínio técnico sob `internal/` e `pkg/`, além de comandos CLI centralizados em `internal/cli/commands.go`.

🟡 **INFERIDO** — A organização recomendada para as specs é **híbrida**, separando comandos/casos de uso CLI e módulos técnicos críticos (`crypto`, `worker`, `keystore/KDF`, `logging`, `tui`, `config`).
