# Spec Impact Matrix — Architect

> Matriz de impacto entre capacidades, componentes, dados, regras e artefatos.  
> Escala: 🟢 CONFIRMADO | 🟡 INFERIDO | 🔴 LACUNA

## Matriz por capacidade

| Capacidade / Spec futura | Componentes impactados | Dados impactados | Regras relacionadas | Testes/validação sugeridos | Confiança |
|---|---|---|---|---|---:|
| Geração de carteira Ethereum | `internal/cli`, `internal/worker`, `internal/crypto/ethereum`, `internal/validation`, `pkg/wallet` | `GenerationCriteria`, `Wallet`, `GenerationResult`, `WorkerStats` | BR-001..BR-016, BR-031 | unit de endereço, checksum, worker race, integração CLI | 🟢 |
| Geração Bitcoin | `internal/crypto/bitcoin`, `internal/worker`, `internal/cli`, `pkg/wallet` | `Wallet.Network`, `Mnemonic` | BR-012, BR-014, BR-017 | validação de P2PKH, mnemonic, persistence behavior | 🟢 |
| Geração Solana | `internal/crypto/solana`, `internal/worker`, `internal/cli`, `internal/crypto/keystore` | `Wallet`, artefatos Solana | BR-012, BR-015, BR-018 | testes keypair, base58, persistência real | 🟢 |
| Checksum EIP-55 | `internal/crypto/checksum`, `internal/validation`, `internal/worker`, `pkg/utils` | `GenerationCriteria.IsChecksum`, `Address` | BR-005, BR-008, BR-013 | testes mixed-case e pattern checksum | 🟢 |
| Cálculo de dificuldade/stats | `pkg/utils`, `pkg/wallet`, `internal/tui/stats`, `internal/cli` | `GenerationStats`, `BenchmarkResult` | BR-007..BR-011 | propriedades matemáticas e limites de overflow | 🟢 |
| Worker pool/performance | `internal/worker`, `internal/progress`, `internal/tui`, `pkg/logging` | `WorkerStats`, `AggregatedStats`, `PerformanceMetrics` | BR-031, ADR-0003 | race detector, benchmark real, cancelamento | 🟢 |
| TUI de progresso | `internal/tui`, `internal/cli`, `internal/worker` | `ProgressMsg`, `WalletResultMsg`, `GenerationStats` | BR-011, BR-033, BR-034 | testes de model update, terminal fallback | 🟢 |
| Progresso texto | `internal/progress`, `internal/cli` | `GenerationStats`, `AggregatedStats` | BR-035 | corrigir deadlocks, reativar fallback textual e cobrir com teste concorrente | 🟢 |
| KeyStore V3 | `internal/crypto/keystore`, `internal/cli`, `internal/crypto/kdf` | `KeyStoreV3`, `KeyStoreCrypto`, KDF params | BR-019..BR-025 | compatibilidade geth/MetaMask, decrypt roundtrip | 🟢 |
| KDF universal | `internal/crypto/kdf`, `pkg/logging`, `internal/config`, `internal/cli` | `CryptoParams`, `CompatibilityReport`, `SecurityLevel` | BR-021..BR-023, ADR-0002 | ranges, aliases, security analysis | 🟢 |
| Password/mnemonic | `internal/crypto/password`, `internal/crypto/keystore`, `internal/worker`, `internal/cli` | `Wallet.Mnemonic`, password file | BR-024, ADR-0005 | complexidade senha, BIP-39 recovery, secret hygiene | 🟢 |
| Secure logging | `pkg/logging`, `internal/worker`, `internal/crypto/kdf`, `internal/cli` | `LogEntry`, `BlocoError` | BR-026..BR-030, ADR-0001 | redaction tests, legacy log audit | 🟢 |
| Configuração | `internal/config`, `internal/cli`, `cmd/bloco-eth` | `Config` e subconfigs | BR-031, BR-032 | env vars, flags, validação de conflitos | 🟢 |
| Benchmark | `internal/cli`, `internal/tui/benchmark`, `internal/worker`, `pkg/wallet`, README | `BenchmarkResult`, samples | GAP-007 | README está desatualizado; alinhar docs às flags reais ou implementar features antes de documentar | 🟢 |
| Docker/runtime | `Dockerfile`, `.github/workflows/docker.yaml`, `.github/workflows/release.yaml` | imagem OCI, labels, healthcheck | BR-039, BR-040 | build multi-arch, non-root, scan Trivy | 🟢 |
| CI/CD security | `.github/workflows/*`, `Makefile` | coverage, SARIF, artifacts | BR-037..BR-041, ADR-0007 | permissões mínimas e scanner health | 🟢 |

## Matriz por componente

