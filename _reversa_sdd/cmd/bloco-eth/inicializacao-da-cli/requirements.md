# Caso de Uso: Inicialização da CLI

> Spec gerada pelo Reversa Writer.  
> Unit pai: `cmd/bloco-eth`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

A inicialização da CLI é o fluxo que prepara o processo `bloco-eth` para executar qualquer comando. O fluxo cria contexto cancelável, carrega configuração, valida limites operacionais, instancia a aplicação CLI e entrega a execução ao Fang/Cobra. 🟢

## Responsabilidades

- Criar um contexto de execução com cancelamento gracioso por sinal do sistema operacional. 🟢
- Carregar configuração padrão e aplicar overrides por variáveis de ambiente. 🟢
- Validar a configuração antes de criar o roteador de comandos. 🟢
- Encerrar a aplicação com erro claro quando a configuração é inválida. 🟢
- Injetar metadados de versão, commit e build time na aplicação CLI. 🟢
- Executar o comando raiz com Fang e propagar falhas para tratamento centralizado. 🟢

## Regras de Negócio

- A CLI não deve iniciar comandos quando a configuração é inválida. 🟢
- A configuração padrão deve ser criada antes de aplicar variáveis de ambiente. 🟢
- A aplicação CLI só deve ser instanciada após validação bem-sucedida da configuração. 🟢
- A execução do comando raiz deve receber um contexto cancelável. 🟢
- Interrupções por `os.Interrupt` e `SIGTERM` devem acionar cancelamento gracioso. 🟢
- Qualquer erro retornado por `fang.Execute` deve finalizar o processo com código `1`. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Criar contexto cancelável no início do processo. | Must | `setupGracefulShutdown()` é chamado no início de `main()` e seu `cancel` é deferido. 🟢 |
| RF-02 | Carregar configuração padrão. | Must | `config.DefaultConfig()` é chamado antes de qualquer validação. 🟢 |
| RF-03 | Aplicar variáveis de ambiente à configuração. | Must | `cfg.LoadFromEnvironment()` é chamado após `DefaultConfig()` e antes de `Validate()`. 🟢 |
| RF-04 | Validar configuração antes da criação da aplicação. | Must | `cfg.Validate()` é executado antes de `cli.NewApplication(...)`. 🟢 |
| RF-05 | Bloquear inicialização em configuração inválida. | Must | Erro de `cfg.Validate()` escreve `Configuration error` em `stderr` e chama `os.Exit(1)`. 🟢 |
| RF-06 | Criar aplicação CLI com metadados de build. | Must | `cli.NewApplication(cfg, Version, GitCommit, BuildTime)` é chamado com os valores globais. 🟢 |
| RF-07 | Executar comando raiz via Fang. | Must | `fang.Execute` recebe o contexto, `app.GetRootCommand()` e `fang.WithNotifySignal(...)`. 🟢 |
| RF-08 | Tratar erro de execução. | Must | Erro de `fang.Execute` chama `handleError(err)` e encerra com código `1`. 🟢 |
| RF-09 | Cancelar contexto ao receber sinal. | Must | A goroutine em `setupGracefulShutdown()` chama `cancel()` ao receber sinal em `sigChan`. 🟢 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Operabilidade | Erros de configuração devem ser visíveis em `stderr` antes do encerramento. | `cmd/bloco-eth/main.go:33-37` | 🟢 |
| Disponibilidade | Operações longas devem poder ser canceladas por contexto quando o processo recebe sinal. | `cmd/bloco-eth/main.go:24-27`, `cmd/bloco-eth/main.go:53-67` | 🟢 |
| Build/Release | Metadados de versão devem ter defaults e serem substituíveis no build. | `cmd/bloco-eth/main.go:17-22`; `Dockerfile:35-39` | 🟢 |
| UX de terminal | A execução deve usar Fang para comando raiz com suporte a sinais. | `cmd/bloco-eth/main.go:42-47` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado que o processo `bloco-eth` é iniciado
Quando `main()` começa a execução
Então um contexto cancelável deve ser criado antes do carregamento da configuração

Dado que a configuração padrão foi criada
Quando variáveis de ambiente existem
Então `LoadFromEnvironment()` deve aplicar os overrides antes da validação

Dado que `cfg.Validate()` retorna erro
Quando a inicialização tenta prosseguir
Então a aplicação deve escrever `Configuration error` em `stderr` e encerrar com código 1

Dado que `cfg.Validate()` retorna sucesso
Quando a inicialização continua
Então `cli.NewApplication` deve receber a configuração validada e os metadados de build

Dado que a aplicação CLI foi criada
Quando o comando raiz é executado
Então `fang.Execute` deve receber o contexto cancelável e `app.GetRootCommand()`

Dado que o sistema operacional envia SIGTERM ou interrupção
Quando a goroutine de sinais recebe o evento
Então a mensagem de desligamento gracioso deve ser emitida e o contexto deve ser cancelado
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Criar contexto cancelável | Must | Base de cancelamento para todos os comandos e operações longas. 🟢 |
| Carregar e validar configuração | Must | Sem configuração válida, o sistema não deve executar comandos. 🟢 |
| Criar aplicação CLI | Must | Necessário para disponibilizar comando raiz e subcomandos. 🟢 |
| Executar via Fang | Must | É o caminho real de execução do binário. 🟢 |
| Injetar metadados de build | Should | Importante para versionamento e suporte, mas não altera a execução funcional do comando. 🟢 |
| Mensagem de shutdown gracioso | Should | Melhora UX operacional durante interrupção. 🟢 |

> Prioridade inferida por frequência de chamada e posição na cadeia de dependências.

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `cmd/bloco-eth/main.go` | `main` | 🟢 |
| `cmd/bloco-eth/main.go` | `setupGracefulShutdown` | 🟢 |
| `cmd/bloco-eth/main.go` | variáveis `Version`, `GitCommit`, `BuildTime` | 🟢 |
| `internal/config/config.go` | `DefaultConfig`, `LoadFromEnvironment`, `Validate` | 🟡 |
| `internal/cli/commands.go` | `NewApplication`, `GetRootCommand` | 🟡 |
| `Dockerfile` | `-ldflags` para metadados de build | 🟢 |
| `.github/workflows/ci.yaml` | build do binário com metadados | 🟢 |
