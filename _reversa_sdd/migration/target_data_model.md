---
schemaVersion: 1
generatedAt: 2026-05-08T11:44:00Z
reversa:
  version: "1.2.34"
kind: target_data_model
producedBy: designer
hash: "sha256:12d33202440e09404b3e795c2fcbf3b7d3a761843640ca156e017980619aed5b"
---

# Target Data Model

> Modelo de dados do sistema novo. Não há banco de dados alvo; o modelo descreve entidades em memória e artefatos persistidos em filesystem local.

## Visão geral

O sistema alvo não usa banco SQL, NoSQL, ORM, migrations, event store ou outbox. A persistência continua local em filesystem, porque `migration_brief.md`, Data Master e `target_business_rules.md` confirmam CLI local sem backend/banco obrigatório. Os dados persistidos são artefatos sensíveis ou operacionais: KeyStore V3 Ethereum, senha gerada, mnemonic, artefato Solana seguro, logs sanitizados e release assets.

O modelo alvo substitui dois formatos inseguros do legado: `.key` bruto Solana e `wallets-YYYYMMDD.log` com private key. A identidade persistida continua derivada do endereço/caminho de arquivo, mas o alvo deve evitar sobrescrita acidental e aplicar permissões restritas.

## Entidades de dados

| Entidade | Tabela / coleção | Aggregate dono | PK | Bounded context |
|---|---|---|---|---|
| `KeyStoreV3File` | arquivo `keystores/<address>.json` | AGG-05 `SecureArtifactSet` | path/address | BC-05 Cofre Local e Artefatos |
| `PasswordFile` | arquivo `keystores/<address>.pwd` | AGG-05 `SecureArtifactSet` | path/address | BC-05 Cofre Local e Artefatos |
| `MnemonicFile` | arquivo `keystores/<address>.mnemonic` | AGG-05 `SecureArtifactSet` | path/address | BC-05 Cofre Local e Artefatos |
| `SolanaSecureArtifactFile` | arquivo seguro Solana, extensão alvo definida pela implementação | AGG-05 `SecureArtifactSet` | path/address | BC-05 Cofre Local e Artefatos |
| `SanitizedLogFile` | arquivo de log sanitizado | AGG-07 `SecureTelemetry` | date/path | BC-06 Observabilidade Segura |
| `GenerationResult` | memória | AGG-03 `WalletGeneration` | não persistido | BC-03 Geração Vanity |
| `WorkerStats` | memória | AGG-03 `WalletGeneration` | workerID durante execução | BC-03 Geração Vanity |
| `BenchmarkResult` | stdout/log/relatório opcional | AGG-03 `WalletGeneration` | execução | BC-03 Geração Vanity |
| `ReleaseAsset` | GitHub Release / GHCR | AGG-08 `ReleaseContract` | version+target | BC-07 Distribuição e Qualidade |

## Schema (DDL ou equivalente)

Não há DDL. Os schemas equivalentes são formatos de arquivo e estruturas JSON.

### `KeyStoreV3File` — `keystores/<address>.json`

```json
{
  "address": "<ethereum-address-without-0x-or-network-specific-address>",
  "crypto": {
    "cipher": "aes-128-ctr",
    "ciphertext": "<hex-ciphertext>",
    "cipherparams": {
      "iv": "<hex-iv>"
    },
    "kdf": "scrypt | pbkdf2 | pbkdf2-sha256 | pbkdf2-sha512",
    "kdfparams": {
      "dklen": 32,
      "salt": "<hex-salt>"
    },
    "mac": "<hex-keccak-mac>"
  },
  "id": "<uuid>",
  "version": 3
}
```

- **Permissão alvo**: `0600`.
- **Escrita alvo**: atômica quando suportada pelo filesystem.
- **Sensibilidade**: contém private key cifrada; nunca logar `ciphertext`, `salt`, `iv` ou `mac` em logs operacionais.
- **Origem**: Data Master `keystores/<endereco>.json`; BR-MIGRAR-017, BR-MIGRAR-019, BR-MIGRAR-021, BR-MIGRAR-023.

### `PasswordFile` — `keystores/<address>.pwd`

```text
<generated-password>
```

- **Permissão alvo**: `0600`.
- **Regra**: senha com mínimo 12 caracteres, cobrindo minúscula, maiúscula, número e especial.
- **Sensibilidade**: segredo; nunca logar nem incluir em eventos de observabilidade.
- **Origem**: Data Master `.pwd`; BR-MIGRAR-018, BR-MIGRAR-021, BR-MIGRAR-023.

### `MnemonicFile` — `keystores/<address>.mnemonic`

```text
<bip39 mnemonic words>
```

- **Permissão alvo**: `0600`.
- **Regra**: salvar quando presente e aplicável; Bitcoin depende de mnemonic para backup.
- **Sensibilidade**: segredo; nunca logar.
- **Origem**: Data Master `.mnemonic`; BR-MIGRAR-015, BR-MIGRAR-021, BR-MIGRAR-023.

### `SolanaSecureArtifactFile` — formato seguro alvo

```json
{
  "network": "solana",
  "address": "<base58-public-address>",
  "crypto": {
    "cipher": "aes-128-ctr",
    "ciphertext": "<encrypted-ed25519-private-material>",
    "cipherparams": {
      "iv": "<hex-iv>"
    },
    "kdf": "scrypt | pbkdf2 | pbkdf2-sha256 | pbkdf2-sha512",
    "kdfparams": {
      "dklen": 32,
      "salt": "<hex-salt>"
    },
    "mac": "<integrity-mac>"
  },
  "version": 1
}
```

