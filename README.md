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

- Go `1.25.10` or newer.
- Git, if building from a clone.

CI and Docker currently use Go `1.25.10`.

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
| `--network` | | `ethereum` | Target network: `ethereum`, `bitcoin` or `solana`. Any other value is rejected before generation starts. |
| `--with-mnemonic` | | `false` | Derives the key from a BIP-39 mnemonic so the saved phrase restores the wallet. Supported for Ethereum and Bitcoin; ignored for Solana (Ed25519). |
| `--threads` | `-t` | `0` | `0` means auto-detect CPU count; positive values set worker count. Config validation rejects more than 128 threads. |
| `--progress` | | `false` | Enables TUI progress when available. Text-mode live progress is limited. |
| `--tui` | | `true` | Enables terminal UI when supported. Use `--tui=false` for plain text output. |
| `--verbose` | `-v` | `false` | Enables verbose output in supported paths. |
| `--quiet` | `-q` | `false` | Suppresses some non-essential output. It is not a secret-redaction guarantee for every output path. |
| `--output` | | `""` | Writes the results to this file with mode `0600` instead of printing the private key to the terminal. |
| `--format` | | `text` | Output format: `text`, `json` or `csv`. Applies to `--output`, and to stdout for `json`/`csv`. |

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

Backup files are enabled by default and written to `./keystores`. Use `--no-keystore` to skip them.

The output directory is created with `0700` and every file inside it with `0600`, through atomic
temporary-file writes.

| Network | Files written |
|---|---|
| Ethereum | `UTC--<timestamp>--0x<address>.json`, an encrypted KeyStore V3 file. With `--with-mnemonic`, also `0x<address>.mnemonic`, which derives the key at `m/44'/60'/0'/0/0`. EIP-55 case is preserved in filenames. |
| Bitcoin | `<address>.json`, an encrypted KeyStore V3 file. With `--with-mnemonic`, also `<address>.mnemonic`, which derives the key at `m/44'/0'/0'/0/0`. |
| Solana | `<address>.json`, an encrypted KeyStore V3 file holding the 64-byte Ed25519 key. |

**The keystore password is not written to disk by default.** It is printed once when the run
finishes, or included in the `--output` file when one is given. Storing it next to the keystore it
unlocks would reduce the KDF to nothing for anyone who copies the directory, so it is opt-in:

| Flag | Default | Behavior |
|---|---:|---|
| `--write-password-file` | `false` | Also writes `<address>.pwd` next to the keystore. Insecure: the password and the file it opens end up in the same directory. |
| `--write-plaintext-key` | `false` | Also writes `<address>.key` with the raw private key hex (Solana). Insecure: unencrypted key material on disk. |

### KeyStore settings

| Flag | Default | Valid values / behavior |
|---|---:|---|
| `--keystore-dir` | `./keystores` | Output directory, created with `0700`. |
| `--keystore-kdf` | `scrypt` | `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512`. |
| `--kdf-params` | auto | JSON parameters. Applied to the generated keystore and held to the security floor below. |
| `--security-level` | `medium` | `low`, `medium`, `high`, `very-high`. Used when `--kdf-params` is not supplied. |
| `--kdf-analysis` | `false` | Prints KDF compatibility/security analysis after keystore generation. |

Minimum accepted KDF strength, enforced both when the flag is parsed and again before the key is
encrypted: scrypt needs `n >= 16384` and `128 * n * r >= 16 MiB`; PBKDF2 needs `c >= 100000`. These
match the `low` security level, so every documented `--security-level` value is accepted.

Example custom KDF parameters:

```bash
./bloco-vgen --prefix abc \
  --keystore-kdf scrypt \
  --kdf-params '{"n":262144,"r":8,"p":1,"dklen":32}' \
  --tui=false
```

## Logging

Operational logging is enabled by default. The secure logger is designed to avoid logging private keys, public keys, seeds, mnemonics, and keystore passwords.

Important: wallet result output on stdout does print private keys for successful generation paths. Do not run the CLI in shared terminals or redirect stdout to insecure locations unless you intend to store secrets there.

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
- Text-mode progress is limited; the previous text progress manager is disabled in generation fallback paths to avoid deadlocks.
- Prefix/suffix validation is hexadecimal for all networks, which limits Bitcoin and Solana vanity searches despite their Base58 address formats. A Bitcoin P2PKH address always starts with `1`, so a prefix that cannot occur will search forever.
- `--with-mnemonic` derives the key from the phrase through BIP-32, which costs roughly 14 ms per candidate. It is practical for short patterns only.
- There is no database. Persistence is local filesystem only.

## Security notes

- Treat stdout as sensitive because successful wallet output includes private keys, keystore passwords and sometimes mnemonics. Use `--output <file>` to write them to a `0600` file instead: the private key is then not printed to the terminal.
- Treat `*.mnemonic` files, and the opt-in `*.pwd` and `*.key` files, as sensitive secrets.
- Every wallet returned in a single run has an independent private key. The generator scans candidates in chained batches for speed, and the chain that produced a delivered key is discarded so that no two delivered keys are a small offset apart.
- Ethereum keystore filenames preserve the address case currently held by the generated wallet, including EIP-55 mixed case when checksum mode is used.
- The project currently targets Go `1.25.9+` and `github.com/ethereum/go-ethereum v1.17.0` to avoid known `govulncheck` findings reported against older versions.

## License

No `LICENSE` file is currently present in this repository.
