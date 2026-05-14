---
schemaVersion: 1
generatedAt: 2026-05-08T10:36:00Z
reversa:
  version: "1.2.34"
kind: discard_log
producedBy: curator
hash: "sha256:7b1552106139f59ee17af5b05176997a83f48c813a28049143e4c6f316563f03"
---

# Discard Log

> Registro completo do que foi descartado da migração e por quê. Cada item tem rastreabilidade para a origem no legado.

## Itens descartados

### BR-DESCARTAR-001 — Nome legado `bloco-vgen` como canônico

- **Origem**: `_reversa_sdd/questions.md` Pergunta 1; `_reversa_sdd/gaps.md` GAP-RV-001
- **Descrição**: Preservar `bloco-vgen` como nome canônico do produto/binário/documentação.
- **Justificativa**: Decisão humana consolidou `bloco-vanity-generator` para produto/repositório/documentação e `bloco-vgen` como binário compatível.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: nome canônico `bloco-vanity-generator`; compatibilidade operacional via `bloco-vgen`.
- **Risco de descartar**: baixo; mitigado por compatibilidade explícita de binário.

### BR-DESCARTAR-002 — Claims não implementados do README

- **Origem**: `_reversa_sdd/questions.md` Pergunta 2; `_reversa_sdd/gaps.md` GAP-RV-002
- **Descrição**: Tratar flags e exemplos do README não confirmados no Cobra como requisitos obrigatórios do alvo.
- **Justificativa**: O Revisor consolidou que README legado está desatualizado quando diverge do Cobra/código.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: documentação alvo deve refletir comportamento implementado e roadmap separado, se houver.
- **Risco de descartar**: baixo; evita inflar escopo com features não confirmadas.

### BR-DESCARTAR-003 — Validação hex global para prefixo/sufixo

- **Origem**: `_reversa_sdd/pkg/wallet/requirements.md`; `_reversa_sdd/questions.md` Pergunta 3
- **Descrição**: Manter regra global de que prefixo/sufixo devem ser hexadecimais para todas as redes.
- **Justificativa**: Decisão humana definiu validação específica por rede: Ethereum hexadecimal/EIP-55, Bitcoin Base58/bech32 quando aplicável e Solana Base58.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: validadores por rede.
- **Risco de descartar**: baixo; melhora correção multirede.

### BR-DESCARTAR-004 — `Wallet.IsValid()` Ethereum-only

- **Origem**: `_reversa_sdd/pkg/wallet/requirements.md`; `_reversa_sdd/questions.md` Pergunta 4
- **Descrição**: Preservar validação de carteira centrada em formatos Ethereum para todas as redes.
- **Justificativa**: Decisão humana definiu validação por `Network`.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: validação por rede em `Wallet` ou serviço de validação.
- **Risco de descartar**: baixo; corrige bug confirmado de multirede.

### BR-DESCARTAR-005 — Persistência Solana em `.key` bruto

- **Origem**: `_reversa_sdd/questions.md` Pergunta 5; `_reversa_sdd/database/data-dictionary.md`; `_reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md`
- **Descrição**: Salvar private key Solana bruta em arquivo `.key`.
- **Justificativa**: Decisão humana definiu persistência Solana criptografada/segura e sem `.key` bruto.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: formato criptografado/seguro definido pelo Designer e validado pelo Inspector.
- **Risco de descartar**: baixo; remove risco de segredo em repouso.

### BR-DESCARTAR-006 — `WalletLogger` com private key em claro

- **Origem**: `_reversa_sdd/questions.md` Pergunta 6; `_reversa_sdd/pkg/wallet/requirements.md`; `_reversa_sdd/pkg/logging/requirements.md`
- **Descrição**: Preservar logger legado que escreve private key em claro em `wallets-YYYYMMDD.log`.
- **Justificativa**: Decisão humana definiu migração para logging seguro/sanitizado.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: logging seguro com whitelist/redaction e testes negativos de segredo.
- **Risco de descartar**: baixo; comportamento legado é risco de segurança.

### BR-DESCARTAR-007 — Progress textual permanentemente desabilitado por deadlock

- **Origem**: `_reversa_sdd/questions.md` Pergunta 7; `_reversa_sdd/internal/progress/requirements.md`
- **Descrição**: Manter progress manager textual desabilitado por risco de deadlock.
- **Justificativa**: Decisão humana definiu correção e reativação como fallback textual.
- **Vinculado a paradigma**: não
- **Reposição no sistema novo**: progress textual redesenhado com lifecycle/cancelamento testáveis.
- **Risco de descartar**: médio; requer implementação concorrente cuidadosa.

### BR-DESCARTAR-008 — `EncryptPrivateKeyWithKDF()` como contrato público

- **Origem**: `_reversa_sdd/questions.md` Pergunta 8; `_reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md`; `_reversa_sdd/migration/paradigm_decision.md`
- **Descrição**: Expor ou preservar `EncryptPrivateKeyWithKDF()` como API pública/contrato de consumidores.
- **Justificativa**: Decisão humana definiu a função como detalhe interno; consumidores devem usar `GenerateKeyStore()`/serviço de alto nível.
- **Vinculado a paradigma**: sim
  - **Como o paradigma alvo absorve o caso**: no desenho Go idiomático transformacional, detalhes de KDF/cipher ficam encapsulados atrás de serviço de alto nível, reduzindo acoplamento e uso incorreto.
- **Reposição no sistema novo**: serviço `GenerateKeyStore` ou equivalente público seguro.
- **Risco de descartar**: baixo; reduz superfície de API insegura.

### BR-DESCARTAR-009 — Cópia rígida da organização interna do legado

- **Origem**: `_reversa_sdd/migration/paradigm_decision.md`; `_reversa_sdd/architecture.md` § Dívidas técnicas e riscos arquiteturais
- **Descrição**: Preservar a estrutura interna do legado como contrato, incluindo acoplamento alto em `internal/cli` e separações históricas de pacotes.
- **Justificativa**: A decisão transformacional autoriza reorganização interna Go mais idiomática, preservando paridade comportamental em vez de topologia interna.
- **Vinculado a paradigma**: sim
  - **Como o paradigma alvo absorve o caso**: Go idiomático usa pacotes por coesão, interfaces pequenas em bordas e funções/serviços testáveis; a estrutura antiga vira referência histórica, não requisito.
- **Reposição no sistema novo**: arquitetura alvo a ser definida pelo Designer com CLI, domínio, crypto, validação, persistência, logging e TUI desacoplados.
- **Risco de descartar**: médio; exige boa matriz de paridade para evitar perda de comportamento.

## Itens descartados por mudança de paradigma

> Lista apenas dos itens cujo `Vinculado a paradigma = sim`. Auditoria explícita para o agente de codificação.

| ID | Origem | Paradigma legado | Substituto no paradigma alvo |
|---|---|---|---|
| BR-DESCARTAR-008 | `_reversa_sdd/questions.md` Pergunta 8 | Função interna exposta/acessível como detalhe operacional | Serviço Go de alto nível com KDF/cipher encapsulados |
| BR-DESCARTAR-009 | `_reversa_sdd/migration/paradigm_decision.md` | Organização histórica de pacotes tratada como contrato | Pacotes Go coesos por domínio/fronteira, com paridade comportamental |

## Notas

Nenhum item de negócio central foi descartado por mudança de paradigma. Os descartes são dívidas, comportamentos inseguros, nomes legados ou estruturas internas acidentais. A paridade deve ser validada pelo Inspector com foco em comportamento externo, segurança e artefatos gerados.
