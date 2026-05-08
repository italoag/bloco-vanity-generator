# Code Analysis — Archaeologist

> Projeto: `bloco-wallet-generator`  
> Linguagem principal: Go  
> Nível de documentação: completo  
> Confiança geral: 🟢 CONFIRMADO, extraído dos arquivos Go do legado.

## Visão executiva

O sistema é uma aplicação CLI para geração de carteiras vanity multi-rede. O fluxo central nasce em `cmd/bloco-eth/main.go`, cria configuração padrão, aplica variáveis de ambiente, valida limites e delega a execução ao comando raiz Cobra por meio do Fang.

A geração efetiva é orquestrada por `internal/cli`, executada por `internal/worker` e apoiada por `internal/crypto`. A aplicação suporta Ethereum, Bitcoin e Solana, com foco mais profundo em Ethereum: geração secp256k1, endereço por Keccak256, checksum EIP-55 e KeyStore V3 com AES-128-CTR, MAC Keccak e KDF configurável.

## Módulos analisados

| Módulo | Papel | Complexidade | Confiança |
|---|---|---:|---|
| `cmd/bloco-eth` | Bootstrap, config, sinais e execução CLI | baixa | 🟢 |
| `internal/cli` | Orquestração dos comandos, flags, TUI/texto, keystore e benchmark | alta | 🟢 |
| `internal/config` | Defaults, variáveis de ambiente e validação | média | 🟢 |
| `internal/crypto` | Geração cripto multi-rede, checksum, keystore, pools e senha | alta | 🟢 |
| `internal/crypto/kdf` | Serviço universal de KDF e análise de compatibilidade | alta | 🟢 |
| `internal/progress` | Progresso textual thread-safe | média | 🟢 |
| `internal/tui` | UI terminal Bubble Tea para progresso, stats e benchmark | alta | 🟢 |
| `internal/validation` | Estratégias de validação de padrão/endereço | média | 🟢 |
| `internal/worker` | Fan-out concorrente, loop de tentativa e métricas | alta | 🟢 |
| `pkg/errors` | Erros estruturados | baixa | 🟢 |
| `pkg/logging` | Logging seguro, sanitização, rotação e buffering | alta | 🟢 |
| `pkg/utils` | Formatação e fórmulas probabilísticas | baixa | 🟢 |
| `pkg/wallet` | Entidades de domínio e estatísticas | média | 🟢 |

## Fluxo principal de geração

1. `main()` cria contexto cancelável por sinal, carrega `config.DefaultConfig()` e aplica `LoadFromEnvironment()`.
2. `config.Validate()` bloqueia configurações inválidas antes da CLI iniciar.
3. `cli.NewApplication()` monta o comando raiz `bloco-eth` e subcomandos `stats`, `benchmark` e `version`.
4. O comando raiz chama `generateWallet()`.
5. `parseFlags()` atualiza a configuração com flags de threads, TUI, keystore, KDF e logging.
6. `getGenerationCriteria()` cria `wallet.GenerationCriteria` com `network`, `prefix`, `suffix`, `checksum` e `with-mnemonic`.
7. A CLI cria `PoolManager`, `ChecksumValidator`, `AddressValidator` e `worker.Pool`.
8. O pool inicia o coletor de estatísticas e dispara `threadCount` goroutines.
9. Cada worker gera chaves/endereço até `matchesCriteria()` aceitar prefixo/sufixo/checksum.
10. O primeiro worker que encontra match envia `GenerationResult` por canal bufferizado.
11. A CLI mostra resultado em TUI ou texto e, se habilitado, salva keystore/mnemonic.

## Algoritmos confirmados

### Geração de endereço Ethereum

`EthereumGenerator.OptimizedAddressGeneration()` valida chave privada de 32 bytes, calcula a chave pública secp256k1 por `ScalarBaseMult`, serializa coordenadas X/Y com padding de 32 bytes, aplica Keccak256 e usa os últimos 20 bytes como endereço hexadecimal de 40 caracteres.

### Checksum EIP-55

`ChecksumValidator.ToChecksumAddress()` converte o endereço para lowercase, calcula Keccak256 da string lowercase e usa cada nibble do hash para decidir se letras `a-f` do endereço devem ficar maiúsculas. `ValidatePatternChecksum()` aplica essa regra apenas às posições cobertas por prefixo/sufixo.

### Dificuldade e probabilidade

`utils.CalculateDifficulty()` usa:

- sem checksum: `16^len(prefix+suffix)`
- com checksum: `16^len(pattern) * 2^letterCount`

`CalculateProbability()` usa `1 - (1 - 1/difficulty)^attempts`. `CalculateProbability50()` usa `log(0.5) / log(1 - 1/difficulty)`.

### Loop concorrente de geração

`Pool.GenerateWalletWithContext()` cria uma goroutine por thread. Cada goroutine mantém contador local, envia `WorkerStats` a cada 100ms ou 1000 tentativas e verifica cancelamento via `ctx.Done()`. Para o caminho otimizado, a reconstrução da chave privada completa só ocorre depois que o endereço gerado bate com o padrão, reduzindo custo por tentativa.

