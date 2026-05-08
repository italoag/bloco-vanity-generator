# Caso de Uso: Gerar Carteiras Vanity, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Unit pai: `internal/cli`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] Comando raiz Cobra `bloco-eth` configurado com `RunE: app.generateWallet`. 🟢
- [ ] Flags globais de geração, performance, TUI, keystore, KDF e logging registradas no comando raiz. 🟢
- [ ] `internal/config.Config` disponível e validável após mutações por flags. 🟡
- [ ] `pkg/wallet.GenerationCriteria` disponível com método `Validate()`. 🟢
- [ ] `internal/worker.WorkerPool` disponível com `Start`, `Shutdown`, `GenerateWalletWithContext` e stats collector. 🟡
- [ ] `internal/tui.TUIManager` disponível para criar progress model e verificar suporte do terminal. 🟡
- [ ] Serviços de keystore/mnemonic disponíveis no módulo `internal/crypto`. 🟡

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.

- [ ] T-GCV-01, Implementar handler do comando raiz `generateWallet`.
  - Origem no legado: `internal/cli/commands.go:126-175`
  - Critério de pronto: o handler obtém contexto, parseia flags, extrai critérios, cria componentes cripto, inicia worker pool, defere shutdown e roteia por `count`.
  - Confiança: 🟢

- [ ] T-GCV-02, Implementar parsing e validação de flags antes da geração.
  - Origem no legado: `internal/cli/commands.go:130-134`; `internal/cli/commands.go:969-1041`
  - Critério de pronto: erro em `parseFlags` é embrulhado como configuração e impede worker pool de iniciar.
  - Confiança: 🟢

- [ ] T-GCV-03, Implementar extração de critérios de geração.
  - Origem no legado: `internal/cli/commands.go:1352-1368`
  - Critério de pronto: critérios incluem `Network`, `Prefix`, `Suffix`, `IsChecksum` e `UseMnemonic`, retornando erro de `criteria.Validate()`.
  - Confiança: 🟢

- [ ] T-GCV-04, Implementar criação de componentes cripto e validação de endereço.
  - Origem no legado: `internal/cli/commands.go:146-149`
  - Critério de pronto: o fluxo cria `crypto.NewPoolManager`, `crypto.NewChecksumValidator` e `validation.NewAddressValidator` antes do worker pool.
  - Confiança: 🟢

- [ ] T-GCV-05, Implementar criação do worker pool por rede.
  - Origem no legado: `internal/cli/commands.go:119-124`; `internal/cli/commands.go:151-155`
  - Critério de pronto: worker pool usa `app.config.Worker.ThreadCount`, `app.config` e `criteria.Network`.
  - Confiança: 🟢

- [ ] T-GCV-06, Implementar lifecycle do worker pool.
  - Origem no legado: `internal/cli/commands.go:157-167`
  - Critério de pronto: `Start()` é chamado antes da geração; `Shutdown()` é deferido e warning de falha é escrito em `stderr`.
  - Confiança: 🟢

- [ ] T-GCV-07, Implementar roteamento single/múltiplo por `count`.
  - Origem no legado: `internal/cli/commands.go:169-174`
  - Critério de pronto: `count == 1` chama fluxo single; caso contrário chama fluxo múltiplo com `count` original.
  - Confiança: 🟢

- [ ] T-GCV-08, Implementar decisão TUI/texto para geração single.
  - Origem no legado: `internal/cli/commands.go:177-199`
  - Critério de pronto: TUI é usada apenas se `TUI.Enabled`, `showProgress`, `!QuietMode` e `ShouldUseTUI()` forem verdadeiros.
  - Confiança: 🟢

- [ ] T-GCV-09, Implementar fluxo single TUI.
  - Origem no legado: `internal/cli/commands.go:202-380`
  - Critério de pronto: cria stats, adapter, model, programa Bubble Tea, canais, goroutines de progresso/geração, envia resultado e retorna erro de geração se houver.
  - Confiança: 🟢

