# Caso de Uso: Salvar Keystore, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Unit pai: `internal/cli`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] O fluxo de geração de carteira já produz `wallet.Wallet` com `Address`, `PrivateKey`, `Network` e `Mnemonic` opcional. 🟢
- [ ] A configuração `app.config.KeyStore` está disponível com `Enabled`, `OutputDir`, `KDFAlgorithm`, `KDFParams`, `SecurityLevel` e `ShowAnalysis`. 🟢
- [ ] A configuração `app.config.CLI` está disponível com `VerboseOutput` e `QuietMode`. 🟢
- [ ] O pacote `internal/crypto` expõe `KeyStoreService`, `KeyStoreConfig`, `KeyStoreV3` e `KeyStoreError`. 🟡
- [ ] O pacote `internal/crypto/kdf` expõe `UniversalKDFService`, `KDFCompatibilityAnalyzer`, `SecurityLevel` e `CompatibilityReport`. 🟡
- [ ] O fluxo de exibição de resultado já chama persistência somente quando `KeyStore.Enabled` está ativo. 🟢

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.

- [ ] T-SK-01, Integrar persistência ao resultado single.
  - Origem no legado: `internal/cli/commands.go:1406-1415`
  - Critério de pronto: `displayWalletResult()` verifica `app.config.KeyStore.Enabled`, chama `generateAndSaveKeystore(result.Wallet)` e imprime warning sem abortar o display quando falha.
  - Confiança: 🟢

- [ ] T-SK-02, Integrar persistência ao resultado múltiplo.
  - Origem no legado: `internal/cli/commands.go:1453-1477`
  - Critério de pronto: `displayMultipleWalletResults()` tenta salvar keystore por resultado, acumula erros e imprime resumo de sucesso/erro.
  - Confiança: 🟢

- [ ] T-SK-03, Integrar persistência silenciosa ao fluxo single TUI.
  - Origem no legado: `internal/cli/commands.go:337-344`
  - Critério de pronto: após gerar wallet em TUI single, se keystore está habilitado, chamar `generateAndSaveKeystoreWithVerbose(wallet, false)` e não interromper o resultado TUI em caso de falha.
  - Confiança: 🟢

- [ ] T-SK-04, Integrar persistência silenciosa ao fluxo múltiplo TUI.
  - Origem no legado: `internal/cli/commands.go:632-638`
  - Critério de pronto: cada wallet bem-sucedida no fluxo TUI múltiplo tenta persistência com `verbose=false` antes de enviar o resultado para TUI.
  - Confiança: 🟢

- [ ] T-SK-05, Implementar wrapper `generateAndSaveKeystore`.
  - Origem no legado: `internal/cli/commands.go:1943-1946`
  - Critério de pronto: o wrapper chama `generateAndSaveKeystoreWithVerbose(w, app.config.CLI.VerboseOutput)`.
  - Confiança: 🟢

- [ ] T-SK-06, Implementar dispatch por rede em `generateAndSaveKeystoreWithVerbose`.
  - Origem no legado: `internal/cli/commands.go:1948-1953`; `internal/cli/commands.go:1979-1980`
  - Critério de pronto: `strings.ToLower(w.Network) == "bitcoin"` seleciona fluxo mnemonic-only; demais redes seguem fluxo de keystore.
  - Confiança: 🟢

- [ ] T-SK-07, Implementar validação Bitcoin com mnemonic obrigatório.
  - Origem no legado: `internal/cli/commands.go:1952-1956`
  - Critério de pronto: carteira Bitcoin sem mnemonic retorna erro `Bitcoin wallet requires mnemonic for backup`.
  - Confiança: 🟢

- [ ] T-SK-08, Implementar salvamento Bitcoin somente com mnemonic.
  - Origem no legado: `internal/cli/commands.go:1958-1977`
  - Critério de pronto: criar `crypto.KeyStoreService` com `Enabled` e `OutputDirectory`, aplicar verbose e chamar `SaveMnemonicFile(w.Address, w.Mnemonic, w.Network)`.
  - Confiança: 🟢

- [ ] T-SK-09, Implementar tratamento amigável de erro no salvamento de mnemonic Bitcoin.
  - Origem no legado: `internal/cli/commands.go:1967-1975`
  - Critério de pronto: se o erro for `*crypto.KeyStoreError` e tiver `UserMessage`, retornar mensagem baseada em `UserMessage`; caso contrário, retornar erro contextual com address.
  - Confiança: 🟢