### KeyStore V3

A criptografia de private key usa:

- KDF: `scrypt`, `pbkdf2`, `pbkdf2-sha256` ou `pbkdf2-sha512`
- salt aleatório de 32 bytes
- IV aleatório de 16 bytes
- AES-128-CTR com `derivedKey[:16]`
- MAC por Keccak256 de `derivedKey[16:32] + ciphertext`
- JSON versão 3 com cipher `aes-128-ctr`

### Password generation

`PasswordGenerator.GenerateSecurePassword()` garante ao menos um caractere de cada classe, preenche o restante com seleção aleatória usando `crypto/rand` e rejection sampling, e embaralha com Fisher-Yates.

### KDF universal

`UniversalKDFService` normaliza aliases de KDF, valida parâmetros com handlers concretos, deriva a chave, registra tentativa/sucesso/erro por logger e expõe defaults/recomendações. O analyzer estima segurança, memória, compatibilidade com clientes (`geth`, `besu`, `anvil`, `reth`, `firefly`) e sugere parâmetros por nível de segurança.

## Regras de negócio embutidas

- **Padrão máximo:** `GenerationCriteria.Validate()` limita prefixo+sufixo a 20 caracteres.
- **Hex obrigatório:** prefixo e sufixo devem conter apenas caracteres hexadecimais.
- **Threads:** configuração global limita `Worker.ThreadCount` a `1..128`; dentro do pool, valores `<=0` viram `1`.
- **Mnemonic:** no worker, mnemonic é suportado somente para Ethereum; para Bitcoin/Solana o worker desativa `UseMnemonic`.
- **Bitcoin:** persistência salva mnemonic, não KeyStore V3.
- **Keystore:** diretório padrão `./keystores`; arquivos sensíveis são artefatos gerados.
- **TUI:** usada apenas quando `TUI.Enabled`, `--progress` e não `quiet`, e quando terminal suporta TUI.
- **Logging:** parâmetros sensíveis são filtrados por whitelist; salt não é logado.

## Tratamento de erros

O projeto usa duas famílias principais:

- `pkg/errors.BlocoError`: erro estruturado com tipo, operação, causa, contexto, timestamp e stack opcional.
- `crypto.KeyStoreError` / `kdf.KDFError`: erros especializados para keystore/KDF, com recuperabilidade e sugestões.

A CLI envolve erros por contexto (`WrapError`) e evita que falhas de shutdown/logging sobreponham a falha principal.

## Concorrência e estado

- `worker.Pool` protege `isRunning` com `sync.RWMutex`.
- `StatsCollector` protege mapas e agregações com `sync.RWMutex`.
- `ProgressManager` usa `atomic.CompareAndSwapInt32` para impedir Start/Stop duplicados.
- TUI multi-wallet usa `sync.Mutex` para `completedWallets` e `sync.Once` para fechar canal de shutdown uma única vez.

## Lacunas e riscos técnicos

| Item | Severidade | Evidência | Confiança |
|---|---|---|---|
| Endereço placeholder em `EncryptPrivateKeyWithKDF()` | média | `address := "0000000000000000000000000000000000000000"` antes de montar KeyStore | 🟢 |
| Comentários indicam progress text desabilitado por deadlocks | média | `generateSingleWalletText()` e `generateMultipleWalletsText()` ignoram progress manager | 🟢 |
| `wallet.Wallet.IsValid()` exige endereço de 40 chars, mas geradores retornam formatos multi-rede variáveis (`0x...`, base58, Bitcoin) | alta | `pkg/wallet/types.go:76`; geradores Bitcoin/Solana | 🟢 |
| `pkg/wallet.pow()` implementa expoente por loop inteiro e pode ser custoso/impreciso para grandes tentativas | baixa | `pkg/wallet/types.go:190` | 🟢 |
| `BenchmarkModel.Update()` escreve debug em stderr | baixa | `internal/tui/benchmark.go:130` | 🟢 |

## Dicionário resumido de dados

A versão completa está em `_reversa_sdd/data-dictionary.md`.

| Entidade | Módulo | Papel |
|---|---|---|
| `Wallet` | `pkg/wallet` | Carteira gerada com endereço, chaves, mnemonic, rede e timestamp |
| `GenerationCriteria` | `pkg/wallet` | Critérios de busca vanity |
| `GenerationResult` | `pkg/wallet` | Resultado de uma geração, com tentativas, duração e worker |
| `KeyStoreV3` | `internal/crypto` | Estrutura JSON compatível Ethereum KeyStore V3 |
| `CryptoParams` | `internal/crypto/kdf` | Envelope de parâmetros KDF/cipher/MAC |
| `WorkerStats` | `internal/worker` | Métricas por worker |
| `LogEntry` | `pkg/logging` | Evento estruturado de logging seguro |
