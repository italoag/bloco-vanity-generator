# Módulo cmd/bloco-vgen, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] Toolchain Go compatível com o projeto está disponível. 🟢
- [ ] Pacotes `internal/config`, `internal/cli` e `pkg/errors` foram reimplementados ou estão disponíveis para importação. 🟢
- [ ] Dependência `github.com/charmbracelet/fang` ou equivalente funcional está disponível. 🟢
- [ ] O comando raiz retornado por `Application.GetRootCommand()` está implementado. 🟢
- [ ] Variáveis de build `Version`, `GitCommit` e `BuildTime` têm defaults e podem ser sobrescritas por build flags. 🟢

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.

- [ ] T-01, Criar o pacote `main` do binário `bloco-vgen` com imports de contexto, saída padrão de erro, sinais do SO, configuração, CLI, erros estruturados e Fang.
  - Origem no legado: `cmd/bloco-vgen/main.go:1-15`
  - Critério de pronto: o entrypoint compila e consegue referenciar `internal/config`, `internal/cli`, `pkg/errors` e `fang.Execute`.
  - Confiança: 🟢

- [ ] T-02, Declarar variáveis globais de versão com defaults seguros para ambiente de desenvolvimento.
  - Origem no legado: `cmd/bloco-vgen/main.go:17-22`
  - Critério de pronto: `Version` tem default `dev`, `GitCommit` tem default `unknown` e `BuildTime` tem default `unknown`.
  - Confiança: 🟢

- [ ] T-03, Implementar `main()` iniciando por `setupGracefulShutdown()` e garantindo `defer cancel()`.
  - Origem no legado: `cmd/bloco-vgen/main.go:24-27`
  - Critério de pronto: o contexto cancelável é criado antes de qualquer carregamento de configuração e o cancelamento é deferido.
  - Confiança: 🟢

- [ ] T-04, Carregar configuração padrão e aplicar variáveis de ambiente antes da validação.
  - Origem no legado: `cmd/bloco-vgen/main.go:29-31`
  - Critério de pronto: `config.DefaultConfig()` é chamado e o retorno recebe `LoadFromEnvironment()` antes de `Validate()`.
  - Confiança: 🟢

- [ ] T-05, Validar configuração e encerrar com erro operacional quando inválida.
  - Origem no legado: `cmd/bloco-vgen/main.go:33-37`
  - Critério de pronto: se `cfg.Validate()` retorna erro, `stderr` contém `Configuration error: <erro>` e o processo encerra com código `1`.
  - Confiança: 🟢

- [ ] T-06, Instanciar a aplicação CLI com configuração e metadados de build.
  - Origem no legado: `cmd/bloco-vgen/main.go:39-40`
  - Critério de pronto: `cli.NewApplication(cfg, Version, GitCommit, BuildTime)` é chamado somente depois de configuração válida.
  - Confiança: 🟢

- [ ] T-07, Executar o comando raiz via Fang com contexto e notificação de sinais.
  - Origem no legado: `cmd/bloco-vgen/main.go:42-47`
  - Critério de pronto: `fang.Execute` recebe `ctx`, `app.GetRootCommand()` e `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)`.
  - Confiança: 🟢

- [ ] T-08, Tratar erro de execução CLI com formatação centralizada e exit code não-zero.
  - Origem no legado: `cmd/bloco-vgen/main.go:47-50`
  - Critério de pronto: se `fang.Execute` retorna erro, `handleError(err)` é invocado antes de `os.Exit(1)`.
  - Confiança: 🟢

- [ ] T-09, Implementar `setupGracefulShutdown()` com `context.WithCancel` e canal de sinal bufferizado.
  - Origem no legado: `cmd/bloco-vgen/main.go:53-58`
  - Critério de pronto: a função retorna `context.Context` e `context.CancelFunc`, cria `sigChan` com buffer `1` e registra `os.Interrupt` e `syscall.SIGTERM`.
  - Confiança: 🟢

- [ ] T-10, Implementar goroutine de desligamento gracioso baseada em sinal.
  - Origem no legado: `cmd/bloco-vgen/main.go:60-64`
  - Critério de pronto: a goroutine aguarda um sinal, escreve mensagem de shutdown em `stderr` e chama `cancel()`.
  - Confiança: 🟢