- [ ] T-GCV-10, Implementar fallback single TUI para texto.
  - Origem no legado: `internal/cli/commands.go:363-367`
  - Critério de pronto: erro em `program.Run()` chama `generateSingleWalletText(ctx, workerPool, criteria, true)`.
  - Confiança: 🟢

- [ ] T-GCV-11, Implementar fluxo single texto.
  - Origem no legado: `internal/cli/commands.go:383-418`
  - Critério de pronto: imprime cabeçalho opcional, desabilita progress manager contínuo, gera wallet via worker pool e chama `displayWalletResult`.
  - Confiança: 🟢

- [ ] T-GCV-12, Implementar decisão TUI/texto para geração múltipla.
  - Origem no legado: `internal/cli/commands.go:420-443`
  - Critério de pronto: aplica a mesma regra do fluxo single e usa texto quando TUI não é elegível.
  - Confiança: 🟢

- [ ] T-GCV-13, Implementar fluxo múltiplo TUI.
  - Origem no legado: `internal/cli/commands.go:446-679`
  - Critério de pronto: cria canais, mutex, contador, ticker de progresso, gera até `count`, envia resultados/erros para TUI e encerra ao fechar canal.
  - Confiança: 🟢

- [ ] T-GCV-14, Implementar fallback múltiplo TUI para texto.
  - Origem no legado: `internal/cli/commands.go:666-670`
  - Critério de pronto: erro em `program.Run()` chama `generateMultipleWalletsText(ctx, workerPool, criteria, count, true)`.
  - Confiança: 🟢

- [ ] T-GCV-15, Implementar fluxo múltiplo texto com sucesso parcial.
  - Origem no legado: `internal/cli/commands.go:682-733`
  - Critério de pronto: loop tenta `count` gerações, continua após erro individual, acumula resultados/tentativas e exibe resumo.
  - Confiança: 🟢

- [ ] T-GCV-16, Implementar display de resultado single.
  - Origem no legado: `internal/cli/commands.go:1395-1419`
  - Critério de pronto: imprime sucesso, address, private key, mnemonic opcional, attempts e duration; tenta keystore quando habilitado.
  - Confiança: 🟢

- [ ] T-GCV-17, Implementar display de resultados múltiplos.
  - Origem no legado: `internal/cli/commands.go:1421-1505`
  - Critério de pronto: trata lista vazia, imprime totais, oculta segredos em quiet, tenta keystore, resume erros e estatísticas.
  - Confiança: 🟢

- [ ] T-GCV-18, Integrar persistência de keystore/mnemonic ao resultado single e múltiplo.
  - Origem no legado: `internal/cli/commands.go:1406-1415`; `internal/cli/commands.go:1453-1464`; `internal/cli/commands.go:1943-2064`
  - Critério de pronto: keystore habilitado chama persistência por rede e falhas são reportadas como warning no display.
  - Confiança: 🟢

- [ ] T-GCV-19, Preservar debug de decisão TUI via `BLOCO_DEBUG`.
  - Origem no legado: `internal/cli/commands.go:188-192`; `internal/cli/commands.go:432-436`
  - Critério de pronto: quando `BLOCO_DEBUG` está definido, a decisão de TUI é impressa para single e múltiplo.
  - Confiança: 🟢

## Tarefas de Teste

- [ ] TT-GCV-01, Testar que flags são parseadas antes dos critérios.
  - Critério de pronto: erro em `parseFlags` impede chamada de `getGenerationCriteria` e criação do worker pool. 🟢

- [ ] TT-GCV-02, Testar critérios válidos e inválidos.
  - Critério de pronto: critérios válidos seguem para worker pool; inválidos retornam erro de validação. 🟢

- [ ] TT-GCV-03, Testar criação e lifecycle do worker pool.
  - Critério de pronto: `Start()` ocorre antes de gerar; `Shutdown()` é chamado mesmo após sucesso ou erro. 🟢

