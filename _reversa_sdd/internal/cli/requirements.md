# Módulo internal/cli

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

O módulo `internal/cli` é a camada de orquestração da aplicação de linha de comando. Ele define o comando raiz e subcomandos Cobra, registra flags, aplica flags à configuração, cria o worker pool, coordena geração de carteiras, exibe resultados por texto ou TUI, executa análise estatística, benchmark e persistência de keystore/mnemonic. 🟢

## Responsabilidades

- Construir a aplicação CLI com configuração, comando raiz e metadados de versão. 🟢
- Declarar flags globais de geração, performance, saída, keystore e logging seguro. 🟢
- Declarar subcomandos `stats`, `benchmark` e `version`. 🟢
- Converter flags em configuração runtime e critérios de geração. 🟢
- Validar critérios de geração antes de iniciar workers. 🟢
- Criar e iniciar worker pool, garantindo shutdown ao final da geração. 🟢
- Escolher entre fluxo TUI e fluxo texto conforme configuração, flag `--progress` e modo quiet. 🟢
- Exibir resultados de carteira, estatísticas e benchmark. 🟢
- Persistir keystore e mnemonic conforme rede e configuração. 🟢
- Envolver erros com categorias estruturadas quando aplicável. 🟢

## Regras de Negócio

- `--threads=0` deve autodetectar a quantidade de CPUs com `runtime.NumCPU()`. 🟢
- Valor positivo em `--threads` deve sobrescrever `config.Worker.ThreadCount`. 🟢
- `--no-keystore` deve desabilitar geração de arquivos keystore. 🟢
- `--keystore-dir`, `--keystore-kdf`, `--kdf-params` e `--security-level` só devem sobrescrever configuração quando a flag foi explicitamente alterada. 🟢
- Após aplicar flags, a configuração deve ser validada novamente. 🟢
- Critérios de geração devem incluir rede, prefixo, sufixo, checksum e uso de mnemonic. 🟢
- Critérios inválidos devem retornar erro de validação antes de iniciar geração. 🟢
- TUI só deve ser usada quando `config.TUI.Enabled`, `--progress` e `!QuietMode` forem verdadeiros, além de `TUIManager.ShouldUseTUI()`. 🟢
- Se TUI falhar em benchmark, deve haver fallback para modo texto. 🟢
- No legado, em modo texto o progress manager foi desabilitado por risco de deadlock; por decisão humana, ele deve ser corrigido e reativado como fallback textual. 🟢
- Em resultados múltiplos, private key e mnemonic só são exibidos quando `QuietMode` é falso. 🟢
- Bitcoin deve salvar somente mnemonic e não KeyStore V3. 🟢
- Ethereum e Solana seguem fluxo de keystore com KDF universal ou formato específico por rede. 🟢
- Se a carteira tem mnemonic, o mnemonic deve ser salvo após o keystore quando aplicável. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Criar uma aplicação CLI com config e metadados de build. | Must | `NewApplication(cfg, version, gitCommit, buildTime)` preenche `Application` e chama `setupCommands()`. 🟢 |
| RF-02 | Configurar comando raiz `bloco-eth` para geração padrão. | Must | `rootCmd.RunE` aponta para `app.generateWallet`. 🟢 |
| RF-03 | Registrar flags globais de geração. | Must | Flags `prefix`, `suffix`, `checksum`, `case-sensitive`, `count`, `with-mnemonic`, `network` existem. 🟢 |
| RF-04 | Registrar flags de performance e UI. | Must | Flags `threads`, `progress` e `tui` existem. 🟢 |
| RF-05 | Registrar flags de keystore e KDF. | Must | Flags `keystore-dir`, `no-keystore`, `keystore-kdf`, `kdf-params`, `kdf-analysis`, `security-level` existem. 🟢 |
| RF-06 | Registrar flags de logging seguro. | Should | Flags `log-level`, `no-logging`, `log-file`, `log-format`, `log-max-size`, `log-max-files`, `log-buffer-size` existem. 🟢 |
| RF-07 | Criar subcomandos de stats, benchmark e version. | Must | `setupCommands()` adiciona `createStatsCommand()`, `createBenchmarkCommand()` e `createVersionCommand()`. 🟢 |
| RF-08 | Aplicar flags na configuração antes da geração. | Must | `generateWallet()` chama `app.parseFlags(cmd)` antes de extrair critérios. 🟢 |
| RF-09 | Extrair e validar critérios de geração. | Must | `getGenerationCriteria()` cria `wallet.GenerationCriteria` e retorna `criteria.Validate()`. 🟢 |
| RF-10 | Criar componentes cripto e worker pool. | Must | `generateWallet()` cria `PoolManager`, checksum validator, address validator e `createWorkerPool(...)`. 🟢 |
| RF-11 | Iniciar e finalizar worker pool. | Must | `workerPool.Start()` é chamado e `Shutdown()` é deferido com warning em erro. 🟢 |
| RF-12 | Suportar geração single e multiple. | Must | `count == 1` chama `generateSingleWallet`; caso contrário chama `generateMultipleWallets`. 🟢 |
| RF-13 | Selecionar TUI ou texto para geração single. | Should | `generateSingleWallet()` usa TUI apenas se configuração, progress, quiet e terminal permitirem. 🟢 |
| RF-14 | Exibir resultado single com endereço, private key, mnemonic opcional, tentativas e duração. | Must | `displayWalletResult()` imprime esses campos. 🟢 |
| RF-15 | Exibir resultados múltiplos com estatísticas agregadas. | Should | `displayMultipleWalletResults()` imprime total attempts, duration, average speed e summary. 🟢 |
| RF-16 | Persistir keystore quando habilitado. | Must | `displayWalletResult()` e `displayMultipleWalletResults()` chamam `generateAndSaveKeystore(...)` se `KeyStore.Enabled`. 🟢 |
| RF-17 | Persistir mnemonic para Bitcoin sem KeyStore V3. | Must | `generateAndSaveKeystoreWithVerbose()` exige mnemonic para `network == bitcoin` e chama `SaveMnemonicFile`. 🟢 |
| RF-18 | Gerar keystore para Ethereum/Solana com KDF universal. | Must | Fluxo cria `UniversalKDFService`, analyzer, params, `KeyStoreService` e salva arquivos. 🟢 |
| RF-19 | Executar benchmark com TUI ou texto e fallback em falha TUI. | Should | `runBenchmarkTUI()` cai para `runBenchmarkText()` se `program.Run()` falhar. 🟢 |
| RF-21 | Tratar claims do README não implementados como documentação desatualizada. | Should | Flags como `benchmark --pattern`, `benchmark --threads` e `--optimize-for` não devem ser consideradas comportamento confirmado se não existirem no Cobra. 🟢 |
| RF-20 | Expor comando raiz para integração com Fang. | Must | `GetRootCommand()` retorna `app.rootCmd`. 🟢 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Performance | `--threads=0` deve usar todos os CPUs disponíveis por `runtime.NumCPU()`. | `internal/cli/commands.go:971-978` | 🟢 |
| Segurança | Logging pode ser desabilitado com `--no-logging` e flags de logging seguro são separadas de dados sensíveis. | `internal/cli/commands.go:109-116`, `internal/cli/commands.go:1044-1050` | 🟢 |
| Segurança | Private key e mnemonic não são exibidos em resultados múltiplos quando `QuietMode` está ativo. | `internal/cli/commands.go:1438-1444` | 🟢 |
| Segurança | Bitcoin sem mnemonic deve falhar no salvamento de backup. | `internal/cli/commands.go:1952-1956` | 🟢 |
| Disponibilidade | Worker pool deve executar shutdown deferido mesmo quando geração retorna. | `internal/cli/commands.go:157-167` | 🟢 |
| UX | Se a TUI de benchmark falhar, a aplicação deve cair para modo texto. | `internal/cli/commands.go:912-917` | 🟢 |
| Operabilidade | Configuração deve ser revalidada após alterações por flags. | `internal/cli/commands.go:1040-1041` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado que uma configuração válida e metadados de build são fornecidos
Quando `NewApplication` é chamado
Então a aplicação deve armazenar config, versão, commit e build time, e configurar o comando raiz

