---
schemaVersion: 1
generatedAt: 2026-05-08T12:08:00Z
reversa:
  version: "1.2.34"
kind: handoff
producedBy: orchestrator
hash: "sha256:aadcfddd07eeccce0fcce679324e72aed8538ef6d4aecac71a4fd4b4a6ddad59"
---

# Handoff para o Agente de Codificação

> Este documento é a porta de entrada para o agente de codificação que vai escrever o sistema novo a partir das specs de migração do Reversa.

## Leitura obrigatória primeiro

1. **`paradigm_decision.md`** — leitura inegociável. O paradigma alvo molda como toda a codificação deve acontecer.
2. **`topology_decision.md`** — leitura inegociável. A topologia escolhida define a árvore de pastas e a fronteira entre módulos.

## Decisões centrais já aprovadas

| Decisão | Resultado |
|---|---|
| Paradigma alvo | Go idiomático: CSP/goroutines Go com procedural estruturado e interfaces leves. |
| Apetite | Transformational. |
| Estratégia | Big Bang controlado + Parallel Run offline obrigatório. |
| Topologia | Monólito modular por capability com hexagonal leve. |
| Produto/binário | Produto/repo/docs: `bloco-vanity-generator`; binário: `bloco-vgen`. |
| Dados | Sem banco; persistência por artefatos locais de filesystem. |
| Segurança stdout | Preservar exibição de private key/mnemonic em fluxo single com aviso explícito. |
| CI/CD permissions | Preservar escopos amplos do legado por compatibilidade operacional, com risco documentado. |

## Ordem de leitura recomendada

1. `paradigm_decision.md`
2. `topology_decision.md`
3. `migration_brief.md`
4. `target_business_rules.md`
5. `migration_strategy.md`
6. `target_architecture.md`
7. `target_domain_model.md`
8. `target_data_model.md`
9. `data_migration_plan.md`
10. `parity_specs.md` + `parity_tests/`
11. `risk_register.md` + `cutover_plan.md`
12. `discard_log.md`
13. `ambiguity_log.md`

## Lista de artefatos produzidos

| Artefato | Produzido por | Status |
|---|---|---|
| `migration_brief.md` | orchestrator | criado |
| `paradigm_decision.md` | paradigm_advisor | criado/aprovado |
| `target_business_rules.md` | curator | criado |
| `discard_log.md` | curator | criado |
| `ambiguity_log.md` | orchestrator + agentes | consolidado |
| `migration_strategy.md` | strategist | criado/aprovado |
| `risk_register.md` | strategist | criado |
| `cutover_plan.md` | strategist | criado |
| `topology_decision.md` | designer Fase 1 | criado/aprovado |
| `target_architecture.md` | designer Fase 2 | criado/aprovado |
| `target_domain_model.md` | designer Fase 2 | criado/aprovado |
| `target_data_model.md` | designer Fase 2 | criado/aprovado |
| `data_migration_plan.md` | designer Fase 2 | criado/aprovado |
| `parity_specs.md` | inspector | criado |
| `parity_tests/001-cli-inicializacao-config.feature` | inspector | criado |
| `parity_tests/002-geracao-vanity-worker.feature` | inspector | criado |
| `parity_tests/003-validacao-multirede.feature` | inspector | criado |
| `parity_tests/004-crypto-keystore-kdf.feature` | inspector | criado |
| `parity_tests/005-persistencia-artefatos.feature` | inspector | criado |
| `parity_tests/006-logging-seguro-stdout.feature` | inspector | criado |
| `parity_tests/007-tui-progress-fallback.feature` | inspector | criado |
| `parity_tests/008-ci-release-docker.feature` | inspector | criado |

## Bloqueadores para começar a implementação

Nenhum bloqueador humano pendente. A implementação pode começar, desde que o agente de codificação respeite:

- Leitura obrigatória de `paradigm_decision.md` e `topology_decision.md` antes de criar árvore/código.
- Gates de paridade descritos em `parity_specs.md`.
- Itens referidos à codificação em `ambiguity_log.md`.

