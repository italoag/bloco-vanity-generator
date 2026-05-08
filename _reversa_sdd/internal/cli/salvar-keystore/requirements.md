# Caso de Uso: Salvar Keystore

> Spec gerada pelo Reversa Writer.  
> Unit pai: `internal/cli`  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

O caso de uso `salvar-keystore` representa a persistência local de artefatos de backup após a geração de uma carteira. A camada CLI decide se a persistência deve ocorrer com base em `app.config.KeyStore.Enabled`, delega o salvamento a `generateAndSaveKeystore()` / `generateAndSaveKeystoreWithVerbose()`, aplica regras específicas por rede e reporta falhas como warning nos fluxos de exibição de resultado. 🟢

## Responsabilidades

- Verificar se a geração de keystore está habilitada antes de persistir artefatos. 🟢
- Acionar persistência depois que uma carteira foi gerada com sucesso. 🟢
- Tratar Bitcoin como fluxo especial de backup via mnemonic, sem KeyStore V3. 🟢
- Exigir mnemonic para backup de carteira Bitcoin. 🟢
- Gerar KeyStore para Ethereum e Solana usando KDF configurado ou parâmetros otimizados. 🟢
- Aplicar KDF universal e análise de compatibilidade quando configurado. 🟢
- Salvar arquivos de keystore/password em disco para redes compatíveis. 🟢
- Salvar mnemonic adicional quando a carteira possuir mnemonic. 🟢
- Transformar falhas de persistência em mensagens úteis para o operador. 🟢
- Não sobrescrever o resultado principal da geração com falhas não fatais de keystore nos fluxos de display. 🟢

## Regras de Negócio

