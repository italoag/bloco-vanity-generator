---
schemaVersion: 1
generatedAt: 2026-05-08T11:42:00Z
reversa:
  version: "1.2.34"
kind: target_domain_model
producedBy: designer
hash: "sha256:1777e1ccf552672dc87470638a5d2a1d862cf8378d6fc7c9e6ee2b578b4bbac4"
---

# Target Domain Model

> Modelo de domínio do sistema novo, com rastreabilidade para `_reversa_sdd/domain.md` e `_reversa_sdd/migration/target_business_rules.md`.

## Bounded contexts e aggregates

O domínio alvo usa Go data-oriented e interfaces leves. Os bounded contexts não copiam a árvore legada 1-para-1; eles agrupam invariantes que mudam/falham juntas: comando/configuração, geração, crypto, cofre local, terminal/observabilidade e distribuição.

## Aggregates

### AGG-01: GenerationRequest

- **Aggregate root**: `GenerationRequest`.
- **Bounded context**: BC-01 Experiência de Comando / BC-02 Configuração Operacional.
- **Invariantes**:
  - Configuração inválida impede inicialização da CLI.
  - Flags só sobrescrevem configuração quando explicitamente alteradas.
  - `quiet` e `verbose` são mutuamente exclusivos.
  - `threads=0` significa autodetectar CPU; limites externos respeitam `1..128` e o motor força mínimo 1.
  - `count == 1` seleciona fluxo single; demais valores selecionam fluxo múltiplo.
- **Comandos aceitos**: `BuildGenerationRequest`, `ApplyFlagOverrides`, `ValidateOperationalConfig`, `SelectGenerationMode`.
- **Eventos publicados**: `operation_start`, `config_validated`, `generation_mode_selected`.
- **Origem no legado**: `domain.md` BR-031..BR-036; `target_business_rules.md` BR-MIGRAR-003..007 e BR-MIGRAR-011.

### AGG-02: VanityCriteria

- **Aggregate root**: `VanityCriteria`.
- **Bounded context**: BC-03 Geração Vanity.
- **Invariantes**:
  - Usuário pode informar prefixo, sufixo ou ambos.
  - Tamanho total de `prefix + suffix` não pode exceder 20 caracteres.
  - `MaxAttempts` não pode ser negativo.
  - Validação de prefixo/sufixo depende da rede: Ethereum hex/EIP-55, Bitcoin Base58/bech32 quando aplicável, Solana Base58.
  - Checksum Ethereum torna o match sensível a maiúsculas/minúsculas conforme EIP-55.
- **Comandos aceitos**: `ValidateCriteria`, `NormalizePattern`, `CalculateDifficulty`, `EstimateProbability`, `BuildMatcher`.
- **Eventos publicados**: `criteria_validated`, `difficulty_estimated`.
- **Origem no legado**: `domain.md` BR-001..BR-010; `target_business_rules.md` BR-MIGRAR-008, BR-MIGRAR-009 e BR-MIGRAR-029.

### AGG-03: WalletGeneration

- **Aggregate root**: `WalletGeneration`.
- **Bounded context**: BC-03 Geração Vanity.
- **Invariantes**:
  - Worker pool usa `context.Context` para cancelamento gracioso.
  - Primeiro resultado vencedor encerra a busca single.
  - Geração múltipla registra erro individual sem abortar todo o lote.
  - Estatísticas por worker e agregadas são atualizadas sem race.
  - O motor de geração não decide como persistir nem como renderizar segredos.
- **Comandos aceitos**: `GenerateOne`, `GenerateMany`, `CancelGeneration`, `CollectWorkerStats`, `RunBenchmark`.
- **Eventos publicados**: `generation_started`, `generation_progressed`, `wallet_candidate_matched`, `wallet_generated`, `generation_completed`, `generation_cancelled`, `generation_item_failed`.
- **Origem no legado**: `domain.md` BR-011, BR-031, BR-036; `target_business_rules.md` BR-MIGRAR-004, BR-MIGRAR-011..013 e BR-MIGRAR-029.

### AGG-04: NetworkWallet

