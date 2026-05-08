# Lacunas Pendentes — bloco-vanity-generator

> Atualizado pelo Revisor em 2026-05-08T09:43:52Z
> Doc level: completo

## Resumo

- **Perguntas pendentes:** 0
- **Perguntas respondidas:** 8 🟢
- **Revisão cruzada externa:** não realizada nesta sessão
- **Status:** lacunas do Revisor resolvidas por decisão humana
- **Perguntas unitárias do Writer:** 35 em 10 arquivos, triadas pelo Revisor; 0 críticas/vermelhas.

## Decisões consolidadas

### GAP-RV-001 — Resolvido

- **Resposta:** bloco-vanity-generator para produto/repositório/documentação; bloco-vgen como binário compatível.
- **Decisão:** A nomenclatura-alvo deve normalizar produto, repo e docs como `bloco-vanity-generator`, mantendo `bloco-vgen` como binário compatível. 🟢
- **Status:** resolvido

### GAP-RV-002 — Resolvido

- **Resposta:** README informa features que não foram implementadas e está desatualizado.
- **Decisão:** Claims do README que não aparecem no Cobra devem ser tratados como documentação desatualizada, não como comportamento legado confirmado. 🟢
- **Status:** resolvido

### GAP-RV-003 — Resolvido

- **Resposta:** Validar por rede, ex. Base58/bech32 quando aplicável.
- **Decisão:** A validação-alvo de padrões vanity deve ser específica por rede: Ethereum hexadecimal/EIP-55; Bitcoin Base58/bech32 conforme tipo; Solana Base58. 🟢
- **Status:** resolvido

### GAP-RV-004 — Resolvido

- **Resposta:** Validar `Wallet.IsValid()` por `Network`.
- **Decisão:** `Wallet.IsValid()` deve evoluir para validação por rede, não permanecer centrado apenas em Ethereum. 🟢
- **Status:** resolvido

### GAP-RV-005 — Resolvido

- **Resposta:** Usar formato criptografado para persistência Solana.
- **Decisão:** Persistência Solana deve evitar `.key` bruto e usar formato criptografado/seguro. 🟢
- **Status:** resolvido

### GAP-RV-006 — Resolvido

- **Resposta:** Migrar `WalletLogger` legado para logging seguro.
- **Decisão:** `WalletLogger` não deve preservar escrita de private key em claro; comportamento deve migrar para logging seguro/sanitizado. 🟢
- **Status:** resolvido

### GAP-RV-007 — Resolvido

- **Resposta:** Corrigir e reativar progress manager textual.
- **Decisão:** O progress manager textual deve ser corrigido contra deadlocks e reativado como fallback textual. 🟢
- **Status:** resolvido

### GAP-RV-008 — Resolvido

- **Resposta:** `EncryptPrivateKeyWithKDF()` é detalhe interno.
- **Decisão:** `EncryptPrivateKeyWithKDF()` não deve ser contrato público; consumidores devem usar `GenerateKeyStore()`/serviço de alto nível. 🟢
- **Status:** resolvido
