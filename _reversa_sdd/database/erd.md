# ERD — Data Master

> Projeto: `bloco-wallet-generator`  
> Agente: `reversa-data-master`  
> Escala: 🟢 DDL/migration direto | 🟡 Inferido de código/artefatos | 🔴 Inacessível

## Conclusão sobre banco de dados

🟢 **CONFIRMADO** — O projeto não possui banco de dados relacional, NoSQL, schema ORM, migrations ou DDL aplicável.

Evidências usadas:

| Evidência | Resultado | Confiança |
|---|---|---:|
| Busca por `*.sql` | Nenhum arquivo encontrado | 🟢 |
| Busca por migrations/schemas | Nenhum arquivo encontrado | 🟢 |
| Busca por `CREATE TABLE`, `ALTER TABLE`, ORM e drivers de uso direto | Nenhum uso direto encontrado no produto | 🟢 |
| Inventário Reversa | `_reversa_sdd/inventory.md` confirma ausência de DDL, migrations, schema ORM ou diretório de banco | 🟢 |
| Código Go | Persistência implementada por arquivos locais em `internal/crypto/keystore.go` e `pkg/wallet/logger.go` | 🟢 |

## Tabelas documentadas

| Grupo | Quantidade | Observação | Confiança |
|---|---:|---|---:|
| Tabelas relacionais | 0 | Não há banco SQL | 🟢 |
| Coleções NoSQL | 0 | Não há uso direto de MongoDB ou outro banco documental | 🟢 |
| Views/materialized views | 0 | Não há DDL | 🟢 |
| Triggers/procedures/funções de banco | 0 | Não há banco | 🟢 |

## ERD de banco

Como não há tabelas, o ERD físico do banco é vazio.

```mermaid
erDiagram
  NO_DATABASE {
    string status "sem tabelas persistidas em banco"
    string storage "filesystem local"
  }
```

## Modelo de persistência por arquivos

O diagrama abaixo não representa tabelas de banco. Ele documenta os artefatos persistidos em filesystem que cumprem o papel de armazenamento local.

```mermaid
erDiagram
  WALLET ||--o| ETHEREUM_KEYSTORE_JSON : "gera"
  WALLET ||--o| KEYSTORE_PASSWORD_FILE : "gera"
  WALLET ||--o| MNEMONIC_FILE : "pode_gerar"
  WALLET ||--o| SOLANA_KEYPAIR_JSON : "gera_quando_solana"
  WALLET ||--o| SOLANA_PRIVATE_KEY_FILE : "legado_atual"
  WALLET ||--o{ WALLET_LOG_FILE : "pode_ser_registrada"
  ETHEREUM_KEYSTORE_JSON ||--|| KEYSTORE_CRYPTO : "contém"
  KEYSTORE_CRYPTO ||--|| CIPHER_PARAMS : "usa"
  KEYSTORE_CRYPTO ||--o| SCRYPT_PARAMS : "usa_scrypt"
  KEYSTORE_CRYPTO ||--o| PBKDF2_PARAMS : "usa_pbkdf2"

  WALLET {
    string Address
    string PublicKey
    string PrivateKey
    string Mnemonic
    string Network
    time CreatedAt
  }

  ETHEREUM_KEYSTORE_JSON {
    string path "keystores/0x-address.json"
    string address
    string id
    int version
    object crypto
  }

  KEYSTORE_PASSWORD_FILE {
    string path "keystores/0x-address.pwd"
    string password
    int mode "0600"
  }

  MNEMONIC_FILE {
    string path "keystores/address.mnemonic"
    string mnemonic
    int mode "0600"
  }

  SOLANA_KEYPAIR_JSON {
    string path "keystores/address.json"
    string type
    string address
    string note
    int mode "0600"
  }

  SOLANA_PRIVATE_KEY_FILE {
    string path "keystores/address.key"
    string private_key
    int mode "0600"
  }

  WALLET_LOG_FILE {
    string path "wallets-YYYYMMDD.log"
    string timestamp
    string address
    string public_key
    string private_key
    int attempts
    string duration
    int mode "0644"
  }

  KEYSTORE_CRYPTO {
    string cipher
    string ciphertext
    string kdf
    object kdfparams
    string mac
  }

  CIPHER_PARAMS {
    string iv
  }

  SCRYPT_PARAMS {
    int dklen
    int n
    int p
    int r
    string salt
  }

  PBKDF2_PARAMS {
    int dklen
    int c
    string prf
    string salt
  }
```

## Observações de segurança

- **Solana `.key`:** o código atual grava chave privada bruta em arquivo `.key`; a decisão humana do Reviewer determina migrar para formato criptografado/seguro.
- **`WalletLogger`:** o logger legado grava `PrivateKey` em claro com permissão `0644`; a decisão humana do Reviewer determina migrar para logging seguro/sanitizado.
- **Arquivos sensíveis:** `*.json`, `*.pwd`, `*.mnemonic` e `*.key` devem ser tratados como material sensível ou derivado de material sensível.