- [ ] T-11, Implementar `handleError()` com suporte a `*errors.BlocoError`.
  - Origem no legado: `cmd/bloco-vgen/main.go:69-81`
  - Critério de pronto: erros estruturados são identificados por type assertion, a mensagem principal é impressa e todo `Context` é listado quando presente.
  - Confiança: 🟢

- [ ] T-12, Implementar exibição condicional de stack trace por `BLOCO_DEBUG`.
  - Origem no legado: `cmd/bloco-vgen/main.go:83-89`
  - Critério de pronto: stack trace só aparece quando `os.Getenv("BLOCO_DEBUG") != ""` e o erro estruturado possui frames.
  - Confiança: 🟢

- [ ] T-13, Implementar fallback de erro genérico.
  - Origem no legado: `cmd/bloco-vgen/main.go:90-93`
  - Critério de pronto: erros que não são `*errors.BlocoError` são impressos como `Error: <erro>`.
  - Confiança: 🟢

- [ ] T-14, Configurar build para injetar versão, commit e data de build.
  - Origem no legado: `Dockerfile:35-39`; `.github/workflows/ci.yaml:178-181`; `.github/workflows/release.yaml:133-136`
  - Critério de pronto: build oficial passa `-ldflags` para sobrescrever `main.version`, `main.commit` e `main.date` ou equivalentes compatíveis com os nomes finais adotados.
  - Confiança: 🟢

## Tarefas de Teste

- [ ] TT-01, Testar happy path de inicialização com configuração válida.
  - Critério de pronto: com `config.Validate()` sem erro e comando raiz stubado, a execução alcança `fang.Execute` com o contexto e comando corretos. 🟢

- [ ] TT-02, Testar falha de configuração.
  - Critério de pronto: quando `cfg.Validate()` retorna erro, o processo emite `Configuration error` em `stderr` e finaliza com código `1`. 🟢

- [ ] TT-03, Testar cancelamento por sinal.
  - Critério de pronto: ao simular `os.Interrupt` ou `SIGTERM`, o contexto retornado por `setupGracefulShutdown()` é cancelado e a mensagem de shutdown é emitida. 🟢

- [ ] TT-04, Testar erro estruturado com contexto.
  - Critério de pronto: `handleError()` imprime a mensagem de `BlocoError` e cada item do `Context`. 🟢

- [ ] TT-05, Testar stack trace condicional.
  - Critério de pronto: stack trace é omitida sem `BLOCO_DEBUG` e exibida quando `BLOCO_DEBUG` está definido. 🟢

- [ ] TT-06, Testar erro genérico.
  - Critério de pronto: `handleError(errors.New("x"))` imprime `Error: x`. 🟢

## Tarefas de Migração de Dados

Não aplicável. O módulo `cmd/bloco-vgen` não manipula banco de dados nem migração de dados persistentes. 🟢

## Ordem Sugerida

1. Implementar variáveis de build e imports básicos (`T-01`, `T-02`). 🟢
2. Implementar bootstrap de contexto e configuração (`T-03`, `T-04`, `T-05`). 🟢
3. Integrar aplicação CLI e Fang (`T-06`, `T-07`, `T-08`). 🟢
4. Implementar desligamento gracioso (`T-09`, `T-10`). 🟢
5. Implementar tratamento de erros (`T-11`, `T-12`, `T-13`). 🟢
6. Configurar injeção de metadados no build (`T-14`). 🟢
7. Executar testes de processo, sinais e erro (`TT-01` a `TT-06`). 🟢

## Lacunas Pendentes (🔴)

- 🔴 Não há mapeamento granular confirmado de exit codes por categoria de erro; o legado usa `1` para falhas de configuração e falhas de execução CLI.
- 🟡 A interação entre `setupGracefulShutdown()` e `fang.WithNotifySignal(...)` deve ser validada em testes de integração, porque ambos observam sinais de interrupção.
- 🟡 O comentário do legado menciona contexto em “verbose mode”, mas `handleError()` imprime contexto sempre que existe; confirmar se esse comportamento deve ser mantido ou condicionado a verbose.