- A persistência só deve ocorrer quando `app.config.KeyStore.Enabled` estiver ativa. 🟢
- `displayWalletResult()` deve tentar salvar keystore após imprimir dados da carteira single. 🟢
- `displayMultipleWalletResults()` deve tentar salvar keystore para cada resultado do lote quando habilitado. 🟢
- Falha de keystore durante exibição de resultado deve ser comunicada como warning/status e não impedir a exibição da carteira. 🟢
- `generateAndSaveKeystore()` deve delegar para `generateAndSaveKeystoreWithVerbose()` usando `app.config.CLI.VerboseOutput`. 🟢
- Para Bitcoin, a CLI deve salvar apenas mnemonic, sem KeyStore V3. 🟢
- Para Bitcoin, carteira sem mnemonic deve gerar erro `Bitcoin wallet requires mnemonic for backup`. 🟢
- Para Ethereum e Solana, a CLI deve criar `UniversalKDFService` e `KDFCompatibilityAnalyzer`. 🟢
- Por decisão humana, Solana deve usar persistência criptografada/segura, não arquivo `.key` bruto. 🟢
- Quando `KDFParams` está vazio, parâmetros default devem ser obtidos por nível de segurança e limite de memória de 512MB. 🟢
- O `KeyStoreConfig` deve usar KDF configurado, KDF params, cipher `aes-128-ctr`, `MaxRetries=3` e `RetryDelay=100`. 🟢
- Quando `ShowAnalysis` ou `verbose` estão ativos, a CLI deve extrair parâmetros completos do keystore e exibir relatório de compatibilidade KDF. 🟢
- Erros tipados como `crypto.KeyStoreError` devem preferir `UserMessage` quando disponível. 🟢
- Se a carteira possui mnemonic, a CLI deve salvar mnemonic além do keystore para Ethereum/Solana. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-SK-01 | Acionar persistência no resultado single quando habilitada. | Must | `displayWalletResult()` chama `generateAndSaveKeystore(result.Wallet)` se `KeyStore.Enabled` é verdadeiro. 🟢 |
| RF-SK-02 | Acionar persistência por carteira em resultados múltiplos quando habilitada. | Must | `displayMultipleWalletResults()` chama `generateAndSaveKeystore(result.Wallet)` para cada resultado. 🟢 |
| RF-SK-03 | Reportar falha de keystore como warning no fluxo single. | Should | Falha imprime `Warning: Failed to generate keystore` e a função retorna `nil`. 🟢 |
| RF-SK-04 | Reportar sucesso de keystore no fluxo single. | Should | Sucesso imprime `Keystore saved to: <dir>` e mensagem de mnemonic quando aplicável. 🟢 |
| RF-SK-05 | Reportar status de keystore por carteira no fluxo múltiplo. | Should | Cada wallet imprime `Keystore: Saved` ou `Keystore: Failed to generate`. 🟢 |
| RF-SK-06 | Exibir resumo de keystores em lote. | Should | O fluxo múltiplo imprime `Keystores saved: x/y` e quantidade de erros quando houver. 🟢 |
| RF-SK-07 | Encapsular chamada simples de keystore. | Must | `generateAndSaveKeystore(w)` delega para `generateAndSaveKeystoreWithVerbose(w, app.config.CLI.VerboseOutput)`. 🟢 |
| RF-SK-08 | Salvar Bitcoin somente como mnemonic. | Must | Para `network == bitcoin`, o fluxo cria `KeyStoreService` apenas para `SaveMnemonicFile`. 🟢 |
| RF-SK-09 | Rejeitar backup Bitcoin sem mnemonic. | Must | Carteira Bitcoin sem mnemonic retorna erro específico antes de criar arquivo. 🟢 |
| RF-SK-10 | Gerar keystore Ethereum/Solana com KDF universal. | Must | O fluxo cria `UniversalKDFService`, analyzer, params, `KeyStoreService` e chama `GenerateKeyStore`. 🟢 |
| RF-SK-11 | Obter parâmetros KDF default quando ausentes. | Must | Se `len(KDFParams) == 0`, usa `GetOptimizedParams(KDFAlgorithm, securityLevel, 512)`. 🟢 |
| RF-SK-12 | Exibir análise KDF quando habilitada. | Should | Com `ShowAnalysis` ou `verbose`, o fluxo chama `AnalyzeKeystore` e `displayCompatibilityReport`. 🟢 |
| RF-SK-13 | Salvar arquivos de keystore/password em disco. | Must | O fluxo chama `SaveKeyStoreFilesToDisk(address, keystore, password, network, privateKey)`. 🟢 |
| RF-SK-16 | Persistir Solana em formato criptografado/seguro. | Must | Reimplementação não deve gerar `.key` bruto com private key em claro. 🟢 |
| RF-SK-14 | Salvar mnemonic adicional quando disponível. | Must | Se `w.Mnemonic != ""`, o fluxo chama `SaveMnemonicFile`. 🟢 |
| RF-SK-15 | Melhorar mensagem de erro para `KeyStoreError`. | Should | Quando erro é `*crypto.KeyStoreError`, `UserMessage` é preferido se existir. 🟢 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Segurança | Bitcoin não deve gerar KeyStore V3; backup depende de mnemonic. | `internal/cli/commands.go:1948-1977` | 🟢 |
| Segurança | KDF deve usar parâmetros explícitos ou defaults otimizados por security level. | `internal/cli/commands.go:1986-1996` | 🟢 |
| Segurança | Cipher configurado para keystore Ethereum/Solana é `aes-128-ctr`. | `internal/cli/commands.go:1998-2007` | 🟢 |
| Compatibilidade | Análise KDF deve ser exibida quando o usuário solicitar ou modo verbose estiver ativo. | `internal/cli/commands.go:2019-2037` | 🟢 |
| Resiliência | Falhas de persistência durante display não devem ocultar o resultado da carteira. | `internal/cli/commands.go:1406-1415`; `internal/cli/commands.go:1453-1464` | 🟢 |
| Operabilidade | Mensagens de erro de keystore devem preferir mensagem amigável quando disponível. | `internal/cli/commands.go:1967-1975`; `internal/cli/commands.go:2039-2059` | 🟢 |
| UX | Modo verbose controla saída detalhada e relatório de compatibilidade. | `internal/cli/commands.go:1943-1951`; `internal/cli/commands.go:2019-2037` | 🟢 |

