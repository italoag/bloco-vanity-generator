# State Machines — Detective

> Projeto: `bloco-wallet-generator`  
> Escala: 🟢 CONFIRMADO | 🟡 INFERIDO | 🔴 LACUNA

## Visão geral

O sistema não possui entidades persistentes com ciclo de vida de negócio rico, como Pedido, Usuário ou Conta. As máquinas de estado encontradas são principalmente **operacionais**: execução CLI, geração de carteira, TUI, worker pool, logging e CI/CD.

## Máquina 1 — Execução CLI

| Estado | Descrição | Confiança |
|---|---|---:|
| `initialized` | Processo iniciou, contexto de cancelamento criado. | 🟢 |
| `config_loaded` | Defaults carregados e variáveis de ambiente aplicadas. | 🟢 |
| `config_invalid` | Validação falhou; processo encerra com código 1. | 🟢 |
| `command_ready` | Aplicação Cobra/Fang criada. | 🟢 |
| `running_command` | Comando raiz/subcomando em execução. | 🟢 |
| `completed` | Execução sem erro. | 🟢 |
| `failed` | Erro tratado por `handleError()`. | 🟢 |
| `cancelled` | SIGINT/SIGTERM cancela contexto. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> initialized
  initialized --> config_loaded: DefaultConfig + LoadFromEnvironment
  config_loaded --> config_invalid: Validate falha
  config_invalid --> failed: stderr + exit 1
  config_loaded --> command_ready: Validate ok
  command_ready --> running_command: fang.Execute
  running_command --> completed: sem erro
  running_command --> failed: erro
  running_command --> cancelled: SIGINT/SIGTERM
  cancelled --> failed: contexto cancelado propagado
  completed --> [*]
  failed --> [*]
```

## Máquina 2 — Geração de carteira

| Estado | Descrição | Confiança |
|---|---|---:|
| `criteria_parsed` | Flags convertidas em `GenerationCriteria`. | 🟢 |
| `criteria_invalid` | Padrão inválido, muito longo ou max attempts negativo. | 🟢 |
| `pool_started` | Worker pool iniciado. | 🟢 |
| `searching` | Workers geram chaves e endereços. | 🟢 |
| `match_found` | Endereço satisfaz prefixo/sufixo/checksum. | 🟢 |
| `keystore_saving` | Persistência de keystore/mnemonic quando habilitada. | 🟢 |
| `result_displayed` | Resultado exibido em TUI/texto. | 🟢 |
| `cancelled` | Contexto cancelado. | 🟢 |
| `failed` | Erro de geração, worker, validação ou keystore. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> criteria_parsed
  criteria_parsed --> criteria_invalid: Validate falha
  criteria_invalid --> failed
  criteria_parsed --> pool_started: Validate ok
  pool_started --> searching: Start workers
  searching --> match_found: matchesCriteria true
  searching --> cancelled: ctx.Done
  searching --> failed: erro do generator/worker
  match_found --> keystore_saving: KeyStore.Enabled
  match_found --> result_displayed: KeyStore disabled
  keystore_saving --> result_displayed: salvo ou warning não fatal
  keystore_saving --> failed: erro fatal retornado no fluxo
  result_displayed --> [*]
  cancelled --> [*]
  failed --> [*]
```

## Máquina 3 — Worker pool

| Estado | Descrição | Confiança |
|---|---|---:|
| `stopped` | Pool criado, não executando. | 🟢 |
| `running` | `Start()` ativou execução e stats. | 🟢 |
| `generating` | Goroutines em loop de tentativa. | 🟢 |
| `winner_selected` | Primeiro resultado enviado em `resultCh`. | 🟢 |
| `shutdown` | `Shutdown()` cancela/encerra execução. | 🟢 |
| `error` | Falha ao iniciar ou gerar. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> stopped
  stopped --> running: Start
  running --> generating: GenerateWalletWithContext
  generating --> winner_selected: primeiro match
  generating --> shutdown: ctx.Done ou Shutdown
  generating --> error: erro fatal
  winner_selected --> shutdown: defer Shutdown
  shutdown --> stopped
  error --> stopped
