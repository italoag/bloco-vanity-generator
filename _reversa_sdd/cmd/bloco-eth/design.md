# Módulo cmd/bloco-eth, Design Técnico

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Interface

O módulo expõe a interface de processo do binário `bloco-eth`. Ele não expõe API HTTP/RPC nem tipos públicos consumidos por outros pacotes; sua interface é o lifecycle do processo e as funções internas de bootstrap. 🟢

| Símbolo | Assinatura | Retorno | Observação |
|---------|-----------|---------|------------|
| `main` | `func main()` | `void` | Orquestra contexto, configuração, aplicação CLI, execução Fang e exit code. 🟢 |
| `setupGracefulShutdown` | `func setupGracefulShutdown() (context.Context, context.CancelFunc)` | `context.Context`, `context.CancelFunc` | Cria contexto cancelável e goroutine que observa `os.Interrupt`/`SIGTERM`. 🟢 |
| `handleError` | `func handleError(err error)` | `void` | Formata erros estruturados `*errors.BlocoError` ou erros genéricos em `stderr`. 🟢 |

## Entradas e Saídas

| Tipo | Item | Descrição | Confiança |
|---|---|---|---:|
| Entrada | Variáveis de ambiente | `cfg.LoadFromEnvironment()` aplica overrides de configuração; `BLOCO_DEBUG` controla stack trace em erro. | 🟢 |
| Entrada | Sinais do SO | `os.Interrupt` e `syscall.SIGTERM` cancelam o contexto. | 🟢 |
| Entrada | Flags CLI | Delegadas ao comando raiz de `internal/cli` via Fang/Cobra. | 🟢 |
| Saída | `stderr` | Erros de configuração, erros de aplicação, contexto de erro, stack trace debug e mensagem de shutdown. | 🟢 |
| Saída | Código de processo | `1` em erro de configuração ou erro retornado por `fang.Execute`; `0` por queda natural sem erro. | 🟢 |

## Fluxo Principal

1. `main()` inicia chamando `setupGracefulShutdown()` para obter `ctx` e `cancel`. 🟢 `cmd/bloco-eth/main.go:24-27`
2. `main()` carrega `config.DefaultConfig()` e aplica `cfg.LoadFromEnvironment()`. 🟢 `cmd/bloco-eth/main.go:29-31`
3. `main()` chama `cfg.Validate()` antes de criar a aplicação CLI. 🟢 `cmd/bloco-eth/main.go:33-37`
4. Se a configuração é válida, `main()` cria `cli.NewApplication(cfg, Version, GitCommit, BuildTime)`. 🟢 `cmd/bloco-eth/main.go:39-40`
5. `main()` chama `fang.Execute(ctx, app.GetRootCommand(), fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM))`. 🟢 `cmd/bloco-eth/main.go:42-47`
6. Se `fang.Execute` retorna `nil`, o processo termina sem chamada explícita a `os.Exit`. 🟢
7. Se `fang.Execute` retorna erro, `main()` chama `handleError(err)` e encerra com `os.Exit(1)`. 🟢 `cmd/bloco-eth/main.go:47-50`

## Fluxos Alternativos

- **Configuração inválida:** se `cfg.Validate()` retorna erro, o módulo escreve `Configuration error: <erro>` em `stderr` e encerra imediatamente com `os.Exit(1)`. 🟢 `cmd/bloco-eth/main.go:33-37`
- **Sinal de interrupção:** `setupGracefulShutdown()` mantém uma goroutine bloqueada em `sigChan`; ao receber sinal, escreve mensagem de shutdown em `stderr` e chama `cancel()`. 🟢 `cmd/bloco-eth/main.go:57-64`
- **Erro estruturado:** `handleError()` faz type assertion para `*errors.BlocoError`, imprime `blocoErr.Error()` e lista `Context` quando houver. 🟢 `cmd/bloco-eth/main.go:70-81`
- **Stack trace em debug:** se `BLOCO_DEBUG` estiver definido e `BlocoError.Stack` não estiver vazio, cada frame é impresso. 🟢 `cmd/bloco-eth/main.go:83-89`
- **Erro genérico:** se o erro não é `*errors.BlocoError`, `handleError()` imprime `Error: <erro>`. 🟢 `cmd/bloco-eth/main.go:90-93`

## Dependências

