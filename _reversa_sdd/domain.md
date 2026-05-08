# Domain — Detective

> Projeto: `bloco-wallet-generator`  
> Fase: Interpretação / Detective  
> Escala: 🟢 CONFIRMADO | 🟡 INFERIDO | 🔴 LACUNA

## Resumo do domínio

O sistema é um gerador CLI de carteiras vanity para redes blockchain. O domínio central é a busca computacional por endereços que satisfaçam padrões informados pelo usuário (`prefix`, `suffix` e, no caso Ethereum, checksum EIP-55). O valor de negócio é equilibrar **customização de endereço**, **segurança de material criptográfico** e **desempenho de busca concorrente**.

A aplicação evoluiu de um gerador Ethereum para um gerador multi-rede com suporte a Ethereum, Bitcoin e Solana, mantendo Ethereum como fluxo mais completo: EIP-55, KeyStore V3, KDF universal, compatibilidade com clientes Ethereum e senha segura.

## Glossário

| Termo | Definição | Confiança | Evidência |
|---|---|---:|---|
| Vanity address | Endereço blockchain que contém prefixo/sufixo escolhido pelo usuário. | 🟢 | README e `GenerationCriteria` |
| Pattern | Concatenação lógica de `prefix + suffix` usada para dificuldade e exibição. | 🟢 | `pkg/wallet/types.go`, `pkg/utils/format.go` |
| Prefix | Sequência hexadecimal desejada no início do endereço. | 🟢 | flags CLI e validação |
| Suffix | Sequência hexadecimal desejada no final do endereço. | 🟢 | flags CLI e validação |
| Checksum EIP-55 | Convenção Ethereum de caixa alta/baixa baseada em Keccak para detectar erros de endereço. | 🟢 | `internal/crypto/checksum.go` |
| Case-sensitive pattern | Padrão que considera maiúsculas/minúsculas quando checksum está habilitado. | 🟢 | commits `390898a`, `d0fddb0`, README |
| Difficulty | Espaço esperado de busca calculado por comprimento do padrão e impacto de checksum. | 🟢 | `pkg/utils.CalculateDifficulty` |
| Probability50 | Tentativas necessárias para 50% de chance de encontrar match. | 🟢 | `CalculateProbability50` |
| Worker | Goroutine que tenta gerar endereços até encontrar um match ou receber cancelamento. | 🟢 | `internal/worker/pool.go` |
| Worker pool | Conjunto de workers concorrentes com métricas agregadas. | 🟢 | `internal/worker` |
| KeyStore V3 | Formato JSON Ethereum para chave privada cifrada com KDF, AES-CTR e MAC. | 🟢 | `internal/crypto/keystore.go`, README |
| KDF | Função de derivação de chave: scrypt/PBKDF2. | 🟢 | `internal/crypto/kdf` |
| Security level | Perfil de segurança usado para otimizar parâmetros KDF. | 🟢 | config/CLI/KDF analyzer |
| Mnemonic | Frase BIP-39 usada para gerar/recuperar carteira. | 🟢 | commits `b1e416c`, `6c80aa2` |
| Secure logging | Logging operacional sem chaves privadas, públicas, mnemonic ou material criptográfico. | 🟢 | README Security Notice, commit `ece8dcf`, `pkg/logging` |
| Artifact/release | Binário, imagem Docker ou checksum produzido por CI/release. | 🟢 | `.github/workflows` |

## Regras de domínio

### Entrada e critérios de geração

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-001 | O usuário pode buscar por prefixo, sufixo ou ambos. | 🟢 | README, flags `--prefix`, `--suffix` |
| BR-002 | No legado, prefixo e sufixo são validados como hexadecimais por `GenerationCriteria.Validate()`. Para evolução/reimplementação, a decisão humana é validar por rede: Ethereum hexadecimal/EIP-55, Bitcoin Base58/bech32 quando aplicável e Solana Base58. | 🟢 | `GenerationCriteria.Validate()` + resposta do Revisor |
| BR-003 | O tamanho total do padrão (`prefix + suffix`) é limitado a 20 caracteres. | 🟢 | `GenerationCriteria.Validate()` |
| BR-004 | `MaxAttempts` não pode ser negativo. | 🟢 | `GenerationCriteria.Validate()` |
| BR-005 | O modo checksum torna o padrão sensível a maiúsculas/minúsculas para Ethereum. | 🟢 | README, commits `390898a`, `d0fddb0` |
| BR-006 | Padrões longos devem ser usados com cautela por custo computacional alto. | 🟢 | README exemplos: 4 chars max recomendado/caution |

