# ERD Complete — Architect

> O sistema não possui banco de dados. Este ERD representa entidades de domínio em memória e artefatos persistidos em filesystem.

## Diagrama ERD

```mermaid
erDiagram
  GENERATION_CRITERIA ||--o{ GENERATION_RESULT : "orienta"
  GENERATION_RESULT ||--o| WALLET : "contém"
  GENERATION_RESULT }o--|| WORKER_STATS : "produzido_por"
  GENERATION_STATS ||--|| GENERATION_CRITERIA : "resume"
  BENCHMARK_RESULT ||--o{ SPEED_SAMPLE : "contém"
  WALLET ||--o| KEYSTORE_V3 : "pode_gerar"
  WALLET ||--o| MNEMONIC_FILE : "pode_salvar"
  KEYSTORE_V3 ||--|| KEYSTORE_CRYPTO : "contém"
  KEYSTORE_CRYPTO ||--|| CIPHER_PARAMS : "usa"
  KEYSTORE_CRYPTO ||--o| SCRYPT_PARAMS : "usa_scrypt"
  KEYSTORE_CRYPTO ||--o| PBKDF2_PARAMS : "usa_pbkdf2"
  KEYSTORE_CRYPTO ||--o| CRYPTO_PARAMS : "mapeia_para"
  LOG_ENTRY }o--o| BLOCO_ERROR : "pode_registrar"
  CONFIG ||--|| WORKER_CONFIG : "contém"
  CONFIG ||--|| TUI_CONFIG : "contém"
  CONFIG ||--|| CRYPTO_CONFIG : "contém"
  CONFIG ||--|| KEYSTORE_CONFIG : "contém"
  CONFIG ||--|| LOGGING_CONFIG : "contém"

  GENERATION_CRITERIA {
    string Network
    string Prefix
    string Suffix
    bool IsChecksum
    bool UseMnemonic
    int64 MaxAttempts
  }

  WALLET {
    string Address
    string PublicKey
    string PrivateKey
    string Mnemonic
    string Network
    time CreatedAt
  }

  GENERATION_RESULT {
    int64 Attempts
    duration Duration
    error Error
    int WorkerID
  }

  GENERATION_STATS {
    string Pattern
    float Difficulty
    int64 Probability50
    int64 CurrentAttempts
    float Speed
    float Probability
    duration EstimatedTime
    time StartTime
    bool IsChecksum
  }

  BENCHMARK_RESULT {
    int64 TotalAttempts
    duration TotalDuration
    float AverageSpeed
    float MinSpeed
    float MaxSpeed
    int ThreadCount
    float ScalabilityEfficiency
    float ThreadBalanceScore
  }

  SPEED_SAMPLE {
    float Speed
    duration Duration
  }

  WORKER_STATS {
    int WorkerID
    int64 Attempts
    float Speed
    time LastUpdate
    bool IsHealthy
    int ErrorCount
    string LastError
  }

  KEYSTORE_V3 {
    string Address
    string ID
    int Version
  }

  KEYSTORE_CRYPTO {
    string Cipher
    string CipherText
    string KDF
    string MAC
  }

  CIPHER_PARAMS {
    string IV
  }

  SCRYPT_PARAMS {
    int DKLen
    int N
    int P
    int R
    string Salt
  }

  PBKDF2_PARAMS {
    int DKLen
    int C
    string PRF
    string Salt
  }

  CRYPTO_PARAMS {
    string KDF
    map KDFParams
    string Cipher
    string CipherText
    map CipherParams
    string MAC
  }

  LOG_ENTRY {
    time Timestamp
    string Level
    string Message
    string Operation
    int ThreadID
    map Fields
    string Error
  }

  BLOCO_ERROR {
    string Type
    string Operation
    string Message
    error Cause
    map Context
    time Timestamp
  }

  CONFIG {
    object Worker
    object TUI
    object Crypto
    object CLI
    object KeyStore
    object Logging
  }

  WORKER_CONFIG {
    int ThreadCount
    int BatchSize
    int ShutdownTimeout
  }

  TUI_CONFIG {
    bool Enabled
    int RefreshRate
  }

  CRYPTO_CONFIG {
    bool SecureRandom
    bool ClearMemory
  }

  KEYSTORE_CONFIG {
    bool Enabled
    string OutputDir
    string KDFAlgorithm
    map KDFParams
    string SecurityLevel
  }

  LOGGING_CONFIG {
    bool Enabled
    string Level
    string Format
    string OutputFile
  }

  MNEMONIC_FILE {
    string Address
    string Mnemonic
    string Network
  }
```

## Entidades e persistência

| Entidade | Natureza | Persistida? | Local/Formato | Confiança |
|---|---|---:|---|---:|
| `Wallet` | Domínio | parcialmente | stdout/TUI, keystore/mnemonic/logs | 🟢 |
| `GenerationCriteria` | Entrada/controle | não | memória | 🟢 |
| `GenerationResult` | Resultado | não diretamente | memória/stdout | 🟢 |
| `GenerationStats` | Métrica runtime | não | memória/TUI | 🟢 |
| `BenchmarkResult` | Métrica runtime | não diretamente | stdout/TUI | 🟢 |
| `WorkerStats` | Métrica runtime | não diretamente | memória/log sanitizado | 🟢 |
| `KeyStoreV3` | Artefato seguro | sim | `keystores/*.json` | 🟢 |
| password file | Artefato sensível | sim | `keystores/*.pwd` | 🟢 |
| mnemonic file | Artefato sensível | sim | `keystores/*.mnemonic` | 🟢 |
| `LogEntry` | Observabilidade | sim | arquivo log/stdout/discard | 🟢 |
| `Config` | Config runtime | não como arquivo app | defaults/env/flags | 🟢 |

## Cardinalidades relevantes

| Relação | Cardinalidade | Observação | Confiança |
|---|---|---|---:|
| `GenerationCriteria` -> `GenerationResult` | 1:N | Um critério pode gerar uma ou várias carteiras (`--count`). | 🟢 |
| `GenerationResult` -> `Wallet` | 1:0..1 | Pode falhar e conter `Error` sem wallet. | 🟢 |
| `Wallet` -> `KeyStoreV3` | 1:0..1 | Depende de `KeyStore.Enabled` e rede. | 🟢 |
| `Wallet` -> mnemonic file | 1:0..1 | Só quando mnemonic existe/salvamento aplicável. | 🟢 |
| `KeyStoreCrypto` -> KDF params | 1:1 | Scrypt ou PBKDF2, mutuamente alternativos. | 🟢 |
| `BenchmarkResult` -> samples | 1:N | Várias amostras de velocidade/duração. | 🟢 |
| `Config` -> subconfigs | 1:1 | Config agregada. | 🟢 |

## Lacunas de modelo

| ID | Lacuna | Confiança |
|---|---|---:|
| ERD-GAP-001 | Não há identificador persistente para geração/execução além de timestamps/logs. | 🟢 |
| ERD-GAP-002 | `Wallet.IsValid()` ainda não modela adequadamente formatos multi-rede. | 🟢 |
| ERD-GAP-003 | Artefatos sensíveis em filesystem não têm lifecycle/expiração modelados. | 🟡 |
