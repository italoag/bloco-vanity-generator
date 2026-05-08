# Caso de Uso: Inicialização da CLI, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Unit pai: `cmd/bloco-eth`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] O módulo principal do binário foi criado em `cmd/bloco-eth`. 🟢
- [ ] A configuração central possui defaults, carregamento por ambiente e validação. 🟡
- [ ] A aplicação CLI expõe construtor e comando raiz executável. 🟡
- [ ] Fang está disponível ou existe equivalente capaz de executar um comando Cobra com contexto. 🟢
- [ ] Estratégia de build define ou mantém defaults para versão, commit e data de build. 🟢

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.

- [ ] T-01, Implementar criação de contexto cancelável como primeira etapa da inicialização.
  - Origem no legado: `cmd/bloco-eth/main.go:24-27`
  - Critério de pronto: `main()` chama `setupGracefulShutdown()` antes de carregar configuração e registra `defer cancel()`.
  - Confiança: 🟢

- [ ] T-02, Implementar variáveis globais de metadados do build.
  - Origem no legado: `cmd/bloco-eth/main.go:17-22`
  - Critério de pronto: existem defaults `Version = "dev"`, `GitCommit = "unknown"` e `BuildTime = "unknown"` ou equivalentes documentados.
  - Confiança: 🟢

- [ ] T-03, Carregar configuração padrão no início do bootstrap.
  - Origem no legado: `cmd/bloco-eth/main.go:29-30`
  - Critério de pronto: `config.DefaultConfig()` é chamado e seu retorno é armazenado antes de qualquer validação.
  - Confiança: 🟢

- [ ] T-04, Aplicar overrides de variáveis de ambiente.
  - Origem no legado: `cmd/bloco-eth/main.go:30-31`
  - Critério de pronto: `cfg.LoadFromEnvironment()` é chamado imediatamente após criar a configuração padrão.
  - Confiança: 🟢

- [ ] T-05, Validar configuração antes de instanciar a aplicação CLI.
  - Origem no legado: `cmd/bloco-eth/main.go:33-40`
  - Critério de pronto: `cli.NewApplication(...)` só é alcançado se `cfg.Validate()` retornar `nil`.
  - Confiança: 🟢

- [ ] T-06, Implementar falha bloqueante para configuração inválida.
  - Origem no legado: `cmd/bloco-eth/main.go:33-37`
  - Critério de pronto: erro de validação escreve `Configuration error: <erro>` em `stderr` e encerra com `os.Exit(1)`.
  - Confiança: 🟢

- [ ] T-07, Criar a aplicação CLI com configuração validada e metadados.
  - Origem no legado: `cmd/bloco-eth/main.go:39-40`
  - Critério de pronto: `cli.NewApplication(cfg, Version, GitCommit, BuildTime)` recebe exatamente a configuração validada e os metadados globais.
  - Confiança: 🟢

- [ ] T-08, Executar o comando raiz por Fang com contexto cancelável.
  - Origem no legado: `cmd/bloco-eth/main.go:42-47`
  - Critério de pronto: `fang.Execute` é chamado com `ctx` e `app.GetRootCommand()`.
  - Confiança: 🟢

- [ ] T-09, Configurar Fang para notificar sinais de interrupção e término.
  - Origem no legado: `cmd/bloco-eth/main.go:46`
  - Critério de pronto: `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` ou comportamento equivalente está presente.
  - Confiança: 🟢

- [ ] T-10, Encaminhar erros de execução para tratamento centralizado.
  - Origem no legado: `cmd/bloco-eth/main.go:47-50`
  - Critério de pronto: erro retornado por `fang.Execute` chama `handleError(err)` e encerra com código `1`.
  - Confiança: 🟢

- [ ] T-11, Implementar registro de sinais no `setupGracefulShutdown()`.
  - Origem no legado: `cmd/bloco-eth/main.go:53-58`
  - Critério de pronto: `signal.Notify` registra `os.Interrupt` e `syscall.SIGTERM` em um canal bufferizado.
  - Confiança: 🟢

- [ ] T-12, Implementar goroutine de cancelamento por sinal.
  - Origem no legado: `cmd/bloco-eth/main.go:60-64`
  - Critério de pronto: a goroutine aguarda sinal, escreve mensagem de shutdown em `stderr` e chama `cancel()`.
  - Confiança: 🟢

- [ ] T-13, Garantir injeção de metadados no build oficial.
  - Origem no legado: `Dockerfile:35-39`; `.github/workflows/ci.yaml:178-181`; `.github/workflows/release.yaml:133-136`
  - Critério de pronto: builds de CI/Docker/release sobrescrevem variáveis de versão, commit e data, ou mantêm compatibilidade explícita se os nomes forem alterados.
  - Confiança: 🟢

## Tarefas de Teste

- [ ] TT-01, Teste de ordem do bootstrap.
  - Critério de pronto: o fluxo observado segue contexto -> configuração padrão -> ambiente -> validação -> aplicação -> Fang. 🟢

- [ ] TT-02, Teste de configuração inválida.
  - Critério de pronto: ao simular erro em `cfg.Validate()`, a aplicação não instancia `cli.NewApplication` e encerra com código `1`. 🟢

- [ ] TT-03, Teste de configuração válida.
  - Critério de pronto: com config válida, `cli.NewApplication` é chamado com `cfg`, `Version`, `GitCommit` e `BuildTime`. 🟢

- [ ] TT-04, Teste de execução Fang.
  - Critério de pronto: `fang.Execute` recebe contexto, comando raiz e opção de notificação dos sinais esperados. 🟢

- [ ] TT-05, Teste de erro retornado por Fang.
  - Critério de pronto: erro simulado em `fang.Execute` aciona `handleError` e saída com código `1`. 🟢

- [ ] TT-06, Teste de cancelamento por sinal.
  - Critério de pronto: ao enviar `os.Interrupt` ou `SIGTERM`, o contexto é cancelado e a mensagem de shutdown é emitida. 🟢

- [ ] TT-07, Teste de metadados de build.
  - Critério de pronto: build com `-ldflags` altera os valores visíveis de versão, commit e data usados pela aplicação CLI. 🟢

## Tarefas de Migração de Dados

Não aplicável. A inicialização da CLI não lê nem transforma dados persistentes de negócio. 🟢

## Ordem Sugerida

1. Definir variáveis globais de metadados e imports necessários (`T-02`). 🟢
2. Implementar contexto e cancelamento por sinal (`T-01`, `T-11`, `T-12`). 🟢
3. Implementar carregamento e validação de configuração (`T-03`, `T-04`, `T-05`, `T-06`). 🟢
4. Criar aplicação CLI e executar Fang (`T-07`, `T-08`, `T-09`, `T-10`). 🟢
5. Integrar metadados em Docker/CI/release (`T-13`). 🟢
6. Executar testes de fluxo, falha e sinal (`TT-01` a `TT-07`). 🟢

## Lacunas Pendentes (🔴)

- 🔴 Não há contrato granular de exit codes por tipo de falha. O comportamento confirmado é código `1` para configuração inválida e erro de execução.
- 🟡 A duplicidade entre `signal.Notify` local e `fang.WithNotifySignal` deve ser mantida apenas se testes confirmarem que não há cancelamento ou mensagem duplicada inesperada.
- 🟡 O comportamento de erro em `LoadFromEnvironment()` não é visível no entrypoint porque a chamada não retorna erro; a validação posterior deve cobrir inconsistências de ambiente.
