# Módulo internal/cli, Contratos

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

Este arquivo documenta os contratos operacionais expostos pela CLI `bloco-vgen`: comandos, flags, entradas, saídas, exit behavior, artefatos gerados e contratos internos que a camada CLI espera dos módulos dependentes. O sistema não expõe endpoints HTTP/RPC. 🟢

## Contrato de Processo

| Item | Contrato | Evidência | Confiança |
|---|---|---|---:|
| Binário | O comando raiz é `bloco-vgen`. | `internal/cli/commands.go:57-66` | 🟢 |
| Execução padrão | Sem subcomando, o root command executa geração de carteira via `RunE: app.generateWallet`. | `internal/cli/commands.go:57-66` | 🟢 |
| Versão | O comando raiz expõe versão no formato `<version> (commit: <gitCommit>, built: <buildTime>)`. | `internal/cli/commands.go:64` | 🟢 |
| Contexto | Execução aceita `context.Context` por Cobra/Fang. | `internal/cli/commands.go:50-52`; `cmd/bloco-vgen/main.go:42-47` | 🟢 |
| Erros | Handlers retornam `error`; entrypoint trata e encerra com código `1`. | `cmd/bloco-vgen/main.go:47-50` | 🟢 |

## Contrato do Comando Raiz: geração de carteiras

### Comando

```text
bloco-vgen [flags]
```

### Flags de geração

| Flag | Tipo | Default | Descrição contratual | Evidência | Confiança |
|---|---|---:|---|---|---:|
| `--prefix`, `-p` | string | `""` | Prefixo de endereço a casar. | `internal/cli/commands.go:81-83` | 🟢 |
| `--suffix`, `-s` | string | `""` | Sufixo de endereço a casar. | `internal/cli/commands.go:82-83` | 🟢 |
| `--checksum`, `-c` | bool | `false` | Habilita validação EIP-55/checksum. | `internal/cli/commands.go:84` | 🟢 |
| `--case-sensitive` | bool | `false` | Declara matching case-sensitive dependente de checksum. Uso efetivo não confirmado em `getGenerationCriteria`. | `internal/cli/commands.go:85`; `internal/cli/commands.go:1352-1368` | 🟡 |
| `--count`, `-n` | int | `1` | Número de carteiras a gerar. | `internal/cli/commands.go:86`; `internal/cli/commands.go:169-174` | 🟢 |
| `--with-mnemonic` | bool | `false` | Solicita geração com mnemonic BIP-39. | `internal/cli/commands.go:87`; `internal/cli/commands.go:1357-1365` | 🟢 |
| `--network` | string | `ethereum` | Rede alvo: `ethereum`, `bitcoin`, `solana`. | `internal/cli/commands.go:88`; `internal/cli/commands.go:1358-1365` | 🟢 |

### Flags de performance e UI

| Flag | Tipo | Default | Descrição contratual | Evidência | Confiança |
|---|---|---:|---|---|---:|
| `--threads`, `-t` | int | `0` | `0` autodetecta CPUs; valor positivo define threads. | `internal/cli/commands.go:91`; `internal/cli/commands.go:971-978` | 🟢 |
| `--progress` | bool | `false` | Solicita exibição de progresso; em modo texto, progresso contínuo está desabilitado por deadlock. | `internal/cli/commands.go:92`; `internal/cli/commands.go:396-398` | 🟢 |
| `--tui` | bool | `true` | Permite TUI quando terminal suporta e `--progress` está ativo. | `internal/cli/commands.go:93`; `internal/cli/commands.go:184-199` | 🟢 |

### Flags de saída

