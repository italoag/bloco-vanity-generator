# Módulo internal/cli, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] O pacote `internal/config` expõe `Config` validável e mutável por flags. 🟡
- [ ] O pacote `pkg/wallet` expõe `GenerationCriteria`, `GenerationResult`, `GenerationStats` e `BenchmarkResult`. 🟢
- [ ] O pacote `internal/worker` expõe `WorkerPool`, `StatsCollector` e geração contextual. 🟡
- [ ] O pacote `internal/crypto` expõe pool manager, checksum validator e keystore service. 🟡
- [ ] O pacote `internal/crypto/kdf` expõe serviço KDF universal e analyzer de compatibilidade. 🟡
- [ ] O pacote `internal/tui` expõe manager, modelos e mensagens Bubble Tea. 🟡
- [ ] Cobra e Bubble Tea estão disponíveis como dependências. 🟢

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.

- [ ] T-01, Implementar a struct `Application` com configuração, comando raiz e metadados de build.
  - Origem no legado: `internal/cli/commands.go:28-35`
  - Critério de pronto: a aplicação mantém `config`, `rootCmd`, `version`, `gitCommit` e `buildTime` como estado interno.
  - Confiança: 🟢

- [ ] T-02, Implementar `NewApplication` chamando setup de comandos.
  - Origem no legado: `internal/cli/commands.go:37-48`
  - Critério de pronto: o construtor inicializa a struct, chama `setupCommands()` e retorna uma aplicação pronta para uso.
  - Confiança: 🟢

- [ ] T-03, Implementar execução contextual e exposição do comando raiz.
  - Origem no legado: `internal/cli/commands.go:50-52`; `internal/cli/commands.go:1917-1920`
  - Critério de pronto: `ExecuteContext(ctx)` delega para `rootCmd.ExecuteContext(ctx)` e `GetRootCommand()` retorna `rootCmd`.
  - Confiança: 🟢

- [ ] T-04, Implementar `setupCommands()` com comando raiz de geração e subcomandos.
  - Origem no legado: `internal/cli/commands.go:55-75`
  - Critério de pronto: root command `bloco-vgen` possui `RunE: app.generateWallet`, versão formatada e subcomandos `stats`, `benchmark`, `version`.
  - Confiança: 🟢

- [ ] T-05, Registrar flags globais de geração, performance, saída, keystore e logging.
  - Origem no legado: `internal/cli/commands.go:77-117`
  - Critério de pronto: todas as flags documentadas em `requirements.md` existem com defaults equivalentes aos do legado.
  - Confiança: 🟢

- [ ] T-06, Implementar criação de worker pool configurado por rede.
  - Origem no legado: `internal/cli/commands.go:119-124`
  - Critério de pronto: `createWorkerPool(...)` cria pool por `app.config.Worker.ThreadCount`, `app.config` e `network`.
  - Confiança: 🟢

- [ ] T-07, Implementar handler principal `generateWallet`.
  - Origem no legado: `internal/cli/commands.go:126-175`
  - Critério de pronto: o handler parseia flags, extrai critérios, cria validadores/worker pool, inicia pool, defere shutdown e roteia single/multiple por `count`.
  - Confiança: 🟢

- [ ] T-08, Implementar seleção TUI/texto para geração single.
  - Origem no legado: `internal/cli/commands.go:177-200`
  - Critério de pronto: TUI é usada somente quando `TUI.Enabled`, `showProgress`, `!QuietMode` e `ShouldUseTUI()` forem verdadeiros; caso contrário usa texto.
  - Confiança: 🟢

- [ ] T-09, Implementar geração single com TUI.
  - Origem no legado: `internal/cli/commands.go:202-380`
  - Critério de pronto: cria stats, adapter, progress model, programa Bubble Tea, goroutines de atualização/geração, salva keystore em modo silencioso e cai para texto se TUI falhar.
  - Confiança: 🟢

- [ ] T-10, Implementar geração single em modo texto.
  - Origem no legado: `internal/cli/commands.go:383-418`
  - Critério de pronto: imprime cabeçalho de progresso quando aplicável, chama `GenerateWalletWithContext`, envolve erro como geração e exibe resultado.
  - Confiança: 🟢

- [ ] T-11, Implementar seleção TUI/texto para múltiplas carteiras.
  - Origem no legado: `internal/cli/commands.go:420-444`
  - Critério de pronto: aplica a mesma regra de TUI do fluxo single e cai para modo texto quando necessário.
  - Confiança: 🟢

