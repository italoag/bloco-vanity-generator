# Caso de Uso: Gerar Carteiras Vanity

> Spec gerada pelo Reversa Writer.  
> Unit pai: `internal/cli`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

O caso de uso `gerar-carteiras-vanity` representa o fluxo acionado pelo comando raiz `bloco-vgen`, responsável por gerar uma ou mais carteiras cujo endereço satisfaça critérios informados por flags, como prefixo, sufixo, checksum, rede e uso de mnemonic. A CLI aplica flags à configuração, valida critérios, inicializa componentes cripto e worker pool, decide entre TUI e modo texto, exibe resultados e persiste keystore/mnemonic quando habilitado. 🟢

## Responsabilidades

- Receber flags do comando raiz e interpretá-las como configuração runtime e critérios de geração. 🟢
- Validar configuração após overrides por flags. 🟢
- Validar critérios de prefixo, sufixo, checksum, mnemonic e rede. 🟢
- Criar componentes de validação cripto e worker pool por rede. 🟢
- Iniciar e finalizar worker pool de forma controlada. 🟢
- Gerar uma carteira quando `--count=1`. 🟢
- Gerar múltiplas carteiras quando `--count>1`. 🟢
- Escolher TUI quando habilitada, com progresso ativo e quiet mode desligado. 🟢
- Usar modo texto como fallback ou quando TUI não é aplicável. 🟢
- Exibir resultados de carteiras e estatísticas agregadas. 🟢
- Persistir keystore/mnemonic quando `KeyStore.Enabled` está ativo. 🟢

## Regras de Negócio

- `--threads=0` deve autodetectar a quantidade de CPUs com `runtime.NumCPU()`. 🟢
- A geração só pode iniciar após `parseFlags()` retornar sem erro e `config.Validate()` aceitar a configuração resultante. 🟢
- Os critérios de geração devem ser validados por `GenerationCriteria.Validate()` antes de iniciar o worker pool. 🟢
- `--count=1` seleciona o fluxo single; qualquer valor diferente de `1` seleciona o fluxo múltiplo. 🟢
- A TUI só deve ser usada quando `config.TUI.Enabled`, `showProgress`, `!QuietMode` e `TUIManager.ShouldUseTUI()` forem verdadeiros. 🟢
- Se a TUI falhar, a geração deve cair para modo texto. 🟢
- O modo texto não deve usar o progress manager contínuo por risco de deadlock confirmado no legado. 🟢
- Em geração múltipla texto, erro em uma carteira não deve abortar todo o lote; o loop continua para a próxima tentativa. 🟢
- Em geração múltipla, private key e mnemonic só devem ser exibidos quando quiet mode está desligado. 🟢
- No fluxo single texto legado, private key é exibida sempre e mnemonic é exibido quando presente. 🟢
- Se keystore está habilitado, a CLI deve tentar salvar artefatos após gerar carteira. 🟢
- Falha na persistência de keystore durante display é tratada como warning e não impede exibição do resultado. 🟢
- Bitcoin deve exigir mnemonic para backup e salvar somente mnemonic. 🟢
- Ethereum e Solana seguem fluxo de keystore com KDF universal, análise opcional e salvamento de arquivos. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-GCV-01 | Processar flags antes de qualquer geração. | Must | `generateWallet()` chama `parseFlags(cmd)` antes de `getGenerationCriteria(cmd)`. 🟢 |
| RF-GCV-02 | Validar critérios de geração. | Must | `getGenerationCriteria()` monta `GenerationCriteria` e retorna erro de `criteria.Validate()` quando inválido. 🟢 |
| RF-GCV-03 | Criar componentes cripto antes do worker pool. | Must | O fluxo cria `PoolManager`, `ChecksumValidator` e `AddressValidator`. 🟢 |
| RF-GCV-04 | Criar worker pool por rede. | Must | `createWorkerPool(..., criteria.Network)` é chamado antes de `Start()`. 🟢 |
| RF-GCV-05 | Garantir lifecycle do worker pool. | Must | O worker pool é iniciado com `Start()` e finalizado com `Shutdown()` deferido. 🟢 |
| RF-GCV-06 | Suportar geração de uma carteira. | Must | `count == 1` chama `generateSingleWallet(ctx, workerPool, criteria, showProgress)`. 🟢 |
| RF-GCV-07 | Suportar geração de múltiplas carteiras. | Must | `count != 1` chama `generateMultipleWallets(ctx, workerPool, criteria, count, showProgress)`. 🟢 |
| RF-GCV-08 | Selecionar TUI em fluxo single quando aplicável. | Should | `generateSingleWallet()` usa TUI apenas se as quatro condições de TUI forem verdadeiras. 🟢 |
| RF-GCV-09 | Selecionar TUI em fluxo múltiplo quando aplicável. | Should | `generateMultipleWallets()` usa TUI apenas se as quatro condições de TUI forem verdadeiras. 🟢 |
| RF-GCV-10 | Realizar fallback para texto se TUI falhar. | Should | Erro em `program.Run()` chama fluxo texto correspondente. 🟢 |
| RF-GCV-11 | Exibir resultado single. | Must | Resultado single texto imprime sucesso, address, private key, mnemonic opcional, attempts e duration. 🟢 |
| RF-GCV-12 | Exibir resumo múltiplo. | Must | Resultados múltiplos imprimem quantidade, total attempts, total duration, average speed e estatísticas quando aplicável. 🟢 |
| RF-GCV-13 | Ocultar segredos em múltiplos resultados quando quiet. | Must | `displayMultipleWalletResults()` não imprime private key/mnemonic quando `QuietMode` é verdadeiro. 🟢 |
| RF-GCV-14 | Persistir keystore/mnemonic quando habilitado. | Must | Fluxos de display ou TUI chamam `generateAndSaveKeystore`/`generateAndSaveKeystoreWithVerbose`. 🟢 |
| RF-GCV-15 | Continuar geração múltipla texto após erro individual. | Should | Erro em uma iteração imprime erro e continua o loop. 🟢 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Performance | Thread count deve ser configurável por flag e autodetectável por CPU. | `internal/cli/commands.go:971-978` | 🟢 |
| Concorrência | Geração deve ser delegada ao worker pool, preservando contexto cancelável. | `internal/cli/commands.go:151-174` | 🟢 |
| UX | TUI deve ser usada somente quando terminal e flags permitem; texto deve funcionar como fallback. | `internal/cli/commands.go:184-199`; `internal/cli/commands.go:428-443` | 🟢 |
| Resiliência | Falha de TUI deve cair para texto. | `internal/cli/commands.go:363-367`; `internal/cli/commands.go:666-670` | 🟢 |
| Segurança | Quiet mode deve ocultar private key/mnemonic em múltiplos resultados. | `internal/cli/commands.go:1438-1444` | 🟢 |
| Operabilidade | Erros principais devem ser categorizados com `pkg/errors`. | `internal/cli/commands.go:130-141`; `internal/cli/commands.go:157-161`; `internal/cli/commands.go:400-408` | 🟢 |
| Segurança operacional | O fluxo single texto expõe private key/mnemonic em stdout conforme legado. | `internal/cli/commands.go:1395-1404` | 🟢 |