- [ ] TT-GCV-04, Testar roteamento por `count`.
  - Critério de pronto: `count=1` chama single; `count=2` chama múltiplo. 🟢

- [ ] TT-GCV-05, Testar seleção TUI single.
  - Critério de pronto: TUI só é selecionada quando todas as condições são verdadeiras. 🟢

- [ ] TT-GCV-06, Testar fallback TUI single.
  - Critério de pronto: erro em `program.Run()` chama fluxo texto com `showProgress=true`. 🟢

- [ ] TT-GCV-07, Testar geração single texto.
  - Critério de pronto: `GenerateWalletWithContext` é chamado uma vez e resultado é exibido. 🟢

- [ ] TT-GCV-08, Testar seleção TUI múltipla.
  - Critério de pronto: TUI múltipla só é selecionada quando todas as condições são verdadeiras. 🟢

- [ ] TT-GCV-09, Testar geração múltipla texto com erro parcial.
  - Critério de pronto: uma falha individual não aborta o lote e resultados bem-sucedidos aparecem no resumo. 🟢

- [ ] TT-GCV-10, Testar quiet mode em múltiplos resultados.
  - Critério de pronto: private key e mnemonic não aparecem quando `QuietMode=true`. 🟢

- [ ] TT-GCV-11, Testar display single com mnemonic.
  - Critério de pronto: mnemonic é exibido quando `Wallet.Mnemonic` não está vazio. 🟢

- [ ] TT-GCV-12, Testar keystore habilitado.
  - Critério de pronto: resultado single/múltiplo tenta persistência quando `KeyStore.Enabled=true`. 🟢

- [ ] TT-GCV-13, Testar warning de keystore não fatal.
  - Critério de pronto: falha de persistência imprime warning, mas o resultado da wallet ainda é exibido. 🟢

- [ ] TT-GCV-14, Testar cancelamento por contexto em TUI múltipla.
  - Critério de pronto: contexto cancelado define erro de geração e aciona shutdown sem pânico por canal fechado. 🟢

## Tarefas de Migração de Dados

Não aplicável. Este caso de uso não migra dados existentes; ele produz novos artefatos locais de carteira, keystore, password e mnemonic conforme configuração. 🟢

## Ordem Sugerida

1. Implementar handler principal e parsing/criteria (`T-GCV-01` a `T-GCV-03`). 🟢
2. Implementar componentes cripto, worker pool e lifecycle (`T-GCV-04` a `T-GCV-07`). 🟢
3. Implementar fluxo single texto e display single (`T-GCV-11`, `T-GCV-16`). 🟢
4. Implementar fluxo múltiplo texto e display múltiplo (`T-GCV-15`, `T-GCV-17`). 🟢
5. Implementar TUI single/múltipla e fallbacks (`T-GCV-08` a `T-GCV-14`, `T-GCV-19`). 🟢
6. Integrar keystore/mnemonic (`T-GCV-18`). 🟢
7. Executar testes de roteamento, erro, TUI, quiet mode, keystore e contexto (`TT-GCV-01` a `TT-GCV-14`). 🟢

## Lacunas Pendentes (🔴)

- 🟢 O progress manager em modo texto deve permanecer desabilitado se a meta for fidelidade ao legado; corrigir o deadlock é mudança comportamental.
- 🟡 `--case-sensitive` deve ser validado antes de decidir se entra em `GenerationCriteria` ou permanece apenas como flag declarada.
- 🟡 `--output` e `--format` não têm uso confirmado no display; se forem requisitos de produto, exigem implementação adicional.
- 🟡 Quiet mode é consistente em múltiplos resultados, mas não foi confirmado como aplicado ao display single.
- 🟢 O fluxo trata falha de keystore como warning; validar se backup deve ser obrigatório em ambientes regulados.
