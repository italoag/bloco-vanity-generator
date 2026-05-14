---
schemaVersion: 1
generatedAt: 2026-05-08T11:56:00Z
reversa:
  version: "1.2.34"
kind: parity_specs
producedBy: inspector
hash: "sha256:d3569370d77d2a98d3efb9b0dbe8a6ee11bd33f583935c8907c6340c883a3a38"
---

# Parity Specs

> Especificação de como provar que `bloco-vanity-generator` / `bloco-vgen` é comportamentalmente equivalente ao legado onde isso importa, respeitando a decisão transformacional e a topologia moderna aprovada.

## Entradas consideradas

| Artefato | Uso no Inspector |
|---|---|
| `_reversa_sdd/migration/paradigm_decision.md` | Identificar paradigma legado/alvo e dimensões adicionais de paridade. |
| `_reversa_sdd/migration/migration_strategy.md` | Confirmar Big Bang controlado com Parallel Run offline obrigatório. |
| `_reversa_sdd/migration/target_architecture.md` | Definir componentes/BCs alvo que recebem testes de paridade. |
| `_reversa_sdd/migration/target_domain_model.md` | Mapear aggregates, invariantes e eventos internos/logáveis. |
| `_reversa_sdd/migration/target_data_model.md` | Definir paridade de artefatos locais e ausência de banco. |
| `_reversa_sdd/migration/target_business_rules.md` | Mapear BR-MIGRAR críticas para fluxos de teste. |
| `_reversa_sdd/code-analysis.md` | Extrair fluxos legados confirmados. |
| `_reversa_sdd/flowcharts/*` | Rastrear passos observáveis dos fluxos críticos. |

## Lacuna de base

`_reversa_sdd/characterization_specs/` não existe. Portanto, a suíte de paridade deve ser derivada de `code-analysis.md`, `flowcharts/`, `target_business_rules.md` e artefatos do Designer. O agente de codificação deve transformar estas specs em testes de caracterização executáveis antes do cutover.

## Transição de paradigma

- **Legado**: híbrido — procedural estruturado + CSP/goroutines Go.
- **Alvo**: CSP/goroutines Go com procedural estruturado e interfaces leves.
- **Mudança efetiva**: baixo gap de paradigma, mas transformação interna/topológica relevante.
- **Princípio de paridade**: comportamento externo e invariantes sensíveis são contrato; árvore de pacotes, nomes internos e acoplamentos legados não são contrato.

## Modos de validação selecionados

| Modo | Status | Aplicação |
|---|---|---|
| Shadow mode online | Não aplicável | Não há serviço remoto, tráfego HTTP/RPC ou roteamento gradual. |
| Parallel Run offline | Obrigatório | Rodar legado e alvo em cenários CLI equivalentes antes do release. |
| Characterization tests | Obrigatório | Capturar comportamento legado confirmado: CLI, flags, códigos de saída, stdout/stderr, arquivos, logs e performance. |
| Contract tests | Obrigatório | Validar contratos externos do binário `bloco-vgen`, artefatos filesystem, logging e release assets. |
| Data parity | Obrigatório em filesystem | Comparar formatos, permissões, ausência de `.key` Solana bruto, não sobrescrita e checksums de release. |
| Security parity | Obrigatório | Bloquear vazamento de private key, mnemonic, password, salt, ciphertext, IV, MAC e material KDF em logs. |
| Performance parity | Obrigatório | Comparar throughput/benchmark com tolerância definida pelo owner técnico. |

## Critérios de paridade aceita

### Métrica primária

- **Divergência funcional crítica**: `0` divergências em fluxos de crypto, validação por rede, persistência sensível, logging seguro, códigos de saída e regras de segurança.
- **Vazamento de segredo**: `0` ocorrências em logs, stderr, arquivos não autorizados e relatórios de teste.
- **Contrato CLI**: `100%` dos cenários críticos Gherkin convertidos em testes devem passar.
- **Performance**: throughput do alvo não pode ficar abaixo do limiar definido pelo owner técnico nos benchmarks comparáveis; recomendação inicial: não pior que 10% em cenários equivalentes, salvo justificativa aprovada.

### Janela de observação

- **Pré-cutover**: uma execução completa de CI + Parallel Run offline em ambiente limpo para cada release candidate.
- **Pós-release**: smoke tests manuais/automatizados na janela definida em `cutover_plan.md`.
- **Sem observação online prolongada**: não há tráfego runtime centralizado.

### Critério de bloqueio do cutover

Bloquear release se qualquer item ocorrer:

1. Divergência criptográfica não explicada em Ethereum, Bitcoin ou Solana.
2. `Wallet.IsValid()` ou validação de padrão aceitar/rejeitar formato incorreto por rede.
3. KeyStore V3 incompatível com o contrato confirmado.
4. Solana gerar `.key` bruto ou artefato seguro não recuperável em round-trip.
5. Private key, mnemonic, password, salt, ciphertext, IV, MAC ou material KDF aparecerem em log.
6. Fluxo single exibir segredo sem warning de segurança.
7. Deadlock, race ou timeout em worker/TUI/progress.
8. Artefatos sensíveis sem permissão restrita quando suportada pelo OS.
9. Release sem binários/checksums esperados.

## Cobertura adaptada ao paradigma