- `internal/config`: fornece `DefaultConfig`, `LoadFromEnvironment` e `Validate`, usados antes da criação da CLI. 🟢
- `internal/cli`: fornece `NewApplication` e `GetRootCommand`, que encapsulam Cobra/Fang. 🟢
- `pkg/errors`: fornece `BlocoError`, usado para tratamento estruturado. 🟢
- `github.com/charmbracelet/fang`: executa o comando raiz e integra notificações de sinais. 🟢
- Pacotes Go padrão `context`, `os`, `os/signal`, `syscall`, `fmt`: controlam processo, saída e sinais. 🟢

## Decisões de Design Identificadas

| Decisão | Evidência no código | Confiança |
|---------|---------------------|-----------|
| Centralizar bootstrap no pacote `main` e delegar comportamento de domínio para `internal/cli`. | `cmd/bloco-eth/main.go:39-47` | 🟢 |
| Validar configuração antes de construir a aplicação CLI. | `cmd/bloco-eth/main.go:29-40` | 🟢 |
| Usar `context.Context` como mecanismo de cancelamento cross-cutting. | `cmd/bloco-eth/main.go:24-27`, `cmd/bloco-eth/main.go:53-67` | 🟢 |
| Usar Fang sobre Cobra para execução com animações/sinais. | `cmd/bloco-eth/main.go:42-47` | 🟢 |
| Habilitar stack trace por variável de ambiente em vez de exibir sempre. | `cmd/bloco-eth/main.go:83-89` | 🟢 |
| Injetar versão, commit e build time por variáveis globais definidas no build. | `cmd/bloco-eth/main.go:17-22`; `Dockerfile:35-39`; `.github/workflows/ci.yaml:178-181` | 🟢 |

## Estado Interno

| Estado | Local | Evolução | Confiança |
|---|---|---|---:|
| `Version` | variável global `main.Version` | Default `dev`; sobrescrita por `-ldflags` em build/release. | 🟢 |
| `GitCommit` | variável global `main.GitCommit` | Default `unknown`; sobrescrita por `-ldflags`. | 🟢 |
| `BuildTime` | variável global `main.BuildTime` | Default `unknown`; sobrescrita por `-ldflags`. | 🟢 |
| `ctx` | variável local em `main` | Criado por `context.WithCancel`; cancelado em defer ou por sinal. | 🟢 |
| `sigChan` | variável local em `setupGracefulShutdown` | Recebe `os.Interrupt`/`SIGTERM` em buffer de tamanho 1. | 🟢 |

## Observabilidade

- Erros de configuração são emitidos em `stderr` com prefixo `Configuration error`. 🟢 `cmd/bloco-eth/main.go:35`
- Sinais de interrupção emitem `Received interrupt signal, shutting down gracefully...` em `stderr`. 🟢 `cmd/bloco-eth/main.go:62`
- Erros estruturados emitem mensagem principal, contexto e stack trace condicional. 🟢 `cmd/bloco-eth/main.go:70-89`
- O módulo não registra logs via `pkg/logging`; ele usa diretamente `fmt.Fprintf(os.Stderr, ...)`. 🟢

## Riscos e Lacunas

- 🟡 **Dupla gestão de sinais:** `setupGracefulShutdown()` registra `os.Interrupt`/`SIGTERM` e `fang.Execute()` também recebe `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)`. A intenção aparente é reforçar cancelamento e UX, mas a interação exata entre as duas camadas depende do comportamento do Fang.
- 🟡 **Comentário “verbose mode”:** `handleError()` sempre mostra contexto quando existe, embora o comentário diga `verbose mode`; não há checagem explícita de flag verbose nesse arquivo.
- 🔴 **Política de exit codes:** só foi confirmado exit code `1` para falhas; não há mapeamento granular por tipo de erro (`configuration`, `validation`, `timeout`, etc.).

## Contratos de Integração Interna

| Contrato | Consumidor/Fornecedor | Condição | Confiança |
|---|---|---|---:|
| `config.DefaultConfig()` deve retornar config inicializável. | `cmd/bloco-eth` consome `internal/config` | Chamado antes de `LoadFromEnvironment`. | 🟢 |
| `cfg.Validate()` deve retornar erro para configuração inválida. | `cmd/bloco-eth` consome `internal/config` | Erro causa exit code `1`. | 🟢 |
| `cli.NewApplication(...).GetRootCommand()` deve retornar comando Cobra válido. | `cmd/bloco-eth` consome `internal/cli` | Passado diretamente a `fang.Execute`. | 🟢 |
| `errors.BlocoError` deve expor `Context` e `Stack`. | `cmd/bloco-eth` consome `pkg/errors` | Campos lidos diretamente em `handleError`. | 🟢 |