| Flag | Tipo | Default | Descrição contratual | Evidência | Confiança |
|---|---|---:|---|---|---:|
| `--verbose`, `-v` | bool | `false` | Ativa saída verbosa e verbose em keystore. | `internal/cli/commands.go:96`; `internal/cli/commands.go:981-984` | 🟢 |
| `--quiet`, `-q` | bool | `false` | Suprime saída não essencial e oculta segredos em múltiplos resultados. | `internal/cli/commands.go:97`; `internal/cli/commands.go:986-988`; `internal/cli/commands.go:1438-1444` | 🟢 |
| `--output` | string | `""` | Contrato declarado para output file, mas uso efetivo não confirmado no fluxo lido. | `internal/cli/commands.go:98` | 🟡 |
| `--format` | string | `text` | Contrato declarado para formato `text/json/csv`, mas uso efetivo não confirmado no fluxo lido. | `internal/cli/commands.go:99` | 🟡 |

### Flags de keystore e KDF

| Flag | Tipo | Default | Descrição contratual | Evidência | Confiança |
|---|---|---:|---|---|---:|
| `--keystore-dir` | string | `./keystores` | Diretório para arquivos de keystore/password/mnemonic. | `internal/cli/commands.go:102`; `internal/cli/commands.go:1000-1004` | 🟢 |
| `--no-keystore` | bool | `false` | Desabilita geração de keystore. | `internal/cli/commands.go:103`; `internal/cli/commands.go:995-998` | 🟢 |
| `--keystore-kdf` | string | `scrypt` | Algoritmo KDF: `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512`. | `internal/cli/commands.go:104`; `internal/cli/commands.go:1007-1012` | 🟢 |
| `--kdf-params` | string JSON | `""` | Parâmetros KDF customizados em JSON. | `internal/cli/commands.go:105`; `internal/cli/commands.go:1014-1021`; `internal/cli/commands.go:1097-1231` | 🟢 |
| `--kdf-analysis` | bool | `false` | Exibe análise de compatibilidade/segurança KDF. | `internal/cli/commands.go:106`; `internal/cli/commands.go:1023-1026` | 🟢 |
| `--security-level` | string | `medium` | Nível mínimo de segurança KDF: `low`, `medium`, `high`, `very-high`. | `internal/cli/commands.go:107`; `internal/cli/commands.go:1258-1271` | 🟢 |

### Flags de logging seguro

| Flag | Tipo | Default | Descrição contratual | Evidência | Confiança |
|---|---|---:|---|---|---:|
| `--log-level` | string | `info` | Nível de logging seguro. | `internal/cli/commands.go:110`; `internal/cli/commands.go:1052-1057` | 🟢 |
| `--no-logging` | bool | `false` | Desabilita logging e ignora demais flags de logging. | `internal/cli/commands.go:111`; `internal/cli/commands.go:1044-1050` | 🟢 |
| `--log-file` | string | `""` | Caminho do arquivo de log. | `internal/cli/commands.go:112`; `internal/cli/commands.go:1066-1071` | 🟢 |
| `--log-format` | string | `text` | Formato do log: `text`, `json`, `structured`. | `internal/cli/commands.go:113`; `internal/cli/commands.go:1059-1064` | 🟢 |
| `--log-max-size` | int64 | `10485760` | Tamanho máximo antes de rotação. | `internal/cli/commands.go:114`; `internal/cli/commands.go:1073-1078` | 🟢 |
| `--log-max-files` | int | `5` | Quantidade máxima de arquivos rotacionados. | `internal/cli/commands.go:115`; `internal/cli/commands.go:1080-1085` | 🟢 |
| `--log-buffer-size` | int | `1000` | Tamanho do buffer assíncrono. | `internal/cli/commands.go:116`; `internal/cli/commands.go:1087-1092` | 🟢 |

## Contrato do Subcomando `stats`

```text
bloco-vgen stats [--prefix <hex>] [--suffix <hex>] [--checksum] [--tui]
```

