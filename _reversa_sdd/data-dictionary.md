# Data Dictionary — Archaeologist

> Projeto: `bloco-wallet-generator`  
> Confiança: 🟢 CONFIRMADO, extraído das structs Go.

## `pkg/wallet.Wallet`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Address` | `string` | sim | Endereço gerado. Ethereum pode aparecer com `0x` em alguns fluxos; validação local espera 40 chars em `IsValid()`. |
| `PublicKey` | `string` | não | Chave pública em hexadecimal quando disponível. |
| `PrivateKey` | `string` | sim | Chave privada em hexadecimal; dado sensível. |
| `Mnemonic` | `string` | não | Frase BIP-39 quando gerada/salva. |
| `Network` | `string` | não | Rede alvo: `ethereum`, `bitcoin`, `solana`. |
| `CreatedAt` | `time.Time` | sim | Timestamp de criação. |

## `pkg/wallet.GenerationCriteria`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Network` | `string` | sim | Rede alvo. Default da CLI: `ethereum`. |
| `Prefix` | `string` | não | Prefixo hexadecimal desejado. |
| `Suffix` | `string` | não | Sufixo hexadecimal desejado. |
| `IsChecksum` | `bool` | sim | Ativa validação EIP-55 para Ethereum. |
| `UseMnemonic` | `bool` | não | Solicita geração com mnemonic; suportado no worker apenas para Ethereum. |
| `MaxAttempts` | `int64` | não | Limite máximo de tentativas; validação apenas impede negativo. |

## `pkg/wallet.GenerationResult`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Wallet` | `*Wallet` | não | Carteira encontrada. |
| `Attempts` | `int64` | sim | Tentativas feitas pelo worker vencedor. |
| `Duration` | `time.Duration` | sim | Duração até encontrar resultado. |
| `Error` | `error` | não | Erro da geração. |
| `WorkerID` | `int` | não | ID do worker que encontrou a carteira. |

## `pkg/wallet.GenerationStats`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Pattern` | `string` | sim | Prefixo+sufixo concatenados. |
| `Difficulty` | `float64` | sim | Dificuldade estimada. |
| `Probability50` | `int64` | sim | Tentativas para 50% de chance. |
| `CurrentAttempts` | `int64` | sim | Tentativas atuais. |
| `Speed` | `float64` | sim | Endereços por segundo. |
| `Probability` | `float64` | sim | Probabilidade acumulada. |
| `EstimatedTime` | `time.Duration` | não | ETA estimado. |
| `StartTime` | `time.Time` | sim | Início da medição. |
| `LastUpdate` | `time.Time` | sim | Última atualização. |
| `IsChecksum` | `bool` | sim | Indica impacto de checksum. |

## `pkg/wallet.BenchmarkResult`

| Campo | Tipo | Descrição |
|---|---|---|
| `TotalAttempts` | `int64` | Total de tentativas amostradas. |
| `TotalDuration` | `time.Duration` | Duração total. |
| `AverageSpeed`, `MinSpeed`, `MaxSpeed` | `float64` | Estatísticas de velocidade. |
| `SpeedSamples` | `[]float64` | Amostras periódicas de velocidade. |
| `DurationSamples` | `[]time.Duration` | Durações das amostras. |
| `ThreadCount` | `int` | Número de workers. |
| `ScalabilityEfficiency` | `float64` | Eficiência multi-thread. |
| `ThreadBalanceScore` | `float64` | Balanceamento de threads. |
| `SpeedupVsSingleThread` | `float64` | Speedup estimado. |
| `AmdahlsLawLimit` | `float64` | Limite teórico calculado. |

## `internal/config.Config`

| Campo | Tipo | Descrição |
|---|---|---|
| `Worker` | `WorkerConfig` | Threads, batch e health/shutdown. |
| `TUI` | `TUIConfig` | Habilitação, refresh, dimensões e suporte visual. |
| `Crypto` | `CryptoConfig` | Pool, random seguro, hashing e limpeza de memória. |
| `CLI` | `CLIConfig` | Verbose/quiet e intervalo de progresso. |
| `KeyStore` | `KeyStoreConfig` | Output, KDF, file mode, análise e nível de segurança. |
| `Logging` | `LoggingConfig` | Nível, formato, arquivo, rotação e buffer. |