- **Aggregate root**: `NetworkWallet`.
- **Bounded context**: BC-04 Criptografia Multirede.
- **Invariantes**:
  - `Network` deve ser uma das redes suportadas: Ethereum, Bitcoin ou Solana.
  - Ethereum usa private key de 32 bytes, secp256k1, Keccak e checksum EIP-55.
  - Bitcoin gera endereço P2PKH com chave pública comprimida e backup por mnemonic.
  - Solana usa Ed25519 e endereço Base58.
  - `Wallet.IsValid()` valida por rede, não por formato Ethereum global.
- **Comandos aceitos**: `GenerateWalletForNetwork`, `ValidateWallet`, `ValidateAddress`, `DerivePublicAddress`, `GenerateMnemonicWhenSupported`.
- **Eventos publicados**: `wallet_generated` como evento interno/logável sem material sensível.
- **Origem no legado**: `domain.md` BR-012..BR-018; `target_business_rules.md` BR-MIGRAR-009, BR-MIGRAR-010 e BR-MIGRAR-014..016.

### AGG-05: SecureArtifactSet

- **Aggregate root**: `SecureArtifactSet`.
- **Bounded context**: BC-05 Cofre Local e Artefatos.
- **Invariantes**:
  - Persistência só ocorre quando keystore/persistência está habilitada.
  - Diretório de saída é configurável.
  - KeyStore Ethereum usa versão 3, AES-128-CTR, scrypt/PBKDF2, salt/IV aleatórios e MAC Keccak.
  - Senha gerada tem no mínimo 12 caracteres com minúscula, maiúscula, número e especial.
  - KDF permitido e parâmetros respeitam limites de segurança.
  - Solana não salva `.key` bruto; usa artefato criptografado/seguro.
  - Arquivos sensíveis usam permissões restritas e escrita segura/atômica quando aplicável.
  - Falha de persistência durante display vira warning/status e não elimina o resultado já gerado.
- **Comandos aceitos**: `GenerateKeyStore`, `GeneratePassword`, `ValidateKDFParams`, `AnalyzeKDFCompatibility`, `PersistArtifacts`, `PersistMnemonic`, `PersistSolanaSecureArtifact`.
- **Eventos publicados**: `artifact_persisted`, `artifact_persistence_failed`, `kdf_analyzed`.
- **Origem no legado**: `domain.md` BR-019..BR-026; Data Master; `target_business_rules.md` BR-MIGRAR-016..023.

### AGG-06: TerminalSession

- **Aggregate root**: `TerminalSession`.
- **Bounded context**: BC-01 Experiência de Comando / BC-06 Observabilidade Segura.
- **Invariantes**:
  - TUI só é usada quando habilitada, progresso ativo, quiet desligado e terminal compatível.
  - Falha da TUI cai para modo texto.
  - `NO_COLOR`, `TERM=dumb`, CI e stdout redirecionado reduzem suporte visual.
  - Progress manager textual deve ter lifecycle/cancelamento testáveis e não causar deadlock.
  - No fluxo single texto, private key/mnemonic podem ser exibidos por compatibilidade, mas com aviso de segurança conforme decisão humana.
- **Comandos aceitos**: `DetectTerminalCapabilities`, `RenderProgress`, `RenderResult`, `ShowSecretWarning`, `FallbackToText`, `StopProgress`.
- **Eventos publicados**: `terminal_mode_selected`, `secret_display_warning_shown`, `tui_fallback_used`.
- **Origem no legado**: `domain.md` BR-033..BR-035; `target_business_rules.md` BR-MIGRAR-026..028 e BR-HUMANA-001 resolvida.

### AGG-07: SecureTelemetry

- **Aggregate root**: `SecureTelemetry`.
- **Bounded context**: BC-06 Observabilidade Segura.
- **Invariantes**:
  - Logs operacionais podem conter endereço, tentativas, duração, thread e status.
  - Logs nunca contêm private key, public key, mnemonic, password, salt ou material criptográfico.
  - Logging pode ser desabilitado por flag/configuração.
  - Em TUI, logging não deve interferir com stdout.
- **Comandos aceitos**: `LogOperationStart`, `LogOperationComplete`, `LogWalletGenerated`, `LogKDFOperation`, `SanitizeFields`, `DisableLogging`.
- **Eventos publicados**: `log_entry_written`, `log_entry_rejected_for_secret`.
- **Origem no legado**: `domain.md` BR-027..BR-030; `target_business_rules.md` BR-MIGRAR-024..025.

### AGG-08: ReleaseContract