```

## Máquina 4 — TUI de progresso

| Estado | Descrição | Confiança |
|---|---|---:|
| `rendering_progress` | Exibe padrão, dificuldade, barra e métricas. | 🟢 |
| `showing_results` | Recebe `WalletResultMsg` e exibe tabela. | 🟢 |
| `complete` | Todos os wallets concluídos; progresso 100%. | 🟢 |
| `quitting` | Usuário pressiona `q`/`ctrl+c` ou `QuitMsg`. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> rendering_progress
  rendering_progress --> rendering_progress: ProgressMsg/TickMsg
  rendering_progress --> showing_results: WalletResultMsg
  showing_results --> showing_results: novos WalletResultMsg
  showing_results --> complete: CompletedWallets >= TotalWallets
  rendering_progress --> quitting: q/ctrl+c/QuitMsg
  showing_results --> quitting: q/ctrl+c/QuitMsg
  complete --> quitting: delay + quit signal
  quitting --> [*]
```

## Máquina 5 — Benchmark TUI

| Estado | Descrição | Confiança |
|---|---|---:|
| `BenchmarkStateProgress` | Benchmark em andamento, exibindo progresso. | 🟢 |
| `BenchmarkStateTransitioning` | Resultado recebido e transição visual. | 🟢 |
| `BenchmarkStateResults` | Tabela final de resultados. | 🟢 |
| `quitting` | Usuário encerra. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> BenchmarkStateProgress
  BenchmarkStateProgress --> BenchmarkStateProgress: BenchmarkUpdateMsg running
  BenchmarkStateProgress --> BenchmarkStateTransitioning: BenchmarkCompleteMsg
  BenchmarkStateTransitioning --> BenchmarkStateResults: 500ms elapsed
  BenchmarkStateResults --> quitting: q/ctrl+c
  BenchmarkStateProgress --> quitting: q/ctrl+c
  quitting --> [*]
```

## Máquina 6 — Logging seguro

| Estado | Descrição | Confiança |
|---|---|---:|
| `disabled` | Logging desabilitado; writer `io.Discard`. | 🟢 |
| `sync_writer` | Escrita síncrona em stdout/arquivo/discard. | 🟢 |
| `buffered_writer` | Buffer assíncrono ativo. | 🟢 |
| `flushing` | Flush aguardando drain do buffer. | 🟢 |
| `rotating` | Rotação por tamanho de arquivo. | 🟢 |
| `closed` | Logger fechado. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> disabled: config.Enabled=false
  [*] --> sync_writer: Enabled + BufferSize=0
  [*] --> buffered_writer: Enabled + BufferSize>0
  buffered_writer --> flushing: Flush
  flushing --> buffered_writer: concluído
  sync_writer --> rotating: arquivo excede limite
  buffered_writer --> rotating: arquivo excede limite
  rotating --> sync_writer
  rotating --> buffered_writer
  sync_writer --> closed: Close
  buffered_writer --> closed: Close
  disabled --> closed: Close
```

## Máquina 7 — CI/CD

| Estado | Descrição | Confiança |
|---|---|---:|
| `test` | Testes curtos, race detector e cobertura. | 🟢 |
| `lint` | golangci-lint, vet e fmt. | 🟢 |
| `security` | gosec, govulncheck, Semgrep/Trivy dependendo workflow. | 🟢 |
| `build` | Binários Linux/Darwin amd64/arm64. | 🟢 |
| `release` | Release por tag, upload de assets, checksum e Docker. | 🟢 |

```mermaid
stateDiagram-v2
  [*] --> test
  test --> lint
  lint --> security
  security --> build
  build --> validate
  validate --> [*]
  [*] --> release: tag v*.*.*
  release --> build_assets
  build_assets --> generate_checksums
  generate_checksums --> docker_build
  docker_build --> notify
  notify --> [*]
```

## Lacunas de estado

| ID | Lacuna | Confiança |
|---|---|---:|
| SM-GAP-001 | Não há status persistente de carteira; carteira é resultado pontual. | 🟢 |
| SM-GAP-002 | Não há estado de importação/recuperação de keystore; apenas geração e gravação. | 🟢 |
| SM-GAP-003 | Não há máquina explícita de retries de keystore apesar de `MaxRetries`/`RetryDelay` aparecerem na config do serviço. | 🟡 |
