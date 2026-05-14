# Relatório de Confiança — bloco-vanity-generator

> Atualizado pelo Revisor em 2026-05-08T09:43:52Z
> Status: final da Fase 5 após respostas humanas.

---

## Resumo Geral

| Nível | Quantidade | Percentual |
|-------|-----------:|-----------:|
| 🟢 CONFIRMADO | 3054 | 79.95% |
| 🟡 INFERIDO | 564 | 14.76% |
| 🔴 LACUNA | 202 | 5.29% |
| **Total** | 3820 | 100% |

**Confiança geral:** 87.33% (soma de 🟢 + metade dos 🟡)

---

## Revisão Cruzada

- **Engine externa consultada:** não aplicável nesta sessão
- **Apontamentos recebidos:** 0
- **Aceitos:** 0 | **Rejeitados:** 0 | **Pendentes:** 0

---

## Respostas Humanas Consolidadas

### Pergunta 1
- **Resposta:** bloco-vanity-generator para produto/repositório/documentação; bloco-vgen como binário compatível.
- **Decisão aplicada:** A nomenclatura-alvo deve normalizar produto, repo e docs como `bloco-vanity-generator`, mantendo `bloco-vgen` como binário compatível. 🟢

### Pergunta 2
- **Resposta:** README informa features que não foram implementadas e está desatualizado.
- **Decisão aplicada:** Claims do README que não aparecem no Cobra devem ser tratados como documentação desatualizada, não como comportamento legado confirmado. 🟢

### Pergunta 3
- **Resposta:** Validar por rede, ex. Base58/bech32 quando aplicável.
- **Decisão aplicada:** A validação-alvo de padrões vanity deve ser específica por rede: Ethereum hexadecimal/EIP-55; Bitcoin Base58/bech32 conforme tipo; Solana Base58. 🟢

### Pergunta 4
- **Resposta:** Validar `Wallet.IsValid()` por `Network`.
- **Decisão aplicada:** `Wallet.IsValid()` deve evoluir para validação por rede, não permanecer centrado apenas em Ethereum. 🟢

### Pergunta 5
- **Resposta:** Usar formato criptografado para persistência Solana.
- **Decisão aplicada:** Persistência Solana deve evitar `.key` bruto e usar formato criptografado/seguro. 🟢

### Pergunta 6
- **Resposta:** Migrar `WalletLogger` legado para logging seguro.
- **Decisão aplicada:** `WalletLogger` não deve preservar escrita de private key em claro; comportamento deve migrar para logging seguro/sanitizado. 🟢

### Pergunta 7
- **Resposta:** Corrigir e reativar progress manager textual.
- **Decisão aplicada:** O progress manager textual deve ser corrigido contra deadlocks e reativado como fallback textual. 🟢

### Pergunta 8
- **Resposta:** `EncryptPrivateKeyWithKDF()` é detalhe interno.
- **Decisão aplicada:** `EncryptPrivateKeyWithKDF()` não deve ser contrato público; consumidores devem usar `GenerateKeyStore()`/serviço de alto nível. 🟢

---

## Por Spec