Como o alvo mantém Go/CSP mas muda a topologia interna, a paridade deve cobrir mais que entrada/saída simples:

| Dimensão | Cobertura obrigatória | Justificativa |
|---|---|---|
| Contrato externo CLI | Flags, env, códigos de saída, stdout/stderr, quiet/verbose e warnings. | A estrutura interna muda; CLI é fronteira pública. |
| Invariantes dos aggregates | `GenerationRequest`, `VanityCriteria`, `WalletGeneration`, `NetworkWallet`, `SecureArtifactSet`, `TerminalSession`, `SecureTelemetry`. | Topologia moderna move regras para contexts/aggregates. |
| Concorrência CSP | Cancelamento por contexto, primeiro vencedor, stats sem race, timeout e progress lifecycle. | Worker pool é hot path crítico e mudança interna pode introduzir race/deadlock. |
| Side effects isolados | Filesystem, logging, terminal e release devem ser verificáveis por contrato. | Hexagonal leve desloca side effects para adapters. |
| Stochastic parity | Não exigir mesmo endereço aleatório; exigir invariantes, formatos e vetores determinísticos quando possível. | Geração usa `crypto/rand`; paridade literal de outputs seria inválida. |
| Divergências intencionais | Solana sem `.key` bruto; logs sem private key; validação por rede em vez de hex global. | Decisões humanas substituíram comportamentos inseguros/incorretos. |
| Data/filesystem parity | Arquivos, permissões, schemas JSON, não sobrescrita, ausência de banco. | Persistência real é filesystem local. |

## Fluxos críticos cobertos

| Spec ID | Arquivo | Fluxo | Regras principais |
|---|---|---|---|
| PT-001 | `parity_tests/001-cli-inicializacao-config.feature` | Inicialização CLI, config, flags e erros estruturados. | BR-MIGRAR-001..007 |
| PT-002 | `parity_tests/002-geracao-vanity-worker.feature` | Geração single/múltipla, worker pool, cancelamento e stats. | BR-MIGRAR-004, 011..013, 029 |
| PT-003 | `parity_tests/003-validacao-multirede.feature` | Validação por rede e `Wallet.IsValid()`. | BR-MIGRAR-008..010, 014..016 |
| PT-004 | `parity_tests/004-crypto-keystore-kdf.feature` | Ethereum crypto, KeyStore V3, KDF e senha segura. | BR-MIGRAR-014, 017..020, 025 |
| PT-005 | `parity_tests/005-persistencia-artefatos.feature` | Keystore/mnemonic/Solana seguro/permissões/não sobrescrita. | BR-MIGRAR-015..023 |
| PT-006 | `parity_tests/006-logging-seguro-stdout.feature` | Logging sanitizado e stdout de segredo com warning. | BR-MIGRAR-005, 024..025, BR-HUMANA-001 |
| PT-007 | `parity_tests/007-tui-progress-fallback.feature` | TUI, fallback texto, terminal env e progress sem deadlock. | BR-MIGRAR-026..028 |
| PT-008 | `parity_tests/008-ci-release-docker.feature` | CI/CD, Docker, release, checksums e permissões. | BR-MIGRAR-002, 030, BR-HUMANA-002 |

## Matriz de decisão sobre divergências legado → alvo

| Divergência | Aceita? | Como validar |
|---|---|---|
| Binário muda de `bloco-eth` para `bloco-vgen` | Sim | Testar `bloco-vgen`; compatibilidade documentada. |
| README legado anuncia flag não confirmada | Sim | Não testar claims descartadas; ver `discard_log.md`. |
| Prefixo/sufixo deixam de ser hex global para todas as redes | Sim | Testar validação por rede. |
| `Wallet.IsValid()` deixa de ser Ethereum-only | Sim | Testar formatos Ethereum, Bitcoin e Solana. |
| Solana não salva `.key` bruto | Sim, obrigatório | Testar ausência de `.key` e round-trip do artefato seguro. |
| `WalletLogger` não grava private key | Sim, obrigatório | Testar logs por whitelist/negative assertions. |
| Progress textual volta a existir | Sim, obrigatório | Testar lifecycle/cancelamento/timeout. |
| `EncryptPrivateKeyWithKDF()` não é API pública | Sim | Testar serviço de alto nível `GenerateKeyStore`/equivalente. |

## Dados e snapshots

- **Banco**: nenhum snapshot de banco.
- **Filesystem temporário**: cada teste de paridade deve usar diretório temporário isolado.
- **Checksums**: usar para release assets e, em teste local, para comparar schemas/round-trip sem expor segredos.
- **Segredos**: não imprimir valores reais em relatórios; usar marcadores ou hash local efêmero quando necessário.

## Responsabilidades do agente de codificação

1. Traduzir os `.feature` para testes executáveis Go/CLI/CI.
2. Criar harness de Parallel Run offline para comparar legado e alvo onde a divergência não é intencional.
3. Usar vetores determinísticos para crypto/KDF quando possível.
4. Usar diretórios temporários e limpar artefatos sensíveis após testes.
5. Bloquear release se qualquer critério de No-Go for violado.

## Próximos artefatos

Os arquivos em `parity_tests/*.feature` são specs de paridade, não testes executáveis. Eles devem guiar implementação, Inspector review e o handoff final da migração.