### Dificuldade, probabilidade e progresso

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-007 | Dificuldade base é `16^len(prefix+suffix)`. | 🟢 | `pkg/utils.CalculateDifficulty()` |
| BR-008 | Com checksum, cada letra hexadecimal (`a-f`/`A-F`) no padrão multiplica a dificuldade por 2. | 🟢 | `CalculateDifficulty()` |
| BR-009 | Probabilidade acumulada é calculada como `1 - (1 - 1/difficulty)^attempts`. | 🟢 | `CalculateProbability()` |
| BR-010 | O ETA deriva de `Probability50`, tentativas restantes e velocidade atual/agregada. | 🟢 | `GenerationStats.Update()`, TUI/progress |
| BR-011 | O progresso TUI para múltiplas carteiras é medido por carteiras concluídas, não apenas probabilidade de uma busca. | 🟢 | `generateMultipleWalletsTUI()` |

### Redes e formatos

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-012 | Redes suportadas: Ethereum, Bitcoin e Solana. | 🟢 | `internal/crypto/{ethereum,bitcoin,solana}.go`, commit `aceb60e` |
| BR-013 | Ethereum usa secp256k1 e endereço derivado por Keccak da chave pública. | 🟢 | `internal/crypto/ethereum.go` |
| BR-014 | Bitcoin gera endereço P2PKH com chave pública comprimida. | 🟢 | `internal/crypto/bitcoin.go` |
| BR-015 | Solana usa Ed25519 e endereço base58 da chave pública. | 🟢 | `internal/crypto/solana.go` |
| BR-016 | Mnemonic no worker é suportado apenas para Ethereum; para outras redes é desabilitado. | 🟢 | `internal/worker/pool.go` |
| BR-017 | Bitcoin persiste apenas mnemonic para backup, não KeyStore V3. | 🟢 | `internal/cli/commands.go` |
| BR-018 | No legado, Solana tem persistência simplificada em área de keystore. Para evolução/reimplementação, a decisão humana é usar persistência Solana criptografada/segura, sem `.key` bruto. | 🟢 | `internal/crypto/keystore.go` + resposta do Revisor |

### Segurança de chaves e keystore

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-019 | KeyStore é gerado por padrão, salvo em diretório configurável. | 🟢 | README e flags `--no-keystore`, `--keystore-dir` |
| BR-020 | KeyStore V3 usa cipher `aes-128-ctr` e versão `3`. | 🟢 | `KeyStoreV3.Validate()` |
| BR-021 | KDF permitido no keystore: scrypt, pbkdf2, pbkdf2-sha256, pbkdf2-sha512. | 🟢 | config/KDF service |
| BR-022 | scrypt exige `N` potência de 2, range 1024..67108864, e valida memória máxima de 2GB. | 🟢 | `internal/crypto/kdf/scrypt.go` |
| BR-023 | PBKDF2 exige ao menos 1000 iterações e recomenda 100000+. | 🟢 | `internal/crypto/kdf/pbkdf2.go` |
| BR-024 | Senhas de keystore devem ter pelo menos 12 caracteres e cobrir minúscula, maiúscula, número e especial. | 🟢 | `PasswordGenerator` |
| BR-025 | MAC do keystore usa Keccak sobre `derivedKey[16:32] + ciphertext`. | 🟢 | `GenerateMAC()` |
| BR-026 | Salt/KDF e dados sensíveis não devem ser registrados em logs. | 🟢 | `SecureKDFLogger`, `pkg/logging`, README |

### Logging e observabilidade

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-027 | Logs operacionais podem conter endereço, tentativas, duração, thread e status. | 🟢 | `pkg/logging` whitelist |
| BR-028 | Logs não devem conter private key, public key, mnemonic ou material criptográfico. | 🟢 | README Security Notice, redaction tests |
| BR-029 | Logging pode ser desabilitado por flag/configuração. | 🟢 | `--no-logging`, `NewSecureLoggerFromConfig()` |
| BR-030 | Em contexto TUI, fallback de logging evita interferência com stdout. | 🟢 | `shouldAvoidStdout()` e inicialização de writer |

### CLI, TUI e execução

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-031 | `--threads=0` significa autodetectar CPUs; internamente o pool força mínimo 1. | 🟢 | README, `parseFlags()`, `NewPoolWithConfig()` |
| BR-032 | `quiet` e `verbose` são mutuamente exclusivos. | 🟢 | `Config.Validate()` |
| BR-033 | TUI só é usada quando habilitada, `--progress` está ativo e `quiet` está desligado. | 🟢 | `generateSingleWallet()`/`generateMultipleWallets()` |
| BR-034 | Se TUI falhar, a CLI cai para modo texto. | 🟢 | `program.Run()` fallback |
| BR-035 | Em modo texto, progress manager está desabilitado por risco de deadlock. | 🟢 | comentários em `internal/cli/commands.go` |
| BR-036 | O processo trata SIGINT/SIGTERM com cancelamento gracioso. | 🟢 | `setupGracefulShutdown()` |