| Spec | 🟢 | 🟡 | 🔴 | Confiança |
|------|---:|---:|---:|----------:|
| `_reversa_sdd/adrs/0001-adotar-logging-seguro.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0002-usar-keystore-v3-com-kdf-universal.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0003-priorizar-performance-com-workers-e-object-pooling.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0004-expandir-de-ethereum-para-multirede.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0005-suportar-mnemonic-bip39-como-opcao-de-geracao.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0006-validar-checksum-e-case-sensitivity-por-rede.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/adrs/0007-usar-ci-com-testes-curtos-e-scans-de-seguranca.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/architecture.md` | 58 | 2 | 1 | 96.72% |
| `_reversa_sdd/c4-components.md` | 37 | 0 | 0 | 100.00% |
| `_reversa_sdd/c4-containers.md` | 14 | 0 | 0 | 100.00% |
| `_reversa_sdd/c4-context.md` | 8 | 1 | 1 | 85.00% |
| `_reversa_sdd/cmd/bloco-vgen/design.md` | 46 | 3 | 2 | 93.14% |
| `_reversa_sdd/cmd/bloco-vgen/inicializacao-da-cli/design.md` | 49 | 8 | 2 | 89.83% |
| `_reversa_sdd/cmd/bloco-vgen/inicializacao-da-cli/requirements.md` | 38 | 3 | 1 | 94.05% |
| `_reversa_sdd/cmd/bloco-vgen/inicializacao-da-cli/tasks.md` | 31 | 5 | 3 | 85.90% |
| `_reversa_sdd/cmd/bloco-vgen/requirements.md` | 35 | 4 | 1 | 92.50% |
| `_reversa_sdd/cmd/bloco-vgen/tasks.md` | 34 | 3 | 3 | 88.75% |
| `_reversa_sdd/code-analysis.md` | 19 | 0 | 0 | 100.00% |
| `_reversa_sdd/data-dictionary.md` | 1 | 0 | 0 | 100.00% |
| `_reversa_sdd/dependencies.md` | 8 | 1 | 1 | 85.00% |
| `_reversa_sdd/domain.md` | 70 | 2 | 1 | 97.26% |
| `_reversa_sdd/erd-complete.md` | 20 | 1 | 0 | 97.62% |
| `_reversa_sdd/gaps.md` | 9 | 0 | 0 | 100.00% |
| `_reversa_sdd/internal/cli/contracts.md` | 75 | 8 | 1 | 94.05% |
| `_reversa_sdd/internal/cli/design.md` | 108 | 4 | 1 | 97.35% |
| `_reversa_sdd/internal/cli/flows.md` | 63 | 4 | 1 | 95.59% |
| `_reversa_sdd/internal/cli/gerar-carteiras-vanity/design.md` | 90 | 4 | 1 | 96.84% |
| `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md` | 65 | 4 | 1 | 95.71% |
| `_reversa_sdd/internal/cli/gerar-carteiras-vanity/tasks.md` | 47 | 8 | 2 | 89.47% |
| `_reversa_sdd/internal/cli/questions.md` | 16 | 12 | 1 | 75.86% |
| `_reversa_sdd/internal/cli/requirements.md` | 72 | 6 | 1 | 94.94% |
| `_reversa_sdd/internal/cli/salvar-keystore/design.md` | 80 | 15 | 1 | 91.15% |
| `_reversa_sdd/internal/cli/salvar-keystore/requirements.md` | 66 | 6 | 1 | 94.52% |
| `_reversa_sdd/internal/cli/salvar-keystore/tasks.md` | 48 | 5 | 2 | 91.82% |
| `_reversa_sdd/internal/cli/tasks.md` | 51 | 9 | 3 | 88.10% |
| `_reversa_sdd/internal/config/carregar-configuracao/design.md` | 26 | 6 | 1 | 87.88% |
| `_reversa_sdd/internal/config/carregar-configuracao/requirements.md` | 22 | 5 | 2 | 84.48% |
| `_reversa_sdd/internal/config/carregar-configuracao/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/config/contracts.md` | 9 | 3 | 1 | 80.77% |
| `_reversa_sdd/internal/config/design.md` | 26 | 6 | 1 | 87.88% |
| `_reversa_sdd/internal/config/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/internal/config/requirements.md` | 22 | 5 | 2 | 84.48% |
| `_reversa_sdd/internal/config/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/crypto/contracts.md` | 12 | 3 | 1 | 84.38% |
| `_reversa_sdd/internal/crypto/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/flows.md` | 8 | 2 | 1 | 81.82% |
| `_reversa_sdd/internal/crypto/gerar-carteira-multirede/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/gerar-carteira-multirede/requirements.md` | 28 | 5 | 2 | 87.14% |
| `_reversa_sdd/internal/crypto/gerar-carteira-multirede/tasks.md` | 17 | 6 | 3 | 76.92% |
| `_reversa_sdd/internal/crypto/gerar-keystore-v3/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md` | 29 | 5 | 2 | 87.50% |
| `_reversa_sdd/internal/crypto/gerar-keystore-v3/tasks.md` | 17 | 6 | 3 | 76.92% |
| `_reversa_sdd/internal/crypto/gerar-senha-segura/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/gerar-senha-segura/requirements.md` | 27 | 5 | 2 | 86.76% |
| `_reversa_sdd/internal/crypto/gerar-senha-segura/tasks.md` | 17 | 6 | 3 | 76.92% |
| `_reversa_sdd/internal/crypto/kdf/analisar-compatibilidade/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/kdf/analisar-compatibilidade/requirements.md` | 24 | 5 | 2 | 85.48% |
| `_reversa_sdd/internal/crypto/kdf/analisar-compatibilidade/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/crypto/kdf/contracts.md` | 9 | 3 | 1 | 80.77% |
| `_reversa_sdd/internal/crypto/kdf/derivar-chave/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/kdf/derivar-chave/requirements.md` | 24 | 5 | 2 | 85.48% |
| `_reversa_sdd/internal/crypto/kdf/derivar-chave/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/crypto/kdf/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/kdf/flows.md` | 7 | 2 | 1 | 80.00% |
| `_reversa_sdd/internal/crypto/kdf/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/internal/crypto/kdf/requirements.md` | 24 | 5 | 2 | 85.48% |
| `_reversa_sdd/internal/crypto/kdf/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/crypto/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/internal/crypto/requirements.md` | 27 | 5 | 2 | 86.76% |
| `_reversa_sdd/internal/crypto/tasks.md` | 17 | 6 | 3 | 76.92% |
| `_reversa_sdd/internal/crypto/validar-checksum-eip55/design.md` | 31 | 6 | 1 | 89.47% |
| `_reversa_sdd/internal/crypto/validar-checksum-eip55/requirements.md` | 27 | 5 | 2 | 86.76% |
| `_reversa_sdd/internal/crypto/validar-checksum-eip55/tasks.md` | 17 | 6 | 3 | 76.92% |
| `_reversa_sdd/internal/progress/design.md` | 22 | 5 | 1 | 87.50% |
| `_reversa_sdd/internal/progress/questions.md` | 3 | 3 | 2 | 56.25% |
| `_reversa_sdd/internal/progress/requirements.md` | 23 | 5 | 2 | 85.00% |
| `_reversa_sdd/internal/progress/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/tui/contracts.md` | 9 | 3 | 1 | 80.77% |
| `_reversa_sdd/internal/tui/design.md` | 24 | 5 | 1 | 88.33% |
| `_reversa_sdd/internal/tui/exibir-stats-benchmark/design.md` | 24 | 5 | 1 | 88.33% |
| `_reversa_sdd/internal/tui/exibir-stats-benchmark/requirements.md` | 23 | 5 | 2 | 85.00% |
| `_reversa_sdd/internal/tui/exibir-stats-benchmark/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/tui/flows.md` | 6 | 2 | 1 | 77.78% |
| `_reversa_sdd/internal/tui/questions.md` | 3 | 3 | 2 | 56.25% |
| `_reversa_sdd/internal/tui/renderizar-progresso/design.md` | 24 | 5 | 1 | 88.33% |
| `_reversa_sdd/internal/tui/renderizar-progresso/requirements.md` | 23 | 5 | 2 | 85.00% |
| `_reversa_sdd/internal/tui/renderizar-progresso/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/tui/requirements.md` | 23 | 5 | 2 | 85.00% |
| `_reversa_sdd/internal/tui/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/internal/validation/design.md` | 19 | 5 | 1 | 86.00% |
| `_reversa_sdd/internal/validation/questions.md` | 3 | 3 | 2 | 56.25% |
| `_reversa_sdd/internal/validation/requirements.md` | 21 | 5 | 2 | 83.93% |
| `_reversa_sdd/internal/validation/tasks.md` | 14 | 6 | 3 | 73.91% |
| `_reversa_sdd/internal/worker/coletar-metricas/design.md` | 25 | 6 | 1 | 87.50% |
| `_reversa_sdd/internal/worker/coletar-metricas/requirements.md` | 22 | 5 | 2 | 84.48% |
| `_reversa_sdd/internal/worker/coletar-metricas/tasks.md` | 16 | 6 | 3 | 76.00% |
| `_reversa_sdd/internal/worker/contracts.md` | 10 | 3 | 1 | 82.14% |
| `_reversa_sdd/internal/worker/design.md` | 25 | 6 | 1 | 87.50% |
| `_reversa_sdd/internal/worker/executar-pool-concorrente/design.md` | 25 | 6 | 1 | 87.50% |
| `_reversa_sdd/internal/worker/executar-pool-concorrente/requirements.md` | 22 | 5 | 2 | 84.48% |
| `_reversa_sdd/internal/worker/executar-pool-concorrente/tasks.md` | 16 | 6 | 3 | 76.00% |
| `_reversa_sdd/internal/worker/flows.md` | 7 | 2 | 1 | 80.00% |
| `_reversa_sdd/internal/worker/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/internal/worker/requirements.md` | 22 | 5 | 2 | 84.48% |
| `_reversa_sdd/internal/worker/tasks.md` | 16 | 6 | 3 | 76.00% |
| `_reversa_sdd/inventory.md` | 20 | 2 | 1 | 91.30% |
| `_reversa_sdd/permissions.md` | 33 | 3 | 1 | 93.24% |
| `_reversa_sdd/pkg/errors/contracts.md` | 7 | 3 | 1 | 77.27% |
| `_reversa_sdd/pkg/errors/design.md` | 20 | 5 | 1 | 86.54% |
| `_reversa_sdd/pkg/errors/requirements.md` | 19 | 5 | 2 | 82.69% |
| `_reversa_sdd/pkg/errors/tasks.md` | 13 | 6 | 3 | 72.73% |
| `_reversa_sdd/pkg/logging/contracts.md` | 9 | 3 | 1 | 80.77% |
| `_reversa_sdd/pkg/logging/design.md` | 26 | 6 | 1 | 87.88% |
| `_reversa_sdd/pkg/logging/flows.md` | 7 | 2 | 1 | 80.00% |
| `_reversa_sdd/pkg/logging/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/pkg/logging/requirements.md` | 24 | 4 | 2 | 86.67% |
| `_reversa_sdd/pkg/logging/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/pkg/utils/design.md` | 20 | 5 | 1 | 86.54% |
| `_reversa_sdd/pkg/utils/requirements.md` | 23 | 5 | 2 | 85.00% |
| `_reversa_sdd/pkg/utils/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/pkg/wallet/contracts.md` | 9 | 3 | 1 | 80.77% |
| `_reversa_sdd/pkg/wallet/design.md` | 24 | 6 | 1 | 87.10% |
| `_reversa_sdd/pkg/wallet/questions.md` | 3 | 4 | 2 | 55.56% |
| `_reversa_sdd/pkg/wallet/requirements.md` | 27 | 4 | 2 | 87.88% |
| `_reversa_sdd/pkg/wallet/tasks.md` | 15 | 6 | 3 | 75.00% |
| `_reversa_sdd/questions.md` | 8 | 0 | 0 | 100.00% |
| `_reversa_sdd/state-machines.md` | 45 | 2 | 1 | 95.83% |
| `_reversa_sdd/traceability/code-spec-matrix.md` | 43 | 3 | 0 | 96.74% |
| `_reversa_sdd/traceability/spec-impact-matrix.md` | 45 | 1 | 1 | 96.81% |
| `_reversa_sdd/user-stories/analisar-performance/story.md` | 5 | 1 | 1 | 78.57% |
| `_reversa_sdd/user-stories/gerar-carteira-vanity/story.md` | 5 | 1 | 1 | 78.57% |
| `_reversa_sdd/user-stories/salvar-backup-keystore/story.md` | 5 | 1 | 1 | 78.57% |