| Componente | Alta probabilidade de impacto quando mudar | Baixa probabilidade de impacto | Observações | Confiança |
|---|---|---|---|---:|
| `internal/cli/commands.go` | Quase todas as capacidades CLI, TUI, worker, keystore e logging | Crypto internals isolados | Ponto de maior concentração de responsabilidades. | 🟢 |
| `internal/worker/pool.go` | Geração, performance, stats, cancelamento, network behavior | Docker/CI | Mudanças exigem race tests. | 🟢 |
| `internal/crypto/ethereum.go` | Ethereum, checksum, keystore address derivation | Bitcoin/Solana | Cuidado com formato `0x` vs 40 chars. | 🟢 |
| `internal/crypto/bitcoin.go` | Bitcoin, mnemonic backup | Ethereum KeyStore | Regras de persistência diferentes. | 🟢 |
| `internal/crypto/solana.go` | Solana, keypair/base58 | EIP-55 | Persistência ainda é risco. | 🟢 |
| `internal/crypto/keystore.go` | KeyStore, password/mnemonic, filesystem artifacts | TUI layout | Fluxo sensível; exige roundtrip tests. | 🟢 |
| `internal/crypto/kdf/*` | Segurança, compatibilidade, performance de keystore | Worker matching | Ranges e memória são críticos. | 🟢 |
| `pkg/wallet/types.go` | Todas as entidades de domínio e validações | Docker/CI | Refatorar para multi-rede impacta muitos testes. | 🟢 |
| `pkg/logging/*` | Segurança, auditoria, observabilidade | Core address generation | Não deve receber segredo bruto. | 🟢 |
| `internal/tui/*` | UX terminal, progresso, stats, benchmark | Crypto output | Depende de mensagens e modelos estáveis. | 🟢 |
| `internal/config/config.go` | Todos os comandos por defaults/validação | Algoritmos cripto puros | Mudanças exigem testes de env e flags. | 🟢 |
| `.github/workflows/*` | Delivery, scans, release, containers | Runtime local direto | Permissões e tags afetam publicação. | 🟢 |
| `Dockerfile` | Deployment containerizado | Lógica de domínio | Healthcheck e user non-root. | 🟢 |

## Rastreabilidade regras -> componentes

| Regra | Componentes principais | Confiança |
|---|---|---:|
| BR-001..BR-006 | `internal/cli`, `pkg/wallet`, `internal/validation` | 🟢 |
| BR-007..BR-011 | `pkg/utils`, `pkg/wallet`, `internal/tui`, `internal/progress` | 🟢 |
| BR-012..BR-018 | `internal/crypto`, `internal/worker`, `internal/cli` | 🟢 |
| BR-019..BR-026 | `internal/crypto/keystore`, `internal/crypto/kdf`, `internal/cli`, `pkg/logging` | 🟢 |
| BR-027..BR-030 | `pkg/logging`, `internal/worker`, `internal/crypto/kdf` | 🟢 |
| BR-031..BR-036 | `cmd/bloco-eth`, `internal/config`, `internal/cli`, `internal/worker`, `internal/tui` | 🟢 |
| BR-037..BR-041 | `.github/workflows`, `Makefile`, `Dockerfile` | 🟢 |

## Lacunas que deveriam virar specs/tarefas

| Lacuna | Componentes | Prioridade sugerida | Confiança |
|---|---|---:|---:|
| Normalizar naming para `bloco-vanity-generator` em produto/repo/docs e `bloco-vgen` como binário compatível. | `go.mod`, README, Dockerfile, workflows, imports | alta | 🟢 |
| Tornar `Wallet.IsValid()` e validação de vanity por `Network`. | `pkg/wallet`, `internal/worker`, `internal/validation`, testes | alta | 🟢 |
| Tratar `EncryptPrivateKeyWithKDF()` como detalhe interno e expor `GenerateKeyStore()` como contrato suportado. | `internal/crypto/keystore`, `internal/crypto/ethereum` | alta | 🟢 |
| Implementar persistência Solana criptografada/segura, sem `.key` bruto. | `internal/crypto/keystore`, `internal/cli` | alta | 🟢 |
| Corrigir e reativar progress manager texto sem deadlocks. | `internal/progress`, `internal/cli`, `internal/worker` | média | 🟢 |
| Corrigir README desatualizado sobre benchmark/flags ou implementar antes de documentar. | `internal/cli`, `internal/worker`, `internal/tui/benchmark`, README | média | 🟢 |
| Migrar logs legados com chaves para logging seguro/sanitizado. | `pkg/wallet/logger.go`, `pkg/logging`, `.gitignore`, docs | alta | 🟢 |
| Alinhar README desatualizado às flags reais do Cobra. | README, `internal/cli/commands.go` | média | 🟢 |