| Item | Contrato | Evidência | Confiança |
|---|---|---|---:|
| Handler | `RunE: app.showStats`. | `internal/cli/commands.go:736-743` | 🟢 |
| Flags próprias | `prefix`, `suffix`, `checksum`. | `internal/cli/commands.go:745-749` | 🟢 |
| Entrada | Critérios extraídos por `getGenerationCriteria`. | `internal/cli/commands.go:753-759` | 🟢 |
| Saída texto | Pattern, tamanho, checksum, dificuldade, 50% probability e estimativas por velocidade. | `internal/cli/commands.go:810-833` | 🟢 |
| Saída TUI | `StatsModel` Bubble Tea quando `--tui` e terminal suportam. | `internal/cli/commands.go:765-805` | 🟢 |
| Fallback | Falha na TUI cai para texto. | `internal/cli/commands.go:800-805` | 🟢 |

## Contrato do Subcomando `benchmark`

```text
bloco-vgen benchmark [--attempts <n>] [--duration <duration>] [--detailed] [--tui]
```

| Item | Contrato | Evidência | Confiança |
|---|---|---|---:|
| Handler | `RunE: app.runBenchmark`. | `internal/cli/commands.go:836-843` | 🟢 |
| `--attempts` | Número de tentativas, default `10000`. | `internal/cli/commands.go:845-847` | 🟢 |
| `--duration` | Duração do benchmark, default `30s`. | `internal/cli/commands.go:847` | 🟢 |
| `--detailed` | Exibe amostras e estatísticas detalhadas. | `internal/cli/commands.go:848`; `internal/cli/commands.go:1840-1881` | 🟢 |
| Saída | `BenchmarkResult` com total attempts, duration, average/min/max speed e métricas de thread. | `internal/cli/commands.go:1666-1680`; `internal/cli/commands.go:1792-1806` | 🟢 |
| Fallback | Falha na TUI cai para texto. | `internal/cli/commands.go:912-917` | 🟢 |
| Lacuna | Work items são criados, mas execução real no pool está marcada como TODO. | `internal/cli/commands.go:1629-1630`; `internal/cli/commands.go:1753-1754` | 🟢 |

## Contrato do Subcomando `version`

```text
bloco-vgen version
```

| Saída | Evidência | Confiança |
|---|---|---:|
| `Bloco-ETH <version>` | `internal/cli/commands.go:954-963` | 🟢 |
| `Git Commit: <gitCommit>` | `internal/cli/commands.go:960-962` | 🟢 |
| `Build Time: <buildTime>` | `internal/cli/commands.go:960-962` | 🟢 |

## Contrato de Saída de Geração Single

| Campo | Condição | Evidência | Confiança |
|---|---|---|---:|
| `Wallet generated successfully!` | Sempre em `displayWalletResult`. | `internal/cli/commands.go:1395-1398` | 🟢 |
| `Address` | Sempre quando `result.Wallet` existe. | `internal/cli/commands.go:1397-1398` | 🟢 |
| `Private Key` | Sempre no fluxo single texto. | `internal/cli/commands.go:1398-1399` | 🟢 |
| `Mnemonic` | Apenas quando `result.Wallet.Mnemonic != ""`. | `internal/cli/commands.go:1400-1402` | 🟢 |
| `Attempts` | Sempre. | `internal/cli/commands.go:1403` | 🟢 |
| `Duration` | Sempre. | `internal/cli/commands.go:1404` | 🟢 |
| Warning de keystore | Quando persistência falha. | `internal/cli/commands.go:1406-1415` | 🟢 |

## Contrato de Saída de Geração Múltipla