- [ ] T-SK-10, Criar serviço KDF universal e analyzer para Ethereum/Solana.
  - Origem no legado: `internal/cli/commands.go:1979-1985`
  - Critério de pronto: fluxo não-Bitcoin cria `kdf.NewUniversalKDFService()` e `kdf.NewKDFCompatibilityAnalyzer(kdfService)`.
  - Confiança: 🟢

- [ ] T-SK-11, Resolver parâmetros KDF default quando ausentes.
  - Origem no legado: `internal/cli/commands.go:1986-1996`
  - Critério de pronto: quando `KDFParams` está vazio, usar `parseSecurityLevel()` e `analyzer.GetOptimizedParams(KDFAlgorithm, securityLevel, 512)`.
  - Confiança: 🟢

- [ ] T-SK-12, Implementar montagem de `crypto.KeyStoreConfig` para Ethereum/Solana.
  - Origem no legado: `internal/cli/commands.go:1998-2007`
  - Critério de pronto: config inclui `Enabled`, `OutputDirectory`, `KDF`, `KDFParams`, cipher `aes-128-ctr`, `MaxRetries=3` e `RetryDelay=100`.
  - Confiança: 🟢

- [ ] T-SK-13, Implementar geração de keystore.
  - Origem no legado: `internal/cli/commands.go:2009-2017`
  - Critério de pronto: criar `KeyStoreService`, aplicar `SetVerboseMode(verbose)` e chamar `GenerateKeyStore(w.PrivateKey, w.Address, w.Network)`.
  - Confiança: 🟢

- [ ] T-SK-14, Implementar extração de parâmetros KDF completos.
  - Origem no legado: `internal/cli/commands.go:1233-1256`; `internal/cli/commands.go:2019-2027`
  - Critério de pronto: extrair parâmetros completos de scrypt/PBKDF2 do keystore gerado para análise de compatibilidade.
  - Confiança: 🟢

- [ ] T-SK-15, Implementar análise e relatório KDF opcional.
  - Origem no legado: `internal/cli/commands.go:2019-2037`; `internal/cli/commands.go:1274-1334`
  - Critério de pronto: se `ShowAnalysis` ou `verbose` estiver ativo, chamar `AnalyzeKeystore` e exibir relatório; erro de análise em verbose vira warning e não aborta salvamento.
  - Confiança: 🟢

- [ ] T-SK-16, Implementar salvamento de arquivos de keystore/password.
  - Origem no legado: `internal/cli/commands.go:2039-2049`
  - Critério de pronto: chamar `SaveKeyStoreFilesToDisk(w.Address, keystore, password, w.Network, w.PrivateKey)` e retornar erro contextual em falha.
  - Confiança: 🟢

- [ ] T-SK-17, Implementar salvamento de mnemonic opcional pós-keystore.
  - Origem no legado: `internal/cli/commands.go:2051-2060`
  - Critério de pronto: se `w.Mnemonic != ""`, chamar `SaveMnemonicFile` e tratar `KeyStoreError.UserMessage` quando existir.
  - Confiança: 🟢

- [ ] T-SK-18, Implementar conversão de security level textual.
  - Origem no legado: `internal/cli/commands.go:1258-1271`
  - Critério de pronto: mapear `low`, `medium`, `high`, `very-high`/`veryhigh`; valores desconhecidos caem para `medium` como no legado.
  - Confiança: 🟢

- [ ] T-SK-19, Preservar mensagens de sucesso e resumo nos displays.
  - Origem no legado: `internal/cli/commands.go:1410-1414`; `internal/cli/commands.go:1469-1477`
  - Critério de pronto: display single informa diretório de keystore/mnemonic; display múltiplo informa quantidade salva e quantidade de erros.
  - Confiança: 🟢

## Tarefas de Teste

- [ ] TT-SK-01, Testar que persistência não é chamada quando `KeyStore.Enabled=false`.
  - Critério de pronto: display single e múltiplo exibem resultados sem chamar helpers de keystore. 🟢