> Inferido a partir do código. Validar política de backup e exposição de credenciais com equipe de segurança.

## Critérios de Aceitação

```gherkin
Dado que uma carteira foi gerada com sucesso
E `KeyStore.Enabled` está ativo
Quando o resultado single é exibido
Então a CLI deve tentar salvar keystore ou mnemonic conforme a rede

Dado que o salvamento de keystore falha no fluxo single
Quando o resultado é exibido
Então a CLI deve imprimir warning e ainda retornar sucesso do display

Dado que múltiplas carteiras foram geradas
E `KeyStore.Enabled` está ativo
Quando os resultados são exibidos
Então a CLI deve tentar persistir artefatos para cada carteira e exibir resumo de sucesso/erro

Dado que a rede da carteira é Bitcoin
E a carteira possui mnemonic
Quando `generateAndSaveKeystoreWithVerbose` é executado
Então a CLI deve salvar apenas o mnemonic usando `SaveMnemonicFile`

Dado que a rede da carteira é Bitcoin
E a carteira não possui mnemonic
Quando a persistência é executada
Então a CLI deve retornar erro informando que Bitcoin requer mnemonic para backup

Dado que a rede é Ethereum ou Solana
E não há KDF params configurados
Quando a persistência é executada
Então a CLI deve obter parâmetros default otimizados a partir do nível de segurança

Dado que `kdf-analysis` ou `verbose` está ativo
Quando o keystore é gerado
Então a CLI deve analisar compatibilidade KDF e exibir relatório

Dado que a carteira Ethereum/Solana possui mnemonic
Quando o keystore é salvo com sucesso
Então a CLI também deve salvar o mnemonic
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Persistência quando `KeyStore.Enabled` | Must | Comportamento integrado ao resultado e backup da carteira. 🟢 |
| Bitcoin somente mnemonic | Must | Regra explícita por rede no código. 🟢 |
| Ethereum/Solana com KDF universal | Must | Caminho principal de keystore compatível. 🟢 |
| Tratamento de erro amigável | Should | Melhora UX sem alterar a geração da carteira. 🟢 |
| Análise KDF | Should | Importante para auditoria/compatibilidade, mas controlada por flag/verbose. 🟢 |
| Resumo de lote | Should | Ajuda operação em geração múltipla. 🟢 |
| Falha de keystore como warning | Should | Preserva comportamento legado, mas pode exigir decisão em ambientes regulados. 🟢 |

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `internal/cli/commands.go` | `displayWalletResult` | 🟢 |
| `internal/cli/commands.go` | `displayMultipleWalletResults` | 🟢 |
| `internal/cli/commands.go` | `generateAndSaveKeystore` | 🟢 |
| `internal/cli/commands.go` | `generateAndSaveKeystoreWithVerbose` | 🟢 |
| `internal/cli/commands.go` | `parseSecurityLevel` | 🟢 |
| `internal/cli/commands.go` | `extractKDFParamsFromKeystore` | 🟢 |
| `internal/cli/commands.go` | `displayCompatibilityReport` | 🟢 |
| `internal/crypto/*` | `KeyStoreService`, `KeyStoreConfig`, `KeyStoreError` | 🟡 |
| `internal/crypto/kdf/*` | `UniversalKDFService`, `KDFCompatibilityAnalyzer` | 🟡 |
| `pkg/wallet/*` | `Wallet` | 🟡 |

## Lacunas e Ambiguidades

- 🟢 Falha de keystore é warning no display, mas o helper retorna erro; validar se automações devem falhar quando backup não é salvo.
- 🟡 O fluxo agrupa Ethereum e Solana; a análise anterior marcou persistência Solana como simplificação/placeholder.
- 🟢 Bitcoin sem mnemonic falha, mas a CLI só terá backup Bitcoin se o fluxo de geração produzir mnemonic.
- 🟡 O local e formato exato dos arquivos salvos dependem de `internal/crypto.KeyStoreService`, não detalhado nesta unit.
- 🟢 A private key é passada para `SaveKeyStoreFilesToDisk`; validar política de retenção e exposição em disco.
