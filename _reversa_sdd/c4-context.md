# C4 Context — Architect

> Escala: 🟢 CONFIRMADO | 🟡 INFERIDO | 🔴 LACUNA

## Diagrama

```mermaid
flowchart TB
  Operator["Pessoa: Operador CLI
Gera carteiras vanity, analisa dificuldade e executa benchmark"]
  Maintainer["Pessoa: Mantenedor
Executa CI/CD, releases e manutenção"]

  System["Sistema: bloco-vgen CLI
Gerador local de carteiras vanity multi-rede"]

  Terminal["Sistema externo: Terminal
stdout/stderr, TUI, sinais SIGINT/SIGTERM"]
  FS["Sistema externo: Filesystem local
keystores, senhas, mnemonics, logs"]
  Ethereum["Bibliotecas Ethereum
go-ethereum, EIP-55, KeyStore V3"]
  Bitcoin["Bibliotecas Bitcoin
btcd, btcec, btcutil"]
  Solana["Bibliotecas Solana
solana-go, Ed25519/base58"]
  GitHub["GitHub Actions
CI, lint, security scan, release"]
  GHCR["GitHub Container Registry
Imagem Docker"]
  Releases["GitHub Releases
Binários e checksums"]
  SecurityTools["Ferramentas de segurança
gosec, govulncheck, Semgrep, Trivy"]

  Operator -->|executa comandos e flags| System
  Operator <-->|visualiza resultados/progresso| Terminal
  System <-->|renderização e sinais| Terminal
  System -->|salva artefatos sensíveis| FS
  System -->|logs sanitizados| FS
  System -->|chamadas de biblioteca Go| Ethereum
  System -->|chamadas de biblioteca Go| Bitcoin
  System -->|chamadas de biblioteca Go| Solana

  Maintainer -->|push, PR, tags| GitHub
  GitHub -->|build/test/scan| System
  GitHub -->|publica imagem| GHCR
  GitHub -->|publica assets| Releases
  GitHub -->|executa scanners| SecurityTools
```

## Relacionamentos

| Origem | Destino | Relação | Protocolo/Formato | Confiança |
|---|---|---|---|---:|
| Operador CLI | `bloco-vgen` | Executa comandos | processo local + flags | 🟢 |
| `bloco-vgen` | Terminal | Exibe resultados/TUI | stdout/stderr/ANSI | 🟢 |
| Terminal | `bloco-vgen` | Envia sinais/cancelamento | SIGINT/SIGTERM, teclado TUI | 🟢 |
| `bloco-vgen` | Filesystem | Salva keystore, password, mnemonic e logs | JSON/texto | 🟢 |
| `bloco-vgen` | Bibliotecas blockchain | Gera chaves/endereço | chamadas Go locais | 🟢 |
| GitHub Actions | `bloco-vgen` | Testa/builda/scaneia | workflows YAML | 🟢 |
| GitHub Actions | GHCR/Releases | Publica artefatos | Docker/Release API | 🟢 |

## Observações

- Não há API externa obrigatória para gerar carteiras; integrações blockchain são bibliotecas locais.
- O principal limite de confiança é o filesystem local, onde ficam artefatos sensíveis.
- Logs legados devem ser tratados como possivelmente sensíveis.