Dado que o usuário executa o comando raiz com flags de geração válidas
Quando `generateWallet` processa o comando
Então as flags devem atualizar a configuração, os critérios devem ser validados e o worker pool deve iniciar

Dado que `--threads=0` foi informado
Quando `parseFlags` processa as flags
Então `config.Worker.ThreadCount` deve receber `runtime.NumCPU()`

Dado que `--no-keystore` foi informado
Quando `parseFlags` processa as flags
Então `config.KeyStore.Enabled` deve ser falso

Dado que `--kdf-params` contém JSON inválido
Quando `parseFlags` processa a flag
Então a função deve retornar erro indicando parâmetros KDF inválidos

Dado que `count` é 1
Quando `generateWallet` executa
Então deve chamar o fluxo de geração de uma única carteira

Dado que `count` é maior que 1
Quando `generateWallet` executa
Então deve chamar o fluxo de geração múltipla

Dado que TUI está habilitada, `--progress` está ativo e quiet está desligado
Quando o terminal suporta TUI
Então a geração single deve usar `generateSingleWalletTUI`

Dado que a TUI de benchmark falha
Quando `program.Run()` retorna erro
Então o benchmark deve continuar em modo texto

Dado que uma carteira Bitcoin sem mnemonic é enviada para persistência
Quando `generateAndSaveKeystoreWithVerbose` é chamado
Então deve retornar erro informando que Bitcoin requer mnemonic para backup
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Construção do comando raiz e flags essenciais | Must | Sem comando raiz e flags, nenhum fluxo principal é acessível. 🟢 |
| Parsing e validação de flags/configuração | Must | Caminho crítico antes de geração, stats e benchmark. 🟢 |
| Geração single/multiple via worker pool | Must | É a funcionalidade central da aplicação. 🟢 |
| Persistência de keystore/mnemonic | Must | Ativada por padrão e integrada à exibição de resultados. 🟢 |
| KDF universal e análise de compatibilidade | Should | Importante para segurança/compatibilidade, mas depende de flags/configuração. 🟢 |
| TUI e fallback para texto | Should | Melhora UX, mas o modo texto mantém operação funcional. 🟢 |
| Benchmark detalhado | Should | Importante para performance, mas não bloqueia geração de carteira. 🟢 |
| Version command | Could | Útil para suporte e release, mas não é fluxo de domínio principal. 🟢 |

