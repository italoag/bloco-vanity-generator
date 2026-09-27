# Bloco Vanity Generator

Bloco Vanity Generator is a Go CLI for generating vanity wallet addresses for Ethereum, Bitcoin, and Solana with optional local filesystem backup files.

This README reflects the current codebase behavior. Some flags exist in the CLI before their behavior is fully implemented; those cases are called out explicitly in [Current limitations](#current-limitations).

## What it does

- Generates wallets until the address matches an optional prefix and/or suffix.
- Uses multiple worker goroutines; `--threads 0` auto-detects `runtime.NumCPU()`.
- Supports three network modes through `--network ethereum|bitcoin|solana`.
- Supports Ethereum EIP-55 checksum formatting and validation through `--checksum`.
- Supports exact Ethereum EIP-55 pattern matching through `--checksum --case-sensitive`.
- Saves local backup artifacts by default under `./keystores` unless `--no-keystore` is set.
- Uses secure operational logging that does not log private keys or keystore passwords.

## Requirements

- Go `1.26.8` or newer.
- Git, if building from a clone.
- For encrypted fnox backups: `fnox` `1.35.2` and `age` `1.2.1` (see [Encrypted backups with fnox](#encrypted-backups-with-fnox)). The repository `mise.toml` pins both, so `mise install` provides them.

CI and Docker currently use Go `1.26.8`.

## Build and test

```bash
git clone <repository-url>
cd bloco-vanity-generator

go mod download
go build -o bloco-vgen ./cmd/bloco-vgen
./bloco-vgen --help
```

Useful Make targets:

```bash
make build       # builds ./bloco-vgen
make test        # runs go test -v ./...
make test-unit   # runs go test -short -v ./...
make vet
make fmt
make clean
```

Do not run `make init` on a normal clone; the Go module already exists as `bloco-vgen`.

## Quick usage

Generate an Ethereum wallet whose address starts with `abc`:

```bash
./bloco-vgen --prefix abc --tui=false
```

Generate five Ethereum wallets ending in `123`:

```bash
./bloco-vgen --suffix 123 --count 5 --tui=false
```

Generate an EIP-55 checksum Ethereum wallet matching `DEAD` exactly:

```bash
./bloco-vgen --prefix DEAD --checksum --case-sensitive --tui=false
```

Generate a Bitcoin wallet:

```bash
./bloco-vgen --network bitcoin --prefix 1 --tui=false
```

Generate a Solana wallet:

```bash
./bloco-vgen --network solana --prefix A --tui=false
```

Disable backup files:

```bash
./bloco-vgen --prefix abc --no-keystore --tui=false
```

Use a custom backup directory:

```bash
./bloco-vgen --prefix abc --keystore-dir ./my-keystores --tui=false
```

## Commands

| Command | Behavior |
|---|---|
| `bloco-vgen` | Generates one or more wallets using the root flags. |
| `bloco-vgen stats` | Shows pattern difficulty estimates. |
| `bloco-vgen benchmark` | Runs the current benchmark command. See [Current limitations](#current-limitations). |
| `bloco-vgen version` | Prints version, git commit, and build time. |
| `bloco-vgen backup doctor` | Verifies the fnox backup provider (binary version and encrypt/decrypt roundtrip). |
| `bloco-vgen backup verify <file>` | Decrypts and validates a `*.fnox.toml` backup artifact; prints public metadata only. |
| `bloco-vgen backup export <file>` | Writes plaintext wallet files to a new directory. Requires `--output-dir` and `--allow-plaintext`. |
| `bloco-vgen completion` | Generates shell completion scripts through Cobra. |

Run `./bloco-vgen <command> --help` for the exact flag list exposed by Cobra/Fang.

## Main flags

| Flag | Short | Default | Current behavior |
|---|---:|---:|---|
| `--prefix` | `-p` | `""` | Prefix the generated address must match. |
| `--suffix` | `-s` | `""` | Suffix the generated address must match. |
| `--checksum` | `-c` | `false` | Enables Ethereum EIP-55 checksum mode. |
| `--case-sensitive` | | `false` | Requires exact case matching. Currently valid only with `--checksum` on Ethereum. |
| `--count` | `-n` | `1` | Number of wallets to generate. |
| `--network` | | `ethereum` | Selects `ethereum`, `bitcoin`, or `solana`. |
| `--with-mnemonic` | | `false` | Uses BIP-39 mnemonic derivation for Ethereum generation. Non-Ethereum mnemonic generation is disabled in the worker path. |
| `--threads` | `-t` | `0` | `0` means auto-detect CPU count; positive values set worker count. Config validation rejects more than 128 threads. |
| `--progress` | | `false` | Enables TUI progress when available. Text-mode live progress is limited. |
| `--tui` | | `true` | Enables terminal UI when supported. Use `--tui=false` for plain text output. |
| `--verbose` | `-v` | `false` | Enables verbose output in supported paths. |
| `--quiet` | `-q` | `false` | Suppresses some non-essential output. It is not a secret-redaction guarantee for every output path. |
| `--output` | | `""` | Flag is registered, but wallet output is currently printed to stdout. |
| `--format` | | `text` | Flag is registered, but wallet output formatting is currently not switched to JSON/CSV. |

## Vanity matching rules

The current `GenerationCriteria.Validate()` applies these rules before generation:

- Combined `prefix + suffix` length must be at most `20` characters.
- Prefix and suffix must contain only hexadecimal characters: `0-9`, `a-f`, `A-F`.
- `--case-sensitive` requires `--checksum`.
- `--case-sensitive` is accepted only for Ethereum.

Network-specific matching behavior:

| Network | Address style | Matching behavior |
|---|---|---|
| Ethereum | `0x` + 40 hex characters | Case-insensitive by default. With `--checksum`, the final wallet address is EIP-55 formatted. With `--checksum --case-sensitive`, prefix/suffix must match the EIP-55 address exactly. |
| Bitcoin | Mainnet P2PKH address generated from secp256k1 public key | Matching is case-sensitive. Current input validation still restricts requested patterns to hex characters. |
| Solana | Base58 public key generated from Ed25519 keypair | Matching is case-sensitive. Current input validation still restricts requested patterns to hex characters. |

## Backup artifacts

Two backup stores are available:

- `files` (default): the legacy plaintext layout described below. These artifacts contain secrets.
- `fnox`: encrypts the entire wallet bundle into a single `*.fnox.toml` artifact per wallet. See [Encrypted backups with fnox](#encrypted-backups-with-fnox).

The `files` store is enabled by default and writes to `./keystores`. Use `--no-keystore` to skip it.

All backup files are written with `0600` permissions through atomic temporary-file writes where implemented.

| Network | Files currently written |
|---|---|
| Ethereum | `0x<address>.json` KeyStore V3 file and `0x<address>.pwd` password file. If generated with mnemonic, also `0x<address>.mnemonic`. EIP-55 case is preserved in filenames. |
| Bitcoin | `<address>.mnemonic` only. The generated mnemonic is for backup metadata and is not used to derive the random private key. |
| Solana | `<address>.json` metadata file and `<address>.key` containing the raw private key hex. Treat this as sensitive material. |

### Ethereum KeyStore settings

| Flag | Default | Valid values / behavior |
|---|---:|---|
| `--keystore-dir` | `./keystores` | Output directory. |
| `--keystore-kdf` | `scrypt` | `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512`. |
| `--kdf-params` | auto | JSON parameters validated according to selected KDF. |
| `--security-level` | `medium` | `low`, `medium`, `high`, `very-high`. Used when KDF params are not supplied. |
| `--kdf-analysis` | `false` | Prints KDF compatibility/security analysis after keystore generation. |

Example custom KDF parameters:

```bash
./bloco-vgen --prefix abc \
  --keystore-kdf scrypt \
  --kdf-params '{"n":262144,"r":8,"p":1,"dklen":32}' \
  --tui=false
```

## Encrypted backups with fnox

`--backup-store fnox` encrypts the complete wallet bundle (private key, address, mnemonic when present, and for Ethereum the KeyStore V3 file plus its password) into a single `<network>-<address>.fnox.toml` artifact under `--backup-dir`. This fixes the `files`-mode coverage gaps: a Bitcoin files-mode mnemonic does **not** restore the private key, and the Solana files-mode `.key` is raw plaintext. fnox mode does not migrate existing `files`-mode wallets — it applies to newly generated wallets only.

With fnox selected:

- The private key, mnemonic, and keystore password are never printed; stdout and the TUI carry public metadata plus the confirmed artifact path only (`Encrypted backup confirmed: "<path>"`).
- No `.pwd`, `.mnemonic`, `.key`, or plaintext keystore JSON files are written.
- The artifact is `0600`, contains ciphertext plus public metadata, and is only published after an encrypted write and decrypt-verify roundtrip. If encryption fails before ciphertext exists, nothing recoverable is claimed and the address must not be used.
- With `--count N` in text mode, each wallet is encrypted and confirmed before the next wallet is generated — a failure mid-batch leaves the already-confirmed backups intact and stops generation rather than collecting everything in memory first.

Requirements: `fnox` `1.35.2` and `age` `1.2.1`. The repository `mise.toml` pins both:

```bash
mise install          # installs fnox 1.35.2 and age 1.2.1
mise run test-backup  # runs the opt-in integration tests (BLOCO_FNOX_INTEGRATION=1)
```

See the fnox docs for the [age provider](https://fnox.jdx.dev/providers/age), the [keychain provider](https://fnox.jdx.dev/providers/keychain), and [mise integration](https://fnox.jdx.dev/guide/mise-integration).

### fnox flags

| Flag | Default | Behavior |
|---|---:|---|
| `--backup-store` | `files` | `files` or `fnox`. With `fnox`, `--keystore-dir` is rejected (use `--backup-dir`) and `--no-keystore` is invalid. |
| `--backup-dir` | `./backups` | Directory for encrypted `*.fnox.toml` artifacts. Never overwrites an existing artifact. |
| `--fnox-bin` | `fnox` | Path to the supported fnox executable (must report `1.35.2`). |
| `--age-recipient` | none | `age1...` recipient, repeatable for multiple recipients/offline restore keys. Required for generation and `doctor`. |
| `--age-identity` | `""` | Age identity file used for decryption. Mutually exclusive with `--keychain-service`/`--keychain-account`. |
| `--keychain-service` | `bloco-vgen` | OS keychain service holding the age identity (macOS Keychain). |
| `--keychain-account` | `age-identity` | OS keychain account/item name for the identity. |
| `--backup-timeout` | `30s` | Timeout per fnox invocation. |

### fnox environment variables

| Variable | Effect |
|---|---|
| `BLOCO_BACKUP_STORE` | `files` or `fnox`. |
| `BLOCO_BACKUP_DIR` | Encrypted backup directory. |
| `BLOCO_FNOX_BINARY` | Path to the fnox executable. |
| `BLOCO_AGE_RECIPIENTS` | Comma-separated `age1...` recipients. |
| `BLOCO_AGE_IDENTITY_FILE` | Age identity file path. |

No secret material is accepted through environment variables.

### Generating with encrypted backups

Set the variables to paths you choose (the identity file's parent directory must already exist; `--backup-dir` is created `0700` if missing and an existing directory must not be group/other-accessible):

```bash
RECIPIENT="age1yourrecipient..."        # from: age-keygen -y "$IDENTITY"
IDENTITY="/secure/dir/backup-identity.txt"
BACKUP_DIR="/secure/dir/backups"

go build -o bloco-vgen ./cmd/bloco-vgen
mise exec -- ./bloco-vgen --backup-store fnox \
  --backup-dir "$BACKUP_DIR" \
  --age-recipient "$RECIPIENT" \
  --age-identity "$IDENTITY" \
  --with-mnemonic --no-tui
```

The identity is required at generation time too: each save decrypts what it just wrote to verify the artifact before publishing it. It is not only needed later for `verify`/`export`.

### Identity: file or OS keychain

Decryption needs one age identity. Simplest portable option is an identity file created once, stored outside the repository in a directory you control:

```bash
age-keygen -o /secure/dir/backup-identity.txt   # prints the secret to the file only
age-keygen -y /secure/dir/backup-identity.txt   # prints the public age1... recipient
```

Pass the public recipient via `--age-recipient` at generation time and the identity via `--age-identity` for `verify`/`export`. Keep at least one second recipient or an offline copy of the identity for disaster recovery; losing all identities makes the backups unrecoverable. Do not commit identity files or backups to Git (the repo `.gitignore` excludes `backups/`, `*.fnox.toml`, and `.fnox-pending-*/`).

Alternatively the identity can live in the OS keychain (default service `bloco-vgen`, account `age-identity`). The CLI never provisions or changes keychain items; when `--age-identity` is omitted the backend reads the selected keychain item through fnox, and the OS may prompt for access. Bootstrapping is a manual step you perform yourself. Example bootstrap:

```toml
# bootstrap.toml (you create this file in a location you choose)
[providers.wallet_keychain]
type = "keychain"
service = "bloco-vgen"
```

```bash
fnox --config /absolute/bootstrap.toml set AGE_IDENTITY \
  --provider wallet_keychain --key-name age-identity \
  --from-file /absolute/private-identity.txt
```

The keychain path is not covered by automated tests; treat the bootstrap as a manual acceptance step and verify it with `bloco-vgen backup doctor` before generating wallets.

### Verifying and exporting backups

`backup doctor` also checks that `--backup-dir` exists or can be created (`0700`), is not a symlink, has no group/other permissions, and is writable — no chmod is applied to existing directories. Newly created directory trees get `0700` on every new level, but parent directories you provide are never modified: a public or group-readable ancestor stays as it is, meaning backup directory *names* may be visible to others even though the leaf and files inside block ordinary traversal. Choose a trusted parent location; the private leaf does not defend against a hostile owner/admin manipulating the surrounding namespace.

```bash
RECIPIENT="age1yourrecipient..."
IDENTITY="/secure/dir/backup-identity.txt"
BACKUP_FILE="/absolute/path/to/your-wallet.fnox.toml"   # replace with your artifact path
NEW_EXPORT_DIR="/secure/dir/new-export"                  # must not exist yet; its parent must exist

./bloco-vgen backup doctor --age-recipient "$RECIPIENT" --age-identity "$IDENTITY"
./bloco-vgen backup verify "$BACKUP_FILE" --age-identity "$IDENTITY"
./bloco-vgen backup export "$BACKUP_FILE" \
  --output-dir "$NEW_EXPORT_DIR" --allow-plaintext \
  --age-identity "$IDENTITY"
```

- `backup verify` prints only network/address/ID and a Bitcoin warning when the stored mnemonic has role `unrelated` (it cannot restore the key).
- `backup export` refuses to run without `--allow-plaintext` and requires `--output-dir` to be a new directory that does not exist (its parent must exist). The export writes plaintext secrets: `wallet-backup.json` contains the complete decrypted bundle (private key, mnemonic, keystore password), plus the network-specific files (Ethereum keystore JSON, `.pwd`, and optional `.mnemonic`; Bitcoin `<address>.key`; Solana `<address>.json` + `.key`). Each file is written to a private temporary file inside the export directory, fsynced, and only then published under its final name without replacing existing files, so a failed export never leaves a partial file at a final filename — a crash between staging and publishing may still leave a `.export-tmp-*` file containing plaintext inside the private export directory, which you should delete. Protect or delete the directory after use. The exported Ethereum keystore keeps its generated three-word password, which is a weak human-memorable password: treat the keystore file itself as a secret.
- If a save fails after the final artifact was linked (for example a post-publish check or directory sync fails), nothing at the final path is removed or overwritten automatically: the staged encrypted copy in `.fnox-pending-*` is retained when present, and the error reports both paths with a `publication was not confirmed` warning so you can decide. Point `backup verify`/`backup export` at either file explicitly to inspect or recover it; a blind retry of the same `Save` is a deliberate collision (the existing artifact is never overwritten) rather than an automatic recovery. Artifact identity is re-checked after the directory sync before a receipt is returned, but another writer with access to the directory can mutate a pathname after the last check — the backup directory must be a trusted, exclusive-use location.
- If a save is interrupted after ciphertext was written, a `.fnox-pending-*` directory retains the encrypted `backup.fnox.toml`; point `backup verify`/`backup export` at that file to recover.
- Scope of the no-plaintext guarantee: generation and `verify` never write plaintext secrets, including on failure paths. An explicit `--allow-plaintext` export writes plaintext by design and could leave files behind if the OS prevents cleanup — it reports the error in that case. Encryption at rest is not protection against a compromised process or user session, and zeroization of Go strings is not guaranteed.

## Logging

Operational logging is enabled by default. The secure logger is designed to avoid logging private keys, public keys, seeds, mnemonics, and keystore passwords.

Important: wallet result output on stdout does print private keys for successful generation paths in the default `files` backup mode (`--backup-store fnox` suppresses them). Do not run the CLI in shared terminals or redirect stdout to insecure locations unless you intend to store secrets there.

| Flag | Default | Behavior |
|---|---:|---|
| `--no-logging` | `false` | Disables operational logging. |
| `--log-level` | `info` | `error`, `warn`, `info`, `debug`. |
| `--log-file` | `""` | Empty means stdout for operational logs. |
| `--log-format` | `text` | `text`, `json`, `structured`. |
| `--log-max-size` | `10485760` | Rotation threshold in bytes. |
| `--log-max-files` | `5` | Number of rotated files to keep. |
| `--log-buffer-size` | `1000` | Async logging buffer size. |

## Environment variables

The application loads configuration from these environment variables before parsing CLI flags:

| Variable | Effect |
|---|---|
| `BLOCO_THREADS` | Worker thread count when positive integer. |
| `BLOCO_BATCH_SIZE` | Worker max batch size when positive integer. |
| `BLOCO_TUI` | Enables/disables TUI. |
| `BLOCO_COLOR` | TUI color support: `auto`, `enabled`, `disabled`. |
| `NO_COLOR` | Forces color support to `disabled`. |
| `BLOCO_VERBOSE` | Enables verbose output. |
| `BLOCO_QUIET` | Enables quiet mode. |
| `BLOCO_KEYSTORE_ENABLED` | Enables/disables backup artifact generation. |
| `BLOCO_KEYSTORE_DIR` | Backup artifact directory. |
| `BLOCO_KEYSTORE_KDF` | KDF algorithm. |
| `BLOCO_KDF_ANALYSIS` | Enables KDF analysis. |
| `BLOCO_SECURITY_LEVEL` | KDF security level. |
| `BLOCO_LOGGING_ENABLED` | Enables/disables operational logging. |
| `BLOCO_LOG_LEVEL` | Logging level. |
| `BLOCO_LOG_FORMAT` | Logging format. |
| `BLOCO_LOG_FILE` | Logging output file. |
| `BLOCO_BACKUP_STORE` | Backup store: `files` or `fnox`. |
| `BLOCO_BACKUP_DIR` | Directory for encrypted fnox backups. |
| `BLOCO_FNOX_BINARY` | Path to the fnox executable. |
| `BLOCO_AGE_RECIPIENTS` | Comma-separated age recipients for backups. |
| `BLOCO_AGE_IDENTITY_FILE` | Age identity file for backup decryption. |
| `BLOCO_DEBUG` | Enables additional debug prints/stack traces in several paths. |

## Stats command

```bash
./bloco-vgen stats --prefix dead --suffix beef --checksum --tui=false
```

Text-mode output includes:

- Pattern length.
- Whether checksum validation is enabled.
- Estimated difficulty.
- Attempts for 50% probability.
- Time estimates at fixed speeds: `1,000`, `10,000`, `50,000`, and `100,000` addresses/second.

## Benchmark command

```bash
./bloco-vgen benchmark --attempts 10000 --duration 30s --tui=false
```

Supported benchmark-specific flags:

| Flag | Default | Behavior |
|---|---:|---|
| `--attempts` | `10000` | Attempt limit for the benchmark command. |
| `--duration` | `30s` | Maximum benchmark duration. |
| `--detailed` | `false` | Prints detailed sample statistics when samples exist. |

## Current limitations

These are current code behavior, not intended long-term product claims:

- `benchmark` does not currently expose a `--pattern` flag, even though older documentation mentioned one.
- The text benchmark path currently does not submit real generation work to workers; it can report `0` attempts and `0 addr/s`.
- `--output` and `--format` are registered flags, but wallet generation currently prints text to stdout and does not write result files or switch to JSON/CSV output.
- Text-mode progress is limited; the previous text progress manager is disabled in generation fallback paths to avoid deadlocks.
- Prefix/suffix validation is hexadecimal for all networks, which limits Bitcoin and Solana vanity searches despite their Base58 address formats.
- Bitcoin mnemonics are generated as backup metadata and are not used to derive the generated private key.
- Solana backup currently writes a raw private key `.key` file when keystore output is enabled. Protect or disable this with `--no-keystore` if that is not acceptable.
- There is no database. Persistence is local filesystem only.

## Security notes

- Treat stdout as sensitive in the default `files` backup mode, because successful wallet output includes private keys and sometimes mnemonics. With `--backup-store fnox`, output is metadata-only and secrets are confined to the encrypted artifact.
- Treat `*.pwd`, `*.mnemonic`, and `*.key` files as sensitive secrets.
- Ethereum keystore filenames preserve the address case currently held by the generated wallet, including EIP-55 mixed case when checksum mode is used.
- The project currently targets Go `1.26.8` and `github.com/ethereum/go-ethereum v1.17.0` to avoid known `govulncheck` findings reported against older versions.

## License

No `LICENSE` file is currently present in this repository.