- [ ] T-12, Implementar geração múltipla com TUI.
  - Origem no legado: `internal/cli/commands.go:446-679`
  - Critério de pronto: cria canais, mutex de contagem, ticker de progresso, gera carteiras sequencialmente, envia resultados para TUI, salva keystore silencioso e encerra ao completar.
  - Confiança: 🟢

- [ ] T-13, Implementar geração múltipla em modo texto.
  - Origem no legado: `internal/cli/commands.go:682-733`
  - Critério de pronto: gera até `count` carteiras, continua após erro individual, acumula tentativas/duração e exibe resumo.
  - Confiança: 🟢

- [ ] T-14, Implementar subcomando `stats` com TUI/text fallback.
  - Origem no legado: `internal/cli/commands.go:736-833`
  - Critério de pronto: `stats` aceita `prefix`, `suffix`, `checksum`, calcula dificuldade/probabilidade e exibe via TUI ou texto.
  - Confiança: 🟢

- [ ] T-15, Implementar subcomando `benchmark` com TUI/text fallback.
  - Origem no legado: `internal/cli/commands.go:836-951`
  - Critério de pronto: `benchmark` aceita `attempts`, `duration`, `detailed`, cria worker pool, roda TUI ou texto e garante shutdown.
  - Confiança: 🟢

- [ ] T-16, Implementar subcomando `version`.
  - Origem no legado: `internal/cli/commands.go:954-965`
  - Critério de pronto: comando imprime versão, commit e build time.
  - Confiança: 🟢

- [ ] T-17, Implementar parsing de flags e validação de configuração.
  - Origem no legado: `internal/cli/commands.go:969-1042`
  - Critério de pronto: flags alteram config conforme regras, `--threads=0` usa `runtime.NumCPU()`, KDF/logging são parseados e `config.Validate()` é chamado ao final.
  - Confiança: 🟢

- [ ] T-18, Implementar parsing de logging flags.
  - Origem no legado: `internal/cli/commands.go:1044-1095`
  - Critério de pronto: `--no-logging` desabilita logging e demais flags de logging alteram level, format, file, max size, max files e buffer size quando explicitamente modificadas.
  - Confiança: 🟢

- [ ] T-19, Implementar parsing e validação de parâmetros KDF em JSON.
  - Origem no legado: `internal/cli/commands.go:1097-1231`
  - Critério de pronto: JSON inválido falha, KDF não suportado falha, scrypt valida `n/r/p/dklen`, PBKDF2 valida `c/dklen/prf`.
  - Confiança: 🟢

- [ ] T-20, Implementar helpers de KDF analysis e security level.
  - Origem no legado: `internal/cli/commands.go:1233-1349`
  - Critério de pronto: parâmetros completos são extraídos do keystore, security level textual é convertido e compatibility report é exibido quando habilitado.
  - Confiança: 🟢

- [ ] T-21, Implementar extração de critérios de geração e helpers de formatação/cálculo.
  - Origem no legado: `internal/cli/commands.go:1352-1393`
  - Critério de pronto: criteria inclui network, prefix, suffix, checksum, mnemonic e retorna `criteria.Validate()`; helpers delegam para `pkg/utils`.
  - Confiança: 🟢

- [ ] T-22, Implementar exibição de resultado single.
  - Origem no legado: `internal/cli/commands.go:1395-1419`
  - Critério de pronto: imprime address, private key, mnemonic opcional, attempts, duration e warning de keystore não fatal.
  - Confiança: 🟢

- [ ] T-23, Implementar exibição de resultados múltiplos.
  - Origem no legado: `internal/cli/commands.go:1421-1505`
  - Critério de pronto: trata lista vazia, imprime totais, oculta segredo em quiet mode, salva keystores, mostra estatísticas e resumo de erros.
  - Confiança: 🟢

- [ ] T-24, Implementar execução de benchmark e montagem de `BenchmarkResult`.
  - Origem no legado: `internal/cli/commands.go:1507-1807`
  - Critério de pronto: amostra stats por ticker, calcula speed samples, duração, eficiência e retorna `BenchmarkResult` com métricas finais.
  - Confiança: 🟢

- [ ] T-25, Implementar exibição de resultados de benchmark.
  - Origem no legado: `internal/cli/commands.go:1809-1914`
  - Critério de pronto: imprime métricas básicas, multi-threading, detalhes opcionais, estatística de velocidade e análise de performance.
  - Confiança: 🟢

- [ ] T-26, Implementar adapter de stats para TUI.
  - Origem no legado: `internal/cli/commands.go:1922-1941`
  - Critério de pronto: adapter converte stats agregados e performance metrics em `tui.ThreadMetrics` e peak speed.
  - Confiança: 🟢