- **Permissão alvo**: `0600`.
- **Regra**: substituir `.key` bruto legado por artefato criptografado/seguro.
- **Decisão de formato**: o Designer define o contrato conceitual; a implementação deve escolher extensão/nome final e garantir round-trip/importação em testes.
- **Origem**: Data Master `.key` legado; BR-MIGRAR-016; `discard_log.md` BR-DESCARTAR-005.

### `SanitizedLogFile`

```jsonl
{"timestamp":"<rfc3339>","level":"info|warn|error","operation":"<name>","message":"<text>","fields":{"address":"<public address>","attempts":123,"duration_ms":456,"thread_id":1,"status":"success"}}
```

- **Permissão alvo**: não sensível por conteúdo; recomendável `0644` ou mais restritivo conforme ambiente.
- **Whitelist permitida**: endereço público, tentativas, duração, thread, status, operação, erro sanitizado.
- **Campos proibidos**: private key, public key, mnemonic, password, salt, ciphertext, IV, MAC, material KDF sensível.
- **Origem**: `pkg/logging`, `wallets-YYYYMMDD.log` legado substituído; BR-MIGRAR-024..025; `discard_log.md` BR-DESCARTAR-006.

## Relacionamentos

| Origem | Destino | Cardinalidade | Integridade | Notas |
|---|---|---|---|---|
| `NetworkWallet(network=ethereum)` | `KeyStoreV3File` | 1:0..1 | aplicação/filesystem | Criado quando persistência/keystore está habilitada. |
| `NetworkWallet(network=ethereum)` | `PasswordFile` | 1:0..1 | aplicação/filesystem | Deve ser criado junto ao KeyStore; falha exige cleanup ou warning consistente. |
| `NetworkWallet(mnemonic present)` | `MnemonicFile` | 1:0..1 | aplicação/filesystem | Aplicável quando mnemonic existe e backup é configurado. |
| `NetworkWallet(network=bitcoin)` | `MnemonicFile` | 1:0..1 | aplicação/filesystem | Bitcoin salva backup por mnemonic, não KeyStore V3. |
| `NetworkWallet(network=solana)` | `SolanaSecureArtifactFile` | 1:0..1 | aplicação/filesystem | Substitui `.key` bruto. |
| `GenerationResult` | `SanitizedLogFile` | N:0..1 | append-only lógico | Log opcional e sanitizado. |
| `ReleaseContract` | `ReleaseAsset` | 1:N | GitHub/GHCR | Assets por target e checksums. |

## Restrições

- **Unicidade**: path derivado do endereço deve ser tratado como único dentro do diretório de saída; implementação deve evitar sobrescrita silenciosa ou documentar política explícita.
- **Integridade referencial**: não há FK; relações são garantidas pela aplicação durante a escrita dos arquivos.
- **Particionamento / sharding**: não aplicável.
- **Índices críticos**: não aplicável.
- **Permissões sensíveis**: `KeyStoreV3File`, `PasswordFile`, `MnemonicFile` e `SolanaSecureArtifactFile` devem usar `0600` quando o sistema operacional suportar.
- **Atomicidade**: escrita atômica desejada para arquivos sensíveis; em conjuntos multi-arquivo, falhas parciais devem produzir warning e cleanup quando seguro.
- **Não vazamento**: nenhum arquivo de log pode conter segredo ou material criptográfico sensível.
- **Sem `.key` bruto Solana**: proibido no alvo.

## Considerações específicas do paradigma alvo

- **Go idiomático**: schemas de arquivo devem ser representados por structs simples com validação explícita e testes de round-trip.
- **CSP/goroutines**: dados de execução concorrente (`WorkerStats`, progresso) são em memória e devem evitar races; não há persistência intermediária obrigatória.
- **Interfaces leves**: filesystem, clock/random e logging devem ser portas testáveis para permitir parity tests e testes negativos de vazamento.
- **Sem event-driven distribuído**: eventos de domínio são internos/logáveis; não há outbox, broker ou event store.

## Origem no legado

| Tabela / coleção nova | Origem no legado | Transformação |
|---|---|---|
| N/A — sem banco | Data Master confirmou 0 tabelas/coleções | preserva ausência de banco |
| `KeyStoreV3File` | `keystores/<endereco>.json` | preservado com schema/segurança explicitados |
| `PasswordFile` | `keystores/<endereco>.pwd` | preservado com permissão e segredo explícitos |
| `MnemonicFile` | `keystores/<endereco>.mnemonic` | preservado com regra por rede |
| `SolanaSecureArtifactFile` | `keystores/<endereco>.key` bruto e JSON simplificado Solana | substituído por formato criptografado/seguro |
| `SanitizedLogFile` | `wallets-YYYYMMDD.log` e `pkg/logging` | substitui log inseguro por whitelist sanitizada |
| `ReleaseAsset` | GitHub Releases/GHCR do legado | preservado/renomeado para `bloco-vgen` |

## Notas

O nome final do arquivo Solana seguro deve ser decidido na implementação com testes de round-trip. A restrição arquitetural é clara: não salvar chave Solana bruta `.key`. O Inspector deve criar testes de paridade que aceitem essa divergência intencional em relação ao legado.
