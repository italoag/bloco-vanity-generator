# Regras de Negócio no Banco — Data Master

> Projeto: `bloco-wallet-generator`  
> Escala: 🟢 DDL/migration direto | 🟡 Inferido de código/artefatos | 🔴 Inacessível

## Conclusão

🟢 **CONFIRMADO** — Não há regras de negócio implementadas em banco de dados porque o projeto não possui banco, DDL, migrations, triggers, procedures, views ou constraints persistidas.

As regras abaixo são regras de persistência local e validação implementadas no código, não no banco.

## Regras de banco inexistentes

| Tipo | Quantidade | Status | Confiança |
|---|---:|---|---:|
| Check constraints | 0 | Não aplicável | 🟢 |
| Unique constraints | 0 | Não aplicável | 🟢 |
| Foreign keys | 0 | Não aplicável | 🟢 |
| Triggers | 0 | Não aplicável | 🟢 |
| Views/materialized views | 0 | Não aplicável | 🟢 |
| Stored procedures/funções | 0 | Não aplicável | 🟢 |

## Regras de persistência local

### DBR-FS-001 — Diretório de keystores configurável

| Campo | Valor |
|---|---|
| Regra | Diretório padrão de saída é `./keystores`, configurável por flag/env. |
| Evidência | `DefaultConfig`, `--keystore-dir`, `BLOCO_KEYSTORE_DIR` |
| Confiança | 🟢 |

### DBR-FS-002 — Keystore pode ser desabilitado

| Campo | Valor |
|---|---|
| Regra | Quando `KeyStore.Enabled=false`, geração/salvamento de keystore é ignorada ou bloqueada conforme método chamado. |
| Evidência | `SaveKeyStoreFiles`, `SaveKeyStoreFilesToDisk`, `SaveMnemonicFile` |
| Confiança | 🟢 |

### DBR-FS-003 — Escrita atômica de arquivos sensíveis

| Campo | Valor |
|---|---|
| Regra | Arquivos sensíveis são escritos por arquivo temporário, `Sync`, `Close`, `Rename` e validação de permissão. |
| Evidência | `writeFileAtomic` |
| Confiança | 🟢 |

### DBR-FS-004 — Permissões esperadas para keystore, senha e mnemonic

| Campo | Valor |
|---|---|
| Regra | Arquivos `*.json`, `*.pwd`, `*.mnemonic` e `*.key` são gravados com permissão `0600`. |
| Evidência | `saveEthereumKeyStore`, `SaveMnemonicFile`, `SavePrivateKeyFile`, `writeFileAtomic` |
| Confiança | 🟢 |

### DBR-FS-005 — Diretório criado com permissão padrão ampla

| Campo | Valor |
|---|---|
| Regra | Diretórios necessários são criados com `0755`. |
| Evidência | `ensureOutputDirectory`, `writeFileAtomic` |
| Confiança | 🟢 |

### DBR-FS-006 — Ethereum gera KeyStore V3 e senha

| Campo | Valor |
|---|---|
| Regra | Para `network=ethereum` ou vazio, salva `<0x-address>.json` e `<0x-address>.pwd`. |
| Evidência | `SaveKeyStoreFilesToDisk`, `saveEthereumKeyStore` |
| Confiança | 🟢 |

### DBR-FS-007 — Bitcoin não gera KeyStore V3

| Campo | Valor |
|---|---|
| Regra | Para `network=bitcoin`, fluxo de keystore retorna erro ou ignora KeyStore V3; backup deve usar mnemonic. |
| Evidência | `SaveKeyStoreFiles`, `SaveKeyStoreFilesToDisk`, `generateAndSaveKeystoreWithVerbose` |
| Confiança | 🟢 |

### DBR-FS-008 — Solana grava JSON e `.key` no comportamento atual

| Campo | Valor |
|---|---|
| Regra | Para `network=solana`, código atual salva JSON de keypair e arquivo `.key` com private key. |
| Evidência | `SaveKeyStoreFilesToDisk`, `saveSolanaKeypair`, `SavePrivateKeyFile` |
| Confiança | 🟢 |

### DBR-FS-009 — Decisão humana substitui `.key` bruto de Solana

| Campo | Valor |
|---|---|
| Regra | Persistência Solana futura deve usar formato criptografado/seguro e evitar `.key` bruto. |
| Evidência | `_reversa_sdd/questions.md`, `_reversa_sdd/gaps.md`, specs revisadas | 
| Confiança | 🟢 |

### DBR-FS-010 — Validação de endereço por rede antes de persistir

| Campo | Valor |
|---|---|
| Regra | Persistência valida endereço conforme rede antes de salvar. |
| Evidência | `validateAddressForNetwork`, `validateEthereumAddress`, `validateBitcoinAddress`, `validateSolanaAddress` |
| Confiança | 🟢 |

### DBR-FS-011 — Senha de keystore não pode ser vazia

| Campo | Valor |
|---|---|
| Regra | `saveEthereumKeyStore` rejeita senha vazia. |
| Evidência | `saveEthereumKeyStore` |
| Confiança | 🟢 |

### DBR-FS-012 — Mnemonic não pode ser vazio ao salvar

| Campo | Valor |
|---|---|
| Regra | `SaveMnemonicFile` rejeita mnemonic vazio. |
| Evidência | `SaveMnemonicFile` |
| Confiança | 🟢 |

### DBR-FS-013 — Logger legado grava material sensível no comportamento atual

| Campo | Valor |
|---|---|
| Regra | `WalletLogger` grava `PrivateKey` em claro em `wallets-YYYYMMDD.log` com permissão `0644`. |
| Evidência | `pkg/wallet/logger.go` |
| Confiança | 🟢 |

### DBR-FS-014 — Decisão humana exige logging seguro

| Campo | Valor |
|---|---|
| Regra | `WalletLogger` legado deve migrar para logging seguro/sanitizado sem private key em claro. |
| Evidência | `_reversa_sdd/questions.md`, `_reversa_sdd/pkg/logging/requirements.md`, `_reversa_sdd/pkg/wallet/requirements.md` |
| Confiança | 🟢 |

## Regras equivalentes a constraints, implementadas no código

| Constraint lógica | Onde vive | Regra | Confiança |
|---|---|---|---:|
| `address` obrigatório | `GenerateKeyStore`, `validateAddressForNetwork` | Endereço vazio é erro | 🟢 |
| `private_key` obrigatória | `GenerateKeyStore`, `SavePrivateKeyFile` | Chave privada vazia é erro | 🟢 |
| KDF permitido | `Config.Validate` | `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512` | 🟢 |
| Security level permitido | `Config.Validate` | `low`, `medium`, `high`, `very-high` | 🟢 |
| Quiet/verbose mutuamente exclusivos | `Config.Validate` | Não podem estar ativos simultaneamente | 🟢 |
| Permissões de arquivo sensível | `ValidateFilePermissions` | Esperado `0600` | 🟢 |

## Riscos de regras fora do banco

| Risco | Descrição | Severidade documental | Confiança |
|---|---|---:|---:|
| Sem transação real | Múltiplos arquivos são coordenados no código, não por transação de banco | Média | 🟢 |
| Sem unicidade persistida | Mesmo endereço deriva o mesmo path e pode sobrescrever artefatos | Média | 🟡 |
| Material sensível fora de cofre | Senha, mnemonic e private key legado são arquivos locais | Alta | 🟢 |
| Diretório `0755` | Conteúdo é `0600`, mas diretório pode listar nomes por outros usuários no mesmo host | Média | 🟢 |