- **Aggregate root**: `ReleaseContract`.
- **Bounded context**: BC-07 Distribuição e Qualidade.
- **Invariantes**:
  - CI executa testes, race, lint, gosec/govulncheck/Semgrep e build.
  - Builds oficiais cobrem Linux/macOS amd64/arm64.
  - Release por tag publica binários, checksums e imagem Docker.
  - Docker usa multi-stage e runtime não-root.
  - Escopos amplos de release são preservados por compatibilidade operacional, com risco documentado.
- **Comandos aceitos**: `RunCI`, `BuildReleaseAssets`, `PublishChecksums`, `PublishDockerImage`, `ValidateReleasePermissions`.
- **Eventos publicados**: `release_candidate_built`, `release_published`, `security_scan_completed`.
- **Origem no legado**: `domain.md` BR-037..BR-041; `target_business_rules.md` BR-MIGRAR-030 e BR-HUMANA-002 resolvida.

## Entidades

| Entidade | Aggregate dono | Atributos principais | Origem no legado |
|---|---|---|---|
| `GenerationRequest` | AGG-01 | command, flags, env, config, mode, output options | `internal/cli`, `internal/config`; BR-MIGRAR-003..007, 011 |
| `VanityCriteria` | AGG-02 | network, prefix, suffix, checksum, maxAttempts, useMnemonic | `GenerationCriteria`; `domain.md` BR-001..006 |
| `DifficultyEstimate` | AGG-02 | pattern, difficulty, probability50, ETA, assumptions | `pkg/utils`; `domain.md` BR-007..010 |
| `WalletGeneration` | AGG-03 | criteria, workers, context, start/end time, status | `internal/worker`; BR-MIGRAR-013 |
| `GenerationResult` | AGG-03 | wallet, attempts, duration, workerID, error | `erd-complete.md`; `domain.md` |
| `WorkerStats` | AGG-03 | workerID, attempts, speed, lastUpdate, health, errors | `erd-complete.md` |
| `BenchmarkResult` | AGG-03 | totalAttempts, totalDuration, avg/min/max speed, thread metrics | `erd-complete.md`; BR-MIGRAR-029 |
| `NetworkWallet` | AGG-04 | address, publicKey, privateKey, mnemonic, network, createdAt | `pkg/wallet.Wallet`; Data Master |
| `NetworkDescriptor` | AGG-04 | id, address format, key algorithm, backup policy | Novo, derivado de BR-MIGRAR-009..016 |
| `KeyStoreV3` | AGG-05 | address, id, version, crypto | `KeyStoreV3`; Data Master |
| `KDFParameters` | AGG-05 | algorithm, dklen, salt, scrypt params, pbkdf2 params | `internal/crypto/kdf`; Data Master |
| `SecureArtifactSet` | AGG-05 | JSON, pwd, mnemonic, solanaSecureArtifact, paths, permissions | Data Master; BR-MIGRAR-021..023 |
| `TerminalSession` | AGG-06 | terminal capabilities, mode, progress sink, color policy | `internal/tui`, `internal/progress`; BR-MIGRAR-026..028 |
| `LogEntry` | AGG-07 | timestamp, level, operation, fields, sanitized error | `pkg/logging`; `erd-complete.md` |
| `ReleaseContract` | AGG-08 | targets, checksums, docker image, scan reports, workflow permissions | `.github/workflows`, Dockerfile; BR-MIGRAR-030 |

## Value objects