| Campo | Condição | Evidência | Confiança |
|---|---|---|---:|
| Mensagem de lista vazia | `len(results) == 0`. | `internal/cli/commands.go:1421-1425` | 🟢 |
| Total de carteiras | Quando há resultados. | `internal/cli/commands.go:1427` | 🟢 |
| Total attempts | Quando há resultados. | `internal/cli/commands.go:1428` | 🟢 |
| Total duration | Quando há resultados. | `internal/cli/commands.go:1429` | 🟢 |
| Average speed | Quando há resultados. | `internal/cli/commands.go:1430` | 🟢 |
| Address por carteira | Sempre por resultado. | `internal/cli/commands.go:1435-1437` | 🟢 |
| Private key/mnemonic | Apenas se `!QuietMode`. | `internal/cli/commands.go:1438-1444` | 🟢 |
| Keystore status | Quando `KeyStore.Enabled`. | `internal/cli/commands.go:1453-1464` | 🟢 |
| Statistics Summary | Quando `len(results) > 1`. | `internal/cli/commands.go:1480-1502` | 🟢 |

## Contrato de Artefatos de Persistência

| Rede | Artefatos esperados | Regra | Evidência | Confiança |
|---|---|---|---|---:|
| Bitcoin | `.mnemonic` | Requer mnemonic e não gera KeyStore V3. | `internal/cli/commands.go:1951-1977` | 🟢 |
| Ethereum | keystore/password e mnemonic opcional | Gera KeyStore com KDF universal e salva arquivos. | `internal/cli/commands.go:1979-2063` | 🟢 |
| Solana | keystore/formato específico e mnemonic opcional | Segue o mesmo fluxo Ethereum/Solana, com ressalvas documentadas em arquitetura. | `internal/cli/commands.go:1979-2063`; `_reversa_sdd/domain.md:GAP-004` | 🟡 |

## Contrato de Erros

| Situação | Erro/Tratamento | Evidência | Confiança |
|---|---|---|---:|
| Falha ao parsear flags | `errors.WrapError(... ErrorTypeConfiguration ...)`. | `internal/cli/commands.go:130-134` | 🟢 |
| Critérios inválidos | `errors.WrapError(... ErrorTypeValidation ...)`. | `internal/cli/commands.go:136-141` | 🟢 |
| Falha ao iniciar worker pool | `errors.WrapError(... ErrorTypeWorker ...)`. | `internal/cli/commands.go:157-161` | 🟢 |
| Falha de geração single texto | `errors.WrapError(... ErrorTypeGeneration ...)`. | `internal/cli/commands.go:400-408` | 🟢 |
| Falha de benchmark texto | `errors.WrapError(... ErrorTypeGeneration ...)`. | `internal/cli/commands.go:943-948` | 🟢 |
| KDF params inválidos | `fmt.Errorf("invalid KDF parameters: %w", err)`. | `internal/cli/commands.go:1014-1021` | 🟢 |
| Bitcoin sem mnemonic | `fmt.Errorf("Bitcoin wallet requires mnemonic for backup")`. | `internal/cli/commands.go:1952-1956` | 🟢 |
| Keystore save falhou | Warning no display ou erro contextual no helper. | `internal/cli/commands.go:1406-1415`; `internal/cli/commands.go:2039-2049` | 🟢 |

## Lacunas Contratuais

| ID | Lacuna | Impacto | Confiança |
|---|---|---|---:|
| CLI-CONTRACT-GAP-001 | `--case-sensitive` é declarada, mas uso efetivo não foi confirmado no critério de geração. | Usuário pode esperar comportamento que depende apenas de `--checksum`. | 🟡 |
| CLI-CONTRACT-GAP-002 | `--output` e `--format` são declaradas, mas uso efetivo não foi confirmado no fluxo lido. | Contrato de saída para arquivo/JSON/CSV pode estar incompleto. | 🟡 |
| CLI-CONTRACT-GAP-003 | README menciona benchmark `--pattern`, mas o contrato Cobra lido não declara essa flag. | Divergência documentação/CLI. | 🟡 |
| CLI-CONTRACT-GAP-004 | Benchmark tem TODO para execução real de work items com pool. | Métricas podem não refletir geração real esperada. | 🟢 |
| CLI-CONTRACT-GAP-005 | Fluxo single texto imprime private key e mnemonic em stdout. | Risco operacional se stdout for capturado. | 🟢 |
