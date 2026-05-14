# Perguntas para Validação — bloco-vanity-generator

> Gerado pelo Revisor em 2026-05-08T07:40:44Z
> Respostas recebidas no chat e consolidadas pelo Revisor em 2026-05-08T09:42:40Z.
> Revisado em 2026-05-14: decisões transversais mantidas e usadas para responder os `questions.md` unitários.

---

## Pergunta 1

**Contexto:** Naming transversal — `go.mod` usa módulo/imports `bloco-vgen`, Dockerfile/CI geram binário `bloco-vgen`, README usa exemplos `./bloco-vanity-generator`, repositório é `bloco-wallet-generator`.
**Spec afetada:** `_reversa_sdd/architecture.md; _reversa_sdd/traceability/spec-impact-matrix.md`
**Pergunta:** Qual deve ser o nome canônico do produto, binário, módulo Go, imagem Docker e exemplos de documentação?
**Impacto:** Define se a migração deve preservar `bloco-vgen`, renomear para `bloco-wallet-generator`, ou manter aliases/compatibilidade. Afeta documentação, CI/CD, Docker, imports e UX da CLI.

**Resposta:** bloco-vanity-generator para produto/repositório/documentação; bloco-vgen como binário compatível.

**Decisão consolidada:** A nomenclatura-alvo deve normalizar produto, repo e docs como `bloco-vanity-generator`, mantendo `bloco-vgen` como binário compatível. 🟢

---

## Pergunta 2

**Contexto:** Divergência README vs Cobra — README anuncia `--optimize-for`, `benchmark --pattern` e `benchmark --threads`; `internal/cli/commands.go:845-849` confirma no subcomando benchmark apenas `--attempts`, `--duration`, `--detailed`.
**Spec afetada:** `_reversa_sdd/internal/cli/requirements.md; _reversa_sdd/traceability/spec-impact-matrix.md`
**Pergunta:** A documentação do README representa roadmap desejado ou está desatualizada em relação ao código atual?
**Impacto:** Se for roadmap, as specs devem manter esses itens como requisitos futuros. Se estiver desatualizada, a reimplementação deve seguir apenas as flags reais do Cobra e corrigir/remover os exemplos.

**Resposta:** README informa features que não foram implementadas e está desatualizado.

**Decisão consolidada:** Claims do README que não aparecem no Cobra devem ser tratados como documentação desatualizada, não como comportamento legado confirmado. 🟢

---

## Pergunta 3

**Contexto:** Multirede vanity — `GenerationCriteria.Validate()` exige prefixo/sufixo hexadecimais em `pkg/wallet/types.go:121-130`, mas a CLI aceita `--network bitcoin|solana` e esses endereços usam Base58/base58-like.
**Spec afetada:** `_reversa_sdd/pkg/wallet/requirements.md; _reversa_sdd/internal/crypto/gerar-carteira-multirede/requirements.md`
**Pergunta:** Para Bitcoin e Solana, padrões vanity devem continuar restritos a hex por compatibilidade com o legado, ou devem validar alfabeto da rede (Base58/bech32 quando aplicável)?
**Impacto:** Define se a regra atual é bug técnico a corrigir ou comportamento legado intencional. Afeta validação, worker matching, testes e documentação multirede.

**Resposta:** Validar por rede, ex. Base58/bech32 quando aplicável.

**Decisão consolidada:** A validação-alvo de padrões vanity deve ser específica por rede: Ethereum hexadecimal/EIP-55; Bitcoin Base58/bech32 conforme tipo; Solana Base58. 🟢

---

## Pergunta 4

**Contexto:** Validação de sucesso — `Wallet.IsValid()` exige endereço com 40 chars e private key com 64 chars (`pkg/wallet/types.go:76-82`), formato centrado em Ethereum. Bitcoin/Solana têm tamanhos e formatos diferentes.
**Spec afetada:** `_reversa_sdd/pkg/wallet/requirements.md; _reversa_sdd/internal/worker/requirements.md`
**Pergunta:** `Wallet.IsValid()` deve permanecer Ethereum-only ou virar validação por `Network` para Bitcoin/Solana?
**Impacto:** Afeta `GenerationResult.IsSuccessful()`, worker pool, retorno da CLI e critérios de paridade para geração multirede.