- [ ] TT-SK-02, Testar persistência single com sucesso.
  - Critério de pronto: `displayWalletResult()` chama helper, imprime diretório e retorna `nil`. 🟢

- [ ] TT-SK-03, Testar persistência single com falha.
  - Critério de pronto: falha de helper imprime warning e `displayWalletResult()` retorna `nil`. 🟢

- [ ] TT-SK-04, Testar persistência múltipla com sucesso parcial.
  - Critério de pronto: erros são acumulados, sucessos contam no resumo e a função retorna `nil`. 🟢

- [ ] TT-SK-05, Testar Bitcoin sem mnemonic.
  - Critério de pronto: `generateAndSaveKeystoreWithVerbose()` retorna erro `Bitcoin wallet requires mnemonic for backup`. 🟢

- [ ] TT-SK-06, Testar Bitcoin com mnemonic.
  - Critério de pronto: chama `SaveMnemonicFile` e não chama `GenerateKeyStore`. 🟢

- [ ] TT-SK-07, Testar tratamento de `KeyStoreError` com `UserMessage`.
  - Critério de pronto: erro retornado usa a mensagem amigável em salvamento de mnemonic, keystore e mnemonic pós-keystore. 🟢

- [ ] TT-SK-08, Testar Ethereum/Solana com KDF params explícitos.
  - Critério de pronto: `GetOptimizedParams` não é chamado e os params configurados entram no `KeyStoreConfig`. 🟢

- [ ] TT-SK-09, Testar Ethereum/Solana sem KDF params.
  - Critério de pronto: `GetOptimizedParams` é chamado com algoritmo, security level e limite de 512MB. 🟢

- [ ] TT-SK-10, Testar análise KDF habilitada.
  - Critério de pronto: `AnalyzeKeystore` e `displayCompatibilityReport` são chamados quando `ShowAnalysis=true`. 🟢

- [ ] TT-SK-11, Testar verbose habilitando relatório KDF.
  - Critério de pronto: `verbose=true` exibe relatório mesmo quando `ShowAnalysis=false`. 🟢

- [ ] TT-SK-12, Testar falha de análise KDF.
  - Critério de pronto: erro de `AnalyzeKeystore` em verbose gera warning e ainda tenta salvar keystore. 🟢

- [ ] TT-SK-13, Testar salvamento de mnemonic opcional em Ethereum/Solana.
  - Critério de pronto: wallet com mnemonic chama `SaveMnemonicFile` após `SaveKeyStoreFilesToDisk`. 🟢

- [ ] TT-SK-14, Testar `parseSecurityLevel` para valores conhecidos e desconhecidos.
  - Critério de pronto: valores conhecidos mapeiam corretamente; valor desconhecido retorna `SecurityLevelMedium`. 🟢

## Tarefas de Migração de Dados

Não aplicável. Este caso de uso cria novos arquivos locais de backup e não migra base de dados existente. 🟢

## Ordem Sugerida

1. Integrar pontos de acionamento em display e TUI (`T-SK-01` a `T-SK-04`). 🟢
2. Implementar wrapper e dispatch por rede (`T-SK-05`, `T-SK-06`). 🟢
3. Implementar fluxo Bitcoin mnemonic-only (`T-SK-07` a `T-SK-09`). 🟢
4. Implementar fluxo Ethereum/Solana com KDF e keystore (`T-SK-10` a `T-SK-17`). 🟢
5. Implementar helpers de security level e mensagens de display (`T-SK-18`, `T-SK-19`). 🟢
6. Executar testes de persistência, erros, KDF, verbose e resumo (`TT-SK-01` a `TT-SK-14`). 🟢

## Lacunas Pendentes (🔴)

- 🟢 Falha de keystore é warning nos displays; se backup for obrigatório, a política de erro deve ser alterada conscientemente.
- 🟡 Solana compartilha fluxo Ethereum/Solana, mas specs anteriores indicam persistência Solana simplificada/placeholder.
- 🟡 Formato exato, nomes e permissões dos arquivos dependem de `internal/crypto.KeyStoreService`, fora desta unit.
- 🟢 `parseSecurityLevel()` usa fallback silencioso para `medium`; validar se esse comportamento deve permanecer.
- 🟢 `SaveKeyStoreFilesToDisk` recebe private key; revisar requisitos de segurança de filesystem e retenção de segredos.