> Inferido a partir do código. Validar com equipe de segurança antes de usar em ambientes automatizados.

## Critérios de Aceitação

```gherkin
Dado que o usuário executa `bloco-vgen` com flags válidas
Quando o comando raiz é processado
Então as flags devem atualizar a configuração, os critérios devem ser validados e o worker pool deve iniciar

Dado que `--count=1`
Quando a geração é executada
Então o fluxo deve gerar uma única carteira usando TUI ou texto conforme configuração

Dado que `--count=3`
Quando a geração é executada
Então o fluxo deve tentar gerar três carteiras e exibir resumo agregado

Dado que `--progress` está ativo, TUI está habilitada e quiet está desligado
Quando o terminal suporta TUI
Então a geração deve usar modelo TUI de progresso

Dado que a TUI retorna erro
Quando a geração está em andamento
Então o fluxo deve cair para modo texto correspondente

Dado que `--quiet` está ativo em geração múltipla
Quando os resultados são exibidos
Então private key e mnemonic não devem ser impressos para cada carteira

Dado que keystore está habilitado
Quando uma carteira é gerada com sucesso
Então a CLI deve tentar salvar keystore ou mnemonic conforme a rede

Dado que uma carteira Bitcoin não possui mnemonic
Quando a persistência é solicitada
Então o fluxo deve retornar erro informando que Bitcoin requer mnemonic para backup
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Parsing/validação de flags e critérios | Must | Sem isso, a geração pode executar com configuração inválida. 🟢 |
| Lifecycle do worker pool | Must | É o mecanismo central de geração concorrente. 🟢 |
| Geração single e múltipla | Must | É o objetivo principal do comando raiz. 🟢 |
| Exibição de resultado | Must | O usuário precisa receber endereço e credenciais. 🟢 |
| Persistência de keystore/mnemonic | Must | É comportamento integrado ao resultado quando habilitado. 🟢 |
| TUI | Should | Melhora UX, mas modo texto permite operação sem TUI. 🟢 |
| Fallback de TUI para texto | Should | Mantém robustez em terminais incompatíveis. 🟢 |
| Continuação após erro individual em lote | Should | Favorece sucesso parcial em geração múltipla. 🟢 |

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `internal/cli/commands.go` | `generateWallet` | 🟢 |
| `internal/cli/commands.go` | `parseFlags` | 🟢 |
| `internal/cli/commands.go` | `getGenerationCriteria` | 🟢 |
| `internal/cli/commands.go` | `createWorkerPool` | 🟢 |
| `internal/cli/commands.go` | `generateSingleWallet`, `generateSingleWalletTUI`, `generateSingleWalletText` | 🟢 |
| `internal/cli/commands.go` | `generateMultipleWallets`, `generateMultipleWalletsTUI`, `generateMultipleWalletsText` | 🟢 |
| `internal/cli/commands.go` | `displayWalletResult`, `displayMultipleWalletResults` | 🟢 |
| `internal/cli/commands.go` | `generateAndSaveKeystore`, `generateAndSaveKeystoreWithVerbose` | 🟢 |
| `internal/worker/*` | `WorkerPool` | 🟡 |
| `pkg/wallet/*` | `GenerationCriteria`, `GenerationResult` | 🟡 |
| `internal/tui/*` | Progress model e mensagens | 🟡 |