---

## Lacunas Pendentes 🔴

- **Lacunas do Revisor pendentes:** 0
- **Perguntas unitárias do Writer:** 35 em 10 arquivos, triadas pelo Revisor; nenhuma pergunta crítica/vermelha pendente.
- **Observação:** símbolos 🔴 residuais em specs per-unit permanecem documentados como pontos de implementação/investigação do legado, mas as 8 decisões humanas solicitadas nesta fase foram resolvidas.

---

## Recomendações

- [ ] Alinhar README ao Cobra real antes de usá-lo como fonte de comportamento.
- [ ] Priorizar segurança: persistência Solana criptografada e migração do `WalletLogger` para logging seguro.
- [ ] Implementar validação por rede em `GenerationCriteria`/`Wallet.IsValid()` antes de reconstrução multirede.
- [ ] Corrigir e reativar o progress manager textual com testes concorrentes.
- [ ] Tratar `EncryptPrivateKeyWithKDF()` como detalhe interno e manter `GenerateKeyStore()` como contrato suportado.

---

## Histórico de Reclassificações e Resoluções

| Item | Antes | Depois | Evidência |
|------|-------|--------|-----------|
| Benchmark README/Cobra | 🟡 | 🟢 | Resposta humana: README desatualizado; `internal/cli/commands.go:845-849` |
| Naming produto/binário | 🔴 | 🟢 | Resposta humana: `bloco-vanity-generator`, binário `bloco-vgen` |
| Validação multirede | 🔴 | 🟢 | Resposta humana: validação por rede |
| `Wallet.IsValid()` | 🔴 | 🟢 | Resposta humana: por `Network` |
| Persistência Solana | 🔴 | 🟢 | Resposta humana: formato criptografado |
| `WalletLogger` | 🔴 | 🟢 | Resposta humana: migrar para logging seguro |
| Progress manager textual | 🔴 | 🟢 | Resposta humana: corrigir e reativar |
| `EncryptPrivateKeyWithKDF()` | 🔴 | 🟢 | Resposta humana: detalhe interno |

## Correções editoriais aplicadas

- Removidas regras de negócio duplicadas em 8 arquivos `requirements.md` durante a revisão.
- Aplicadas decisões humanas nos artefatos `questions.md`, `gaps.md`, `confidence-report.md` e specs afetadas.