## `internal/crypto.KeyStoreV3`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Address` | `string` | sim | Endereço associado ao keystore. |
| `Crypto` | `KeyStoreCrypto` | sim | Parâmetros criptográficos. |
| `ID` | `string` | sim | UUID do keystore. |
| `Version` | `int` | sim | Deve ser `3`. |

## `internal/crypto.KeyStoreCrypto`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `Cipher` | `string` | sim | Deve ser `aes-128-ctr`. |
| `CipherText` | `string` | sim | Private key cifrada em hex. |
| `CipherParams` | `CipherParams` | sim | IV do AES-CTR. |
| `KDF` | `string` | sim | `scrypt` ou `pbkdf2`. |
| `KDFParams` | `interface{}` | sim | `ScryptParams` ou `PBKDF2Params`. |
| `MAC` | `string` | sim | Keccak MAC. |

## `internal/crypto.ScryptParams`

| Campo | Tipo | Regra |
|---|---|---|
| `DKLen` | `int` | 16 a 128, default 32. |
| `N` | `int` | potência de 2, 1024 a 67108864. |
| `P` | `int` | 1 a 16. |
| `R` | `int` | 1 a 1024. |
| `Salt` | `string` | Hex não vazio. |

## `internal/crypto.PBKDF2Params`

| Campo | Tipo | Regra |
|---|---|---|
| `DKLen` | `int` | 16 a 128, default 32. |
| `C` | `int` | 1000 a 10000000. |
| `PRF` | `string` | `hmac-sha256` ou `hmac-sha512`. |
| `Salt` | `string` | Hex não vazio. |

## `internal/crypto/kdf.CryptoParams`

| Campo | Tipo | Descrição |
|---|---|---|
| `KDF` | `string` | Nome do KDF, com aliases normalizados pelo serviço. |
| `KDFParams` | `map[string]interface{}` | Parâmetros de derivação, incluindo salt. |
| `Cipher` | `string` | Cipher usado no keystore. |
| `CipherText` | `string` | Texto cifrado. |
| `CipherParams` | `map[string]interface{}` | Parâmetros do cipher, como IV. |
| `MAC` | `string` | Código de integridade. |

## `internal/worker.WorkerStats`

| Campo | Tipo | Descrição |
|---|---|---|
| `WorkerID` | `int` | Identificador do worker. |
| `Attempts` | `int64` | Tentativas acumuladas. |
| `Speed` | `float64` | Tentativas por segundo. |
| `LastUpdate` | `time.Time` | Última atualização. |
| `IsHealthy` | `bool` | Status de saúde. |
| `ErrorCount` | `int` | Total de erros. |
| `LastError` | `string` | Último erro textual. |

## `pkg/errors.BlocoError`

| Campo | Tipo | Descrição |
|---|---|---|
| `Type` | `ErrorType` | Categoria: validation, crypto, worker, configuration, tui, generation, timeout, cancellation. |
| `Operation` | `string` | Operação que falhou. |
| `Message` | `string` | Mensagem de domínio. |
| `Cause` | `error` | Erro original. |
| `Context` | `map[string]interface{}` | Dados adicionais. |
| `Timestamp` | `time.Time` | Momento do erro. |
| `Stack` | `[]string` | Stack opcional. |

## `pkg/logging.LogEntry`

| Campo | Tipo | Descrição |
|---|---|---|
| `Timestamp` | `time.Time` | Timestamp UTC. |
| `Level` | `LogLevel` | ERROR/WARN/INFO/DEBUG. |
| `Message` | `string` | Mensagem. |
| `Operation` | `string` | Operação opcional. |
| `ThreadID` | `int` | Worker/thread opcional. |
| `Fields` | `map[string]interface{}` | Campos sanitizados. |
| `Error` | `string` | Erro sanitizado. |