| Value object | Atributos | Validações | Origem |
|---|---|---|---|
| `Network` | id | enum: ethereum, bitcoin, solana | BR-MIGRAR-014..016 |
| `Address` | network, value | Ethereum hex/EIP-55; Bitcoin Base58/bech32; Solana Base58 | BR-MIGRAR-009..010 |
| `PrivateKey` | bytes/string, network, sensitive marker | tamanho/algoritmo por rede; nunca logável | BR-MIGRAR-014..016, 024 |
| `PublicKey` | value, network, sensitive policy | formato por rede; não logar por decisão de segurança | BR-MIGRAR-024 |
| `Mnemonic` | words, language/profile | BIP-39; não vazio quando persistido | BR-MIGRAR-015, 021 |
| `Pattern` | prefix, suffix, checksum | total <= 20; formato por rede | BR-MIGRAR-008..009 |
| `ThreadCount` | value, source | 0 autodetect no CLI; engine mínimo 1; config 1..128 | BR-MIGRAR-007 |
| `MaxAttempts` | value | não negativo | BR-MIGRAR-008 |
| `KDFAlgorithm` | value | scrypt, pbkdf2, pbkdf2-sha256, pbkdf2-sha512 | BR-MIGRAR-019 |
| `ScryptParams` | n, r, p, dklen, salt | n potência de 2; memória <= limite; ranges válidos | BR-MIGRAR-019 |
| `PBKDF2Params` | c, prf, dklen, salt | c mínimo; prf permitido | BR-MIGRAR-019 |
| `FileMode` | mode | sensíveis 0600; logs sanitizados sem segredo | BR-MIGRAR-023..024 |
| `OutputPath` | baseDir, filename | sanitização de nome e política de não sobrescrita | Data Master; RISK-013 |
| `ProbabilityEstimate` | difficulty, probability, attempts, speed, eta | fórmulas rastreáveis; revisar bases não-hex por rede | BR-MIGRAR-029 |
| `SecretDisplayPolicy` | showSecrets, warningRequired, quiet | fluxo single pode mostrar segredos com warning | BR-HUMANA-001 |
| `WorkflowPermissionPolicy` | scopes, rationale | escopos amplos preservados com risco documentado | BR-HUMANA-002 |

## Eventos de domínio

> O paradigma não é event-driven distribuído, mas o domínio usa eventos internos/logáveis para observabilidade e paridade. Eles não exigem fila, broker ou consistência eventual.

| Evento | Publicado por | Consumido por | Schema (resumido) |
|---|---|---|---|
| `operation_start` | AGG-01 | AGG-07 | operation, timestamp, sanitized config summary |
| `config_validated` | AGG-01 | AGG-07 | config profile, warnings, no secrets |
| `criteria_validated` | AGG-02 | AGG-03, AGG-07 | network, prefix length, suffix length, checksum |
| `difficulty_estimated` | AGG-02 | AGG-06, AGG-07 | difficulty, probability50, ETA, assumptions |
| `generation_started` | AGG-03 | AGG-06, AGG-07 | network, threadCount, mode, startedAt |
| `generation_progressed` | AGG-03 | AGG-06 | attempts, speed, probability, worker summaries |
| `wallet_generated` | AGG-03 / AGG-04 | AGG-05, AGG-06, AGG-07 | address, network, attempts, duration, workerID; no secrets in event sent to logger |
| `generation_item_failed` | AGG-03 | AGG-06, AGG-07 | index, error category, recoverable |
| `artifact_persisted` | AGG-05 | AGG-06, AGG-07 | artifact kind, path, mode, network |
| `artifact_persistence_failed` | AGG-05 | AGG-06, AGG-07 | artifact kind, recoverable, warning text |
| `secret_display_warning_shown` | AGG-06 | AGG-07 | mode, network, timestamp |
| `tui_fallback_used` | AGG-06 | AGG-07 | reason, terminal capabilities |
| `log_entry_rejected_for_secret` | AGG-07 | tests/audit | field, reason, operation |
| `release_candidate_built` | AGG-08 | release process | version, targets, checksums |

## Regras de domínio

