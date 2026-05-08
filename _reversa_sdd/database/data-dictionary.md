# Dicionário de Dados — Data Master

> Projeto: `bloco-wallet-generator`  
> Escala: 🟢 DDL/migration direto | 🟡 Inferido de código/artefatos | 🔴 Inacessível

## Sumário executivo

🟢 **CONFIRMADO** — Não existem tabelas, coleções, migrations, DDL, modelos ORM ou conexão de banco usados diretamente pela aplicação.

Este dicionário documenta:

- **0 tabelas físicas de banco**
- **0 coleções NoSQL**
- **Artefatos locais persistidos em filesystem** usados para keystores, senhas, mnemonics, chaves Solana e logs legados

## Inventário de tabelas/coleções

| Nome | Tipo | Propósito | Confiança |
|---|---|---|---:|
| N/A | Tabela SQL | Não aplicável; não há banco SQL | 🟢 |
| N/A | Coleção NoSQL | Não aplicável; não há uso direto de NoSQL | 🟢 |

## Artefatos persistidos em filesystem

### `keystores/<endereco>.json` — KeyStore V3 / keypair Solana

| Atributo | Tipo/Formato | Nullable | Default | Regra | Evidência | Confiança |
|---|---|---:|---|---|---|---:|
| `path` | arquivo local | não | `./keystores` | Diretório configurável por `--keystore-dir`/`BLOCO_KEYSTORE_DIR` | `internal/config/config.go`, `internal/cli/commands.go` | 🟢 |
| `mode` | permissão Unix | não | `0600` | Escrita atômica valida permissão final | `writeFileAtomic`, `ValidateFilePermissions` | 🟢 |
| `address` | string | não | N/A | Ethereum normaliza sem `0x` no JSON; outras redes preservam endereço | `GenerateKeyStore` | 🟢 |
| `crypto` | objeto | não | N/A | Contém parâmetros criptográficos do KeyStore V3 | `KeyStoreV3`, `KeyStoreCrypto` | 🟢 |
| `id` | UUID string | não | gerado | Identificador do keystore | `KeyStoreV3` | 🟢 |
| `version` | int | não | `3` | Versão KeyStore V3 | `KeyStoreV3` | 🟢 |

### `KeyStoreCrypto`

| Campo JSON | Tipo | Nullable | Default/valores | Regra | Confiança |
|---|---|---:|---|---|---:|
| `cipher` | string | não | `aes-128-ctr` | Cipher padrão do serviço | 🟢 |
| `ciphertext` | string hex | não | N/A | Chave privada cifrada | 🟢 |
| `cipherparams.iv` | string hex | não | gerado | IV do AES-CTR | 🟢 |
| `kdf` | string | não | `scrypt` | Aceita `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512` na configuração | 🟢 |
| `kdfparams` | objeto | não | depende do KDF | Parâmetros validados pelo serviço de KDF | 🟢 |
| `mac` | string hex | não | gerado | Código de integridade | 🟢 |

### `ScryptParams`

| Campo JSON | Tipo | Nullable | Regra | Confiança |
|---|---|---:|---|---:|
| `dklen` | int | não | 16 a 128; default usual 32 | 🟢 |
| `n` | int | não | positivo, potência de 2; faixa documentada 1024 a 67108864 | 🟢 |
| `p` | int | não | 1 a 16 | 🟢 |
| `r` | int | não | 1 a 1024 | 🟢 |
| `salt` | string hex | não | não vazio | 🟢 |

### `PBKDF2Params`

| Campo JSON | Tipo | Nullable | Regra | Confiança |
|---|---|---:|---|---:|
| `dklen` | int | não | 16 a 128; default usual 32 | 🟢 |
| `c` | int | não | 1000 a 10000000 | 🟢 |
| `prf` | string | não | `hmac-sha256` ou `hmac-sha512` | 🟢 |
| `salt` | string hex | não | não vazio | 🟢 |

### `keystores/<endereco>.pwd` — senha do keystore

| Atributo | Tipo/Formato | Nullable | Default | Regra | Evidência | Confiança |
|---|---|---:|---|---|---|---:|
| `path` | arquivo local | não | `./keystores/<endereco>.pwd` | Ethereum usa prefixo `0x` no nome | `saveEthereumKeyStore` | 🟢 |
| `conteúdo` | string | não | senha gerada | Senha gerada por `PasswordGenerator` | `GenerateKeyStore` | 🟢 |
| `mode` | permissão Unix | não | `0600` | Escrita atômica e validação final | `writeFileAtomic` | 🟢 |