- [ ] T-27, Implementar persistência de keystore e mnemonic por rede.
  - Origem no legado: `internal/cli/commands.go:1943-2064`
  - Critério de pronto: Bitcoin salva mnemonic obrigatório; Ethereum/Solana geram keystore com KDF universal, análise opcional, save de keystore/password e mnemonic se existir.
  - Confiança: 🟢

## Tarefas de Teste

- [ ] TT-01, Testar construção da aplicação e comando raiz.
  - Critério de pronto: `NewApplication` inicializa root command com flags e subcomandos esperados. 🟢

- [ ] TT-02, Testar parsing de flags principais.
  - Critério de pronto: `--threads`, `--quiet`, `--verbose`, `--tui`, `--no-keystore`, KDF e logging alteram `config` conforme esperado. 🟢

- [ ] TT-03, Testar extração e validação de critérios.
  - Critério de pronto: prefix/suffix/checksum/network/mnemonic geram `GenerationCriteria` válido ou erro de validação. 🟢

- [ ] TT-04, Testar geração single em texto e TUI fallback.
  - Critério de pronto: TUI é usada quando suportada e modo texto é usado quando TUI falha ou está desabilitada. 🟢

- [ ] TT-05, Testar geração múltipla com erro parcial.
  - Critério de pronto: erro em uma carteira não aborta todo o fluxo texto e resultados bem-sucedidos são exibidos. 🟢

- [ ] TT-06, Testar `stats` em modo texto e TUI fallback.
  - Critério de pronto: dificuldade, probability50 e estimativas são exibidas corretamente. 🟢

- [ ] TT-07, Testar benchmark com duração e limite de tentativas.
  - Critério de pronto: `BenchmarkResult` contém duração, samples, average/min/max speed e métricas de thread. 🟡

- [ ] TT-08, Testar KDF params JSON.
  - Critério de pronto: JSON inválido, scrypt inválido, PBKDF2 inválido e KDF não suportado retornam erro adequado. 🟢

- [ ] TT-09, Testar persistência Bitcoin.
  - Critério de pronto: carteira Bitcoin sem mnemonic falha; com mnemonic chama `SaveMnemonicFile` sem KeyStore V3. 🟢

- [ ] TT-10, Testar persistência Ethereum/Solana.
  - Critério de pronto: fluxo cria KDF service, gera keystore, salva arquivos e salva mnemonic quando presente. 🟢

- [ ] TT-11, Testar quiet mode em múltiplos resultados.
  - Critério de pronto: private key e mnemonic não aparecem no output quando `QuietMode` é verdadeiro. 🟢

- [ ] TT-12, Testar warnings não fatais de keystore.
  - Critério de pronto: falha de keystore é exibida como warning e não impede exibição da wallet. 🟢

## Tarefas de Migração de Dados

Não aplicável. O módulo `internal/cli` não executa migração de banco de dados. Ele cria artefatos locais de keystore, password, mnemonic e logs conforme configuração. 🟢

## Ordem Sugerida

1. Implementar `Application`, construtor, comando raiz e flags (`T-01` a `T-05`). 🟢
2. Implementar parsing de flags, KDF, logging e critérios (`T-17` a `T-21`). 🟢
3. Implementar worker pool e geração single/multiple em texto (`T-06`, `T-07`, `T-10`, `T-13`). 🟢
4. Implementar TUI e adapters (`T-08`, `T-09`, `T-11`, `T-12`, `T-26`). 🟢
5. Implementar stats, benchmark e version (`T-14` a `T-16`, `T-24`, `T-25`). 🟢
6. Implementar exibição de resultados e persistência de keystore/mnemonic (`T-22`, `T-23`, `T-27`). 🟢
7. Executar testes funcionais, de erro, de segurança de saída e de integração com módulos dependentes (`TT-01` a `TT-12`). 🟢

## Lacunas Pendentes (🔴)

- 🟢 O progress manager em texto está explicitamente desabilitado por deadlock; uma reimplementação fiel deve manter o comportamento ou resolver a causa com testes concorrentes.
- 🟢 O benchmark possui TODO para submissão real de `WorkItem` ao pool; reimplementar como está preserva a lacuna, corrigir altera comportamento legado.
- 🟡 A flag `case-sensitive` é registrada, mas não foi confirmada na extração de critérios do módulo; validar se deve ser conectada a `GenerationCriteria` ou removida.
- 🟡 As flags `output` e `format` são registradas, mas uso efetivo não foi confirmado nos trechos analisados; validar comportamento esperado antes de reimplementar saída para arquivo/JSON/CSV.
- 🔴 Não há contrato confirmado para compatibilidade completa da persistência Solana; a unit depende dos riscos já documentados em `internal/crypto`.