| Regra (ID) | Local no domínio novo | Origem (target_business_rules.md) |
|---|---|---|
| BR-MIGRAR-001 | AD-01 e BC runtime local | BR-MIGRAR-001 |
| BR-MIGRAR-002 | AGG-01 `GenerationRequest` e AGG-08 `ReleaseContract` | BR-MIGRAR-002 |
| BR-MIGRAR-003 | AGG-01 `ValidateOperationalConfig` | BR-MIGRAR-003 |
| BR-MIGRAR-004 | AGG-03 `CancelGeneration` / context | BR-MIGRAR-004 |
| BR-MIGRAR-005 | AGG-01 error policy + `pkg/errors` | BR-MIGRAR-005 |
| BR-MIGRAR-006 | AGG-01 `ApplyFlagOverrides` | BR-MIGRAR-006 |
| BR-MIGRAR-007 | Value object `ThreadCount` | BR-MIGRAR-007 |
| BR-MIGRAR-008 | AGG-02 `VanityCriteria` | BR-MIGRAR-008 |
| BR-MIGRAR-009 | Value object `Address` / `Pattern` validators | BR-MIGRAR-009 |
| BR-MIGRAR-010 | AGG-04 `ValidateWallet` | BR-MIGRAR-010 |
| BR-MIGRAR-011 | AGG-01 `SelectGenerationMode`, AGG-03 `GenerateOne/GenerateMany` | BR-MIGRAR-011 |
| BR-MIGRAR-012 | AGG-03 `GenerateMany` result aggregation | BR-MIGRAR-012 |
| BR-MIGRAR-013 | AGG-03 worker engine | BR-MIGRAR-013 |
| BR-MIGRAR-014 | AGG-04 Ethereum provider | BR-MIGRAR-014 |
| BR-MIGRAR-015 | AGG-04 Bitcoin provider + AGG-05 mnemonic backup | BR-MIGRAR-015 |
| BR-MIGRAR-016 | AGG-04 Solana provider + AGG-05 Solana secure artifact | BR-MIGRAR-016 |
| BR-MIGRAR-017 | AGG-05 `GenerateKeyStore` | BR-MIGRAR-017 |
| BR-MIGRAR-018 | AGG-05 `GeneratePassword` | BR-MIGRAR-018 |
| BR-MIGRAR-019 | AGG-05 `ValidateKDFParams` | BR-MIGRAR-019 |
| BR-MIGRAR-020 | AGG-05 `AnalyzeKDFCompatibility` | BR-MIGRAR-020 |
| BR-MIGRAR-021 | AGG-05 `PersistArtifacts` | BR-MIGRAR-021 |
| BR-MIGRAR-022 | AGG-05 warning semantics + AGG-06 display | BR-MIGRAR-022 |
| BR-MIGRAR-023 | Value object `FileMode` + filesystem adapter | BR-MIGRAR-023 |
| BR-MIGRAR-024 | AGG-07 sanitizer/whitelist | BR-MIGRAR-024 |
| BR-MIGRAR-025 | AGG-07 KDF logging policy | BR-MIGRAR-025 |
| BR-MIGRAR-026 | AGG-06 terminal mode selection | BR-MIGRAR-026 |
| BR-MIGRAR-027 | AGG-06 terminal capability detector | BR-MIGRAR-027 |
| BR-MIGRAR-028 | AGG-06 progress lifecycle | BR-MIGRAR-028 |
| BR-MIGRAR-029 | AGG-02/AGG-03 metrics and benchmark | BR-MIGRAR-029 |
| BR-MIGRAR-030 | AGG-08 release contract | BR-MIGRAR-030 |

## Rastreabilidade para o legado

| Elemento novo | Origem no legado | Tipo de mapeamento |
|---|---|---|
| AGG-01 `GenerationRequest` | `cmd/bloco-eth`, `internal/cli`, `internal/config` | dividido/fundido |
| AGG-02 `VanityCriteria` | `pkg/wallet.GenerationCriteria`, `internal/validation`, `pkg/utils` | fundido |
| AGG-03 `WalletGeneration` | `internal/worker`, `internal/cli` geração single/múltipla, benchmark | fundido/dividido |
| AGG-04 `NetworkWallet` | `internal/crypto/{ethereum,bitcoin,solana}`, `pkg/wallet.Wallet` | fundido/dividido |
| AGG-05 `SecureArtifactSet` | `internal/crypto/keystore`, `internal/crypto/kdf`, filesystem helpers | fundido |
| AGG-06 `TerminalSession` | `internal/tui`, `internal/progress`, display em `internal/cli` | fundido |
| AGG-07 `SecureTelemetry` | `pkg/logging`, `WalletLogger` legado descartado | substituição segura |
| AGG-08 `ReleaseContract` | `.github/workflows`, Dockerfile, Makefile | preservado/modernizado |
| `Address` por rede | Validação global legada + decisões humanas | substituição |
| `SolanaSecureArtifact` | `.key` bruto legado | substituição; ver `discard_log.md` BR-DESCARTAR-005 |
| `SecretDisplayPolicy` | stdout single legado + decisão humana | preservado com mitigação |
| `WorkflowPermissionPolicy` | workflows legados + decisão humana | preservado com risco aceito |

## Notas

A modelagem evita eventos distribuídos e agregados artificiais. Eventos listados são contratos internos/logáveis para observabilidade, testes e Inspector. O Inspector deve validar comportamento externo e invariantes sensíveis, não a estrutura interna exata.