### CI/CD e governança técnica

| ID | Regra | Confiança | Evidência |
|---|---|---:|---|
| BR-037 | CI executa testes curtos com race detector e timeout de 90s. | 🟢 | `.github/workflows/ci.yaml` |
| BR-038 | CI valida `go vet`, `gofmt`, lint, gosec e govulncheck. | 🟢 | CI workflow |
| BR-039 | Builds oficiais cobrem Linux e macOS, amd64 e arm64. | 🟢 | CI/release workflows |
| BR-040 | Release por tag `v*.*.*` publica binários, checksums e imagem Docker. | 🟢 | release workflow |
| BR-041 | Workflows têm permissões explícitas para conteúdo, pacotes e security-events conforme necessidade. | 🟢 | workflow permissions, commit `c1856bf` |

## Eventos de negócio e logs

| Evento | Descrição | Fonte | Confiança |
|---|---|---|---:|
| `wallet_generated` | Carteira encontrada com endereço, tentativas, duração e worker. | `LogWalletGenerated()` | 🟢 |
| `operation_start` / `operation_complete` | Início/fim de operações como KDF ou geração. | `LogOperationStart/Complete()` | 🟢 |
| `kdf_derive` | Tentativa, sucesso e erro de derivação de chave. | `SecureKDFLogger` | 🟢 |
| `worker_startup` | Inicialização de worker com config sanitizada. | `LogWorkerStartup()` | 🟢 |
| `performance_metrics` | Métricas de performance e escalabilidade. | `LogPerformanceMetrics()` | 🟢 |

Foram encontrados arquivos `wallets-*.log` em `internal/worker/`. A análise foi deliberadamente limitada a metadados, porque os próprios nomes/conteúdo indicam presença de termos de private key e o domínio impõe higiene de segredos.

## Lacunas e ambiguidades

| ID | Lacuna | Impacto | Confiança |
|---|---|---|---:|
| GAP-001 | O produto se chama `Bloco Vanity Generator`, mas o módulo/binário/caminhos ainda misturam `bloco-eth`, `bloco-wallet-generator` e `bloco-vanity-generator`. | Pode confundir instalação, release, Docker e imports. | 🟢 |
| GAP-002 | README descreve flags `--optimize-for` e benchmark `--pattern`, mas a leitura de `commands.go` não confirmou essas flags no trecho analisado. | Documentação pode anunciar comportamento inexistente/incompleto. | 🟡 |
| GAP-003 | `EncryptPrivateKeyWithKDF()` usa endereço Ethereum placeholder ao montar KeyStore. | Pode gerar keystore com endereço incorreto em fluxo específico. | 🟢 |
| GAP-004 | Persistência Solana contém placeholder/simplificação para keypair. | Risco de backup/import incompleto para Solana. | 🟢 |
| GAP-005 | `Wallet.IsValid()` assume endereço de 40 caracteres e private key de 64, o que conflita com Bitcoin/Solana e endereços Ethereum com `0x`. | Validação de sucesso pode falhar fora do caso Ethereum sem prefixo. | 🟢 |
| GAP-006 | Progress manager texto está desabilitado por deadlocks, sem resolução documentada. | Funcionalidade de progresso textual parcial. | 🟢 |
| GAP-007 | Benchmark cria `WorkItem`, mas comentários indicam TODO para integração real com ants pool. | Métricas de benchmark podem não exercitar geração real como esperado. | 🟢 |
| GAP-008 | Não há RBAC de usuário; permissões existem só no contexto de GitHub Actions/artefatos. | Sem matriz de permissões de aplicativo além de CLI/CI. | 🟢 |

## Decisões inferidas pelo Detective

1. Priorizar segurança de segredos após versões anteriores possivelmente registrarem chaves privadas em logs.
2. Adotar KeyStore V3 e KDF universal para compatibilidade com ecossistema Ethereum.
3. Tratar performance como requisito central: multithreading, object pooling, hot path otimizado.
4. Expandir de Ethereum para multi-rede, mas manter Ethereum como domínio mais maduro.
5. Evitar progresso textual quando há risco de deadlock, preferindo TUI ou modo texto simples.
