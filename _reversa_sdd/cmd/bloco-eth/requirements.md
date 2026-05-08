# Módulo cmd/bloco-eth

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

O módulo `cmd/bloco-eth` é o ponto de entrada executável do sistema. Ele inicializa o contexto cancelável, carrega e valida configuração, instancia a aplicação CLI e delega a execução do comando raiz ao Fang/Cobra. 🟢

## Responsabilidades

- Inicializar o contexto de execução com suporte a cancelamento por `os.Interrupt` e `SIGTERM`. 🟢
- Carregar configuração padrão e aplicar variáveis de ambiente antes de executar qualquer comando. 🟢
- Bloquear a execução quando a configuração é inválida, escrevendo erro em `stderr` e retornando exit code `1`. 🟢
- Criar a aplicação CLI com informações de versão, commit e build time injetadas no build. 🟢
- Executar o comando raiz com `fang.Execute` e notificação de sinais. 🟢
- Renderizar erros estruturados `BlocoError` com contexto e stack trace opcional por `BLOCO_DEBUG`. 🟢

## Regras de Negócio

- Configuração inválida deve impedir a inicialização da CLI e finalizar o processo com `os.Exit(1)`. 🟢
- Sinais `os.Interrupt` e `SIGTERM` devem cancelar o contexto de execução para desligamento gracioso. 🟢
- Erros do tipo `*errors.BlocoError` devem exibir mensagem estruturada e contexto quando disponível. 🟢
- Stack trace só deve ser exibido quando a variável de ambiente `BLOCO_DEBUG` estiver definida. 🟢
- Erros genéricos devem ser exibidos como `Error: <erro>` em `stderr`. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Inicializar um `context.Context` cancelável antes de carregar configuração. | Must | Ao iniciar o binário, `setupGracefulShutdown()` é chamado antes de `config.DefaultConfig()`. 🟢 |
| RF-02 | Carregar configuração padrão e aplicar variáveis de ambiente. | Must | A inicialização chama `config.DefaultConfig()` e em seguida `cfg.LoadFromEnvironment()`. 🟢 |
| RF-03 | Validar a configuração antes de criar a aplicação CLI. | Must | Se `cfg.Validate()` retornar erro, o processo escreve `Configuration error` em `stderr` e sai com código `1`. 🟢 |
| RF-04 | Criar a aplicação CLI com metadados de versão. | Must | `cli.NewApplication(cfg, Version, GitCommit, BuildTime)` recebe os valores globais definidos por build flags. 🟢 |
| RF-05 | Executar o comando raiz com Fang e sinais de interrupção. | Must | `fang.Execute(ctx, app.GetRootCommand(), fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM))` é invocado. 🟢 |
| RF-06 | Tratar erros retornados pela execução CLI. | Must | Se `fang.Execute` retornar erro, `handleError(err)` é chamado e o processo sai com código `1`. 🟢 |
| RF-07 | Exibir contexto de `BlocoError` quando existir. | Should | Para `BlocoError` com `Context`, cada chave/valor é escrito em `stderr`. 🟢 |
| RF-08 | Exibir stack trace somente em modo debug. | Could | Para `BlocoError` com stack, a stack só aparece quando `BLOCO_DEBUG` não está vazio. 🟢 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Disponibilidade | A aplicação deve responder a sinais de interrupção com cancelamento gracioso em vez de depender apenas de término abrupto. | `cmd/bloco-eth/main.go:53-67` | 🟢 |
| Operabilidade | Falhas de configuração devem ser reportadas em `stderr` com exit code não-zero. | `cmd/bloco-eth/main.go:33-37` | 🟢 |
| Observabilidade | Erros estruturados devem expor contexto e, em debug, stack trace. | `cmd/bloco-eth/main.go:69-94` | 🟢 |
| Build/Release | Versão, commit e data de build devem ser injetáveis por flags de build. | `cmd/bloco-eth/main.go:17-22`; `Dockerfile:35-39`; `.github/workflows/ci.yaml:178-181` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado que o binário `bloco-eth` é iniciado com configuração válida
Quando a função `main` executa
Então a aplicação deve criar a CLI e delegar a execução ao Fang com o comando raiz

Dado que a configuração carregada é inválida
Quando `cfg.Validate()` retorna erro
Então o processo deve escrever `Configuration error` em `stderr` e encerrar com código 1

Dado que o usuário envia SIGTERM ou interrupção
Quando `setupGracefulShutdown` recebe o sinal
Então o contexto de execução deve ser cancelado e a mensagem de desligamento gracioso deve ser emitida em `stderr`

Dado que `fang.Execute` retorna um `BlocoError` com contexto
Quando `handleError` processa o erro
Então a mensagem do erro e os pares de contexto devem ser escritos em `stderr`

Dado que `fang.Execute` retorna um `BlocoError` com stack trace
Quando `BLOCO_DEBUG` está vazio
Então a stack trace não deve ser exibida

Dado que `fang.Execute` retorna um erro genérico
Quando `handleError` processa o erro
Então `stderr` deve conter `Error: <erro>`
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Inicialização, configuração e validação antes da CLI | Must | Caminho crítico; sem ele nenhum comando pode ser executado. 🟢 |
| Execução via Fang/Cobra | Must | É o mecanismo central de despacho do binário. 🟢 |
| Cancelamento por sinais | Must | Necessário para operações longas de geração de carteira. 🟢 |
| Tratamento de erro estruturado | Should | Melhora diagnóstico e preserva contexto, mas erro genérico tem fallback. 🟢 |
| Stack trace condicional por `BLOCO_DEBUG` | Could | Útil para debug, mas não é necessária ao fluxo normal. 🟢 |

> Prioridade inferida por frequência de chamada e posição na cadeia de dependências.

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `cmd/bloco-eth/main.go` | `main` | 🟢 |
| `cmd/bloco-eth/main.go` | `setupGracefulShutdown` | 🟢 |
| `cmd/bloco-eth/main.go` | `handleError` | 🟢 |
| `internal/config/config.go` | `DefaultConfig`, `LoadFromEnvironment`, `Validate` | 🟡 |
| `internal/cli/commands.go` | `NewApplication`, `GetRootCommand` | 🟡 |
| `pkg/errors/types.go` | `BlocoError` | 🟡 |
| `Dockerfile` | build flags `Version`, `GitCommit`, `BuildTime` | 🟢 |
| `.github/workflows/ci.yaml` | build com `-ldflags` | 🟢 |