> Prioridade inferida por frequência de chamada e posição na cadeia de dependências.

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `internal/cli/commands.go` | `Application` | 🟢 |
| `internal/cli/commands.go` | `NewApplication` | 🟢 |
| `internal/cli/commands.go` | `setupCommands`, `addGlobalFlags` | 🟢 |
| `internal/cli/commands.go` | `createStatsCommand`, `createBenchmarkCommand`, `createVersionCommand` | 🟢 |
| `internal/cli/commands.go` | `generateWallet`, `generateSingleWallet`, `generateMultipleWallets` | 🟢 |
| `internal/cli/commands.go` | `parseFlags`, `parseLoggingFlags`, `parseKDFParams` | 🟢 |
| `internal/cli/commands.go` | `getGenerationCriteria` | 🟢 |
| `internal/cli/commands.go` | `displayWalletResult`, `displayMultipleWalletResults` | 🟢 |
| `internal/cli/commands.go` | `runBenchmarkText`, `executeBenchmarkWithTUI`, `executeBenchmark`, `displayBenchmarkResults` | 🟢 |
| `internal/cli/commands.go` | `generateAndSaveKeystoreWithVerbose` | 🟢 |
| `internal/config/config.go` | `Config` | 🟡 |
| `internal/worker/pool.go` | `WorkerPool`, `StatsCollector` | 🟡 |
| `internal/crypto/keystore.go` | `KeyStoreService` | 🟡 |
| `internal/crypto/kdf/*` | `UniversalKDFService`, `KDFCompatibilityAnalyzer` | 🟡 |
| `internal/tui/*` | TUI models e manager | 🟡 |