## Itens referidos à codificação

| ID | Item | Ação esperada |
|---|---|---|
| COD-PAR-001 | Paridade comportamental vale mais que cópia da estrutura interna do legado. | Implementar desenho Go idiomático com testes de paridade para CLI, crypto, persistência, logging, validação por rede e performance. |
| COD-STR-001 | Big Bang exige gates de paridade antes do release. | Implementar harness de Parallel Run offline e bloquear cutover em divergências críticas. |
| COD-DES-001 | Solana precisa de artefato seguro sem `.key` bruto. | Definir extensão/formato final, implementar round-trip e testes de não geração de `.key`. |
| COD-DES-002 | Não há banco de dados alvo; dados são artefatos locais. | Usar filesystem local com permissões, escrita segura, não sobrescrita e compatibilidade de artefatos. |
| COD-INS-001 | Specs Gherkin são contratos, não testes executáveis. | Traduzir `parity_tests/*.feature` para testes Go/CLI/CI conforme stack alvo. |

## Árvore alvo de referência

A topologia aprovada é a moderna proposta. Use esta árvore como ponto de partida, ajustando apenas quando houver justificativa técnica explícita:

```text
cmd/
  bloco-vgen/
internal/
  app/
    generate/
    stats/
    benchmark/
    version/
  domain/
    wallet/
    criteria/
    network/
    result/
  generation/
    engine/
    match/
    performance/
  crypto/
    ethereum/
    bitcoin/
    solana/
    keystore/
    kdf/
    mnemonic/
  adapters/
    cli/
    filesystem/
    logging/
    terminal/
    release/
  ui/
    tui/
    text/
  config/
pkg/
  errors/
  version/
.github/
  workflows/
Dockerfile
Makefile
```

## Próximos passos para o agente de codificação

1. **Ler `paradigm_decision.md` e internalizar**: o alvo é Go idiomático com CSP/goroutines, procedural estruturado e interfaces leves.
2. **Ler `topology_decision.md` e internalizar**: a topologia escolhida é monólito modular por capability com hexagonal leve.
3. **Configurar o repositório novo** com módulo Go, binário `bloco-vgen`, Cobra/Fang, Bubble Tea/Bubbles/Lip Gloss, Docker multi-stage e GitHub Actions.
4. **Implementar bottom-up**:
   - value objects e validações por rede;
   - crypto determinística/testável por rede;
   - KDF/KeyStore V3/senha segura;
   - filesystem seguro;
   - logging sanitizado;
   - worker engine concorrente;
   - app services;
   - CLI/TUI/fallback;
   - release/CI.
5. **Escrever testes desde o início** usando `parity_specs.md` e `parity_tests/*.feature` como contratos.
6. **Implementar harness de Parallel Run offline** antes do primeiro release candidate.
7. **Executar go/no-go do `cutover_plan.md`** antes de publicar release.
8. **Não migrar banco**: seguir `data_migration_plan.md` para artefatos locais e compatibilidade.

## Critérios mínimos de pronto para release candidate

- `go test` e race detector verdes.
- Vetores de crypto por rede verdes.
- KeyStore V3/KDF com round-trip verde.
- Solana sem `.key` bruto e com artefato seguro validado.
- Logs sem segredo em testes negativos.
- CLI `bloco-vgen` com flags/códigos de saída testados.
- TUI/fallback texto sem deadlock.
- Benchmarks dentro do limiar aceito.
- Docker/release assets/checksums gerados.
- Parallel Run offline sem divergência crítica não explicada.

## Itens auto-decididos

Pipeline executado em modo interativo. Nenhum item foi auto-decidido por `--auto`.

## Notas finais

A migração não autoriza backend, banco, fila, API remota ou event sourcing. O novo sistema continua sendo um CLI local em Go. A transformação é interna: árvore por capability, fronteiras testáveis, segurança de segredos, validação multirede correta e paridade comportamental externa.