### `keystores/<endereco>.mnemonic` — mnemonic

| Atributo | Tipo/Formato | Nullable | Default | Regra | Evidência | Confiança |
|---|---|---:|---|---|---|---:|
| `path` | arquivo local | não | `./keystores/<endereco>.mnemonic` | Nome usa endereço formatado por rede | `SaveMnemonicFile` | 🟢 |
| `conteúdo` | string BIP-39 | não | N/A | Não pode ser vazio | `SaveMnemonicFile` | 🟢 |
| `mode` | permissão Unix | não | `0600` | Escrita atômica e validação final | `writeFileAtomic`, `ValidateFilePermissions` | 🟢 |

### `keystores/<endereco>.key` — chave privada Solana legado atual

| Atributo | Tipo/Formato | Nullable | Default | Regra | Evidência | Confiança |
|---|---|---:|---|---|---|---:|
| `path` | arquivo local | não | `./keystores/<endereco>.key` | Usado para Solana no comportamento atual | `SavePrivateKeyFile` | 🟢 |
| `conteúdo` | string hex | não | N/A | Não pode ser vazio | `SavePrivateKeyFile` | 🟢 |
| `mode` | permissão Unix | não | `0600` | Escrita atômica | `writeFileAtomic` | 🟢 |
| `status futuro` | decisão de produto | não | N/A | Deve migrar para formato criptografado/seguro, sem `.key` bruto | Resposta do Revisor | 🟢 |

### `wallets-YYYYMMDD.log` — log legado de carteiras

| Atributo | Tipo/Formato | Nullable | Default | Regra | Evidência | Confiança |
|---|---|---:|---|---|---|---:|
| `path` | arquivo local | não | diretório atual | Nome diário por `time.Now().Format("20060102")` | `pkg/wallet/logger.go` | 🟢 |
| `mode` | permissão Unix | não | `0644` | Criado com `os.OpenFile(..., 0644)` | `NewWalletLogger` | 🟢 |
| `timestamp` | RFC3339 | não | momento do log | Gravado em cada linha | `LogWallet` | 🟢 |
| `address` | string | não | N/A | Endereço gerado | `LogWallet` | 🟢 |
| `public_key` | string | sim | N/A | Chave pública | `LogWallet` | 🟢 |
| `private_key` | string | não | N/A | Chave privada em claro no legado atual | `LogWallet` | 🟢 |
| `attempts` | int64 | não | N/A | Tentativas até geração | `LogWallet` | 🟢 |
| `duration` | string/duration | não | N/A | Duração da geração | `LogWallet` | 🟢 |
| `status futuro` | decisão de produto | não | N/A | Deve migrar para logging seguro/sanitizado | Resposta do Revisor | 🟢 |

## Campos de domínio em memória relevantes

### `pkg/wallet.Wallet`

| Campo | Tipo Go | Persistência direta | Sensível | Observação | Confiança |
|---|---|---:|---:|---|---:|
| `Address` | `string` | sim, em nomes/conteúdo de arquivos | não | Formato depende da rede | 🟢 |
| `PublicKey` | `string` | sim, no log legado | não | Pode estar vazio dependendo da rede/fluxo | 🟢 |
| `PrivateKey` | `string` | sim, em keystore cifrado; legado grava em `.key`/log | sim | Deve ser protegido | 🟢 |
| `Mnemonic` | `string` | sim, em `.mnemonic` | sim | Usado quando habilitado | 🟢 |
| `Network` | `string` | indireta | não | `ethereum`, `bitcoin`, `solana` | 🟢 |
| `CreatedAt` | `time.Time` | não diretamente | não | Validação exige não-zero | 🟢 |

## Índices, PKs, FKs e constraints

| Item | Status | Confiança |
|---|---|---:|
| Primary keys | Não aplicável; não há tabelas | 🟢 |
| Foreign keys | Não aplicável; não há tabelas | 🟢 |
| Índices | Não aplicável; não há tabelas | 🟢 |
| Constraints de banco | Não aplicável; regras vivem no código | 🟢 |
| Unicidade de endereço | Não há constraint persistida; sobrescrita de arquivo é possível pelo mesmo caminho | 🟡 |