**Resposta:** Validar `Wallet.IsValid()` por `Network`.

**Decisão consolidada:** `Wallet.IsValid()` deve evoluir para validação por rede, não permanecer centrado apenas em Ethereum. 🟢

---

## Pergunta 5

**Contexto:** Persistência Solana — `SaveKeyStoreFilesToDisk()` salva keypair JSON e também `.key` com private key bruta para Solana (`internal/crypto/keystore.go:1405-1445`).
**Spec afetada:** `_reversa_sdd/internal/cli/salvar-keystore/requirements.md; _reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md`
**Pergunta:** A persistência Solana deve gerar arquivo `.key` com chave privada bruta, usar formato criptografado, ou ser explicitamente não suportada?
**Impacto:** Define requisito de segurança e paridade. Pode exigir mudança de spec de backup, permissões, mensagens de warning e testes de recuperação Solana.

**Resposta:** Usar formato criptografado para persistência Solana.

**Decisão consolidada:** Persistência Solana deve evitar `.key` bruto e usar formato criptografado/seguro. 🟢

---

## Pergunta 6

**Contexto:** Logger legado — `pkg/wallet/logger.go:37-65` escreve header com `PRIVATE_KEY` e grava `result.Wallet.PrivateKey` em `wallets-YYYYMMDD.log` com modo `0644`.
**Spec afetada:** `_reversa_sdd/pkg/wallet/requirements.md; _reversa_sdd/pkg/logging/requirements.md`
**Pergunta:** O `WalletLogger` legado deve ser preservado por compatibilidade, desativado, migrado para logging seguro, ou removido?
**Impacto:** Afeta segurança operacional, requisitos de auditoria, migração de logs existentes e critérios de não vazamento de segredos.

**Resposta:** Migrar `WalletLogger` legado para logging seguro.

**Decisão consolidada:** `WalletLogger` não deve preservar escrita de private key em claro; comportamento deve migrar para logging seguro/sanitizado. 🟢

---

## Pergunta 7

**Contexto:** Progress manager textual — Architect registrou deadlocks; specs indicam que fluxo CLI evita progress manager textual e usa TUI/fallback texto simples.
**Spec afetada:** `_reversa_sdd/internal/progress/requirements.md; _reversa_sdd/traceability/spec-impact-matrix.md`
**Pergunta:** O progress manager textual deve ser corrigido e reativado, mantido como desabilitado, ou removido da reimplementação?
**Impacto:** Define escopo de reconstrução do módulo `internal/progress`, testes concorrentes e compatibilidade da UX sem TUI.

**Resposta:** Corrigir e reativar progress manager textual.

**Decisão consolidada:** O progress manager textual deve ser corrigido contra deadlocks e reativado como fallback textual. 🟢

---

## Pergunta 8

**Contexto:** Keystore API direta — `EncryptPrivateKeyWithKDF()` ainda cria keystore com endereço placeholder zero em `internal/crypto/keystore.go:1049-1054`; `GenerateKeyStore()` corrige o endereço depois em `internal/crypto/keystore.go:1287-1292`.
**Spec afetada:** `_reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md; _reversa_sdd/internal/crypto/contracts.md`
**Pergunta:** `EncryptPrivateKeyWithKDF()` é contrato público a preservar/corrigir, ou detalhe interno que só deve ser usado via `GenerateKeyStore()`?
**Impacto:** Define se a reimplementação precisa expor função direta segura com endereço real ou manter apenas o serviço de alto nível como contrato suportado.

**Resposta:** `EncryptPrivateKeyWithKDF()` é detalhe interno.

**Decisão consolidada:** `EncryptPrivateKeyWithKDF()` não deve ser contrato público; consumidores devem usar `GenerateKeyStore()`/serviço de alto nível. 🟢

---
