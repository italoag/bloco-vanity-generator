---
schemaVersion: 1
generatedAt: 2026-05-08T11:28:00Z
reversa:
  version: "1.2.34"
kind: topology_decision
producedBy: designer
hash: "sha256:e786213874896e907e5da36d240036ba4bb9eec4d3fafe1eb35ed7fc127a94b6"
---

# Topology Decision

> Decisão consciente sobre como organizar o sistema novo: preservar a topologia do legado, adotar uma topologia moderna ou aplicar um híbrido.
> Este artefato é leitura obrigatória do próprio Designer para decompor bounded contexts e do agente de codificação para criar a árvore de pastas.

## Topologia do legado detectada

- **Padrão organizacional**: híbrido: monólito CLI local com package-by-layer leve e módulos por responsabilidade técnica.
- **Confiança**: 🟢 CONFIRMADO.
- **Evidências**:
  - `_reversa_sdd/architecture.md` descreve o sistema como aplicação CLI monolítica local em Go, sem backend, banco, API HTTP ou fila, com bootstrap em `cmd/`, orquestração em `internal/cli`, configuração em `internal/config`, geração em `internal/crypto`, concorrência em `internal/worker`, terminal em `internal/tui` e tipos utilitários em `pkg/`.
  - `_reversa_sdd/architecture.md` classifica a topologia como monólito CLI local e a modularização como camadas/pacotes por responsabilidade.
  - `_reversa_sdd/inventory.md` confirma 61 arquivos Go, entry point único em `cmd/bloco-vgen/`, comandos em `internal/cli` e pacotes técnicos em `internal/*` e `pkg/*`.
  - `_reversa_sdd/dependencies.md` confirma Go Modules, dependências de CLI/TUI/crypto e ausência de stack web, banco ou orquestração distribuída.
- **Mapa da árvore legada** (resumido):
  ```text
  cmd/
    bloco-vgen/
      main.go
  internal/
    cli/
    config/
    crypto/
      kdf/
    progress/
    tui/
    validation/
    worker/
  pkg/
    errors/
    logging/
    utils/
    wallet/
  .github/
    workflows/
  Dockerfile
  Makefile
  ```

## Diagnóstico estrutural

- **Acoplamento**: médio. O legado tem pacotes separados, mas `_reversa_sdd/architecture.md` mostra `internal/cli.Application` orquestrando flags, geração, stats, benchmark, keystore, TUI/texto e filesystem, concentrando responsabilidades de fluxo.
- **Coesão por módulo**: média/alta. Pacotes técnicos como crypto, worker, validation, TUI, logging e wallet têm responsabilidades reconhecíveis em `_reversa_sdd/inventory.md`, mas a separação é mais técnica do que orientada a capabilities ou contratos de domínio.
- **Módulos órfãos / mortos**: nenhum confirmado nesta fase. `internal/progress` é módulo de risco/dívida, não órfão confirmado, porque o Revisor decidiu corrigir e reativar o progress manager textual.
- **Camadas redundantes**: nenhuma camada redundante confirmada. Há, porém, sobreposição conceitual entre validação no domínio, worker e crypto que deve ser resolvida no desenho alvo.
- **Violações de fronteira**: parcialmente problemáticas. O fluxo principal mistura CLI, aplicação, geração, persistência e apresentação no mesmo eixo de orquestração; `WalletLogger` legado também mistura domínio e logging, segundo decisões consolidadas pelo Revisor e curadas em `target_business_rules.md`.
- **Mistura de paradigmas/estilos**: híbrido Go aceitável: procedural estruturado no CLI/configuração, CSP/goroutines no worker pool e structs/data-oriented no domínio, conforme `paradigm_decision.md`.
- **Avaliação geral**: parcialmente problemática. A estrutura é compreensível para um CLI Go pequeno/médio, mas centraliza demais a orquestração no CLI e não expressa explicitamente capabilities, portas de persistência, segurança de segredos e contratos de paridade.

## Topologia moderna proposta

- **Padrão**: modular monolith CLI por capability, com hexagonal leve nas bordas críticas.
- **Justificativa**: o alvo continua sendo um binário Go local; portanto, microserviços, backend, banco e event-driven externo seriam excesso. A decisão transformacional e a estratégia Big Bang permitem redesenho interno mais profundo, mas o Parallel Run offline exige fronteiras testáveis. A topologia proposta organiza o código por capabilities de produto e isola bordas de CLI, filesystem, logging, TUI e crypto sem introduzir framework pesado.
- **Ganhos concretos esperados**:
  - Testabilidade por contrato externo: CLI, geração, validação por rede, KeyStore/KDF, filesystem, logging e TUI podem ter golden tests/parity tests isolados.
  - Menor acoplamento do CLI: comandos deixam de carregar detalhes de worker, crypto, persistência e logging.
  - Segurança de segredos mais auditável: persistência e logging viram bordas explícitas com testes negativos de vazamento.
  - Onboarding mais rápido: árvore organizada por capacidades do produto, não só por tipo técnico de pacote.
  - Melhor suporte ao Big Bang controlado: o sistema novo pode ser reconstruído sem copiar a estrutura antiga, mas com gates por capability.
- **Custo / risco**:
  - Exige reorientar a equipe do mapa `internal/cli`/`internal/crypto` para capabilities e portas.
  - Pode parecer estruturalmente maior que o legado; deve ser mantido leve e idiomático em Go, sem DI container pesado.
  - Requer disciplina para não transformar cada pacote legado em bounded context 1-para-1.
- **Esboço da árvore proposta**:
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

## Opções apresentadas ao usuário

1. **Preservar topologia legada** (conservador)
   - Consequências: mantém `cmd/`, `internal/cli`, `internal/crypto`, `internal/worker`, `internal/tui` e `pkg/*` como mapa principal; reduz risco de migração; perpetua parte do acoplamento do CLI e a separação por responsabilidade técnica.
2. **Adotar topologia moderna proposta** (transformacional)
   - Consequências: reorganiza por capabilities com bordas hexagonais leves; maximiza a decisão de Go idiomático e Big Bang controlado; aumenta esforço inicial e exige paridade rigorosa.
3. **Híbrido** (equilibrado)
   - Consequências: preserva `cmd/`, `internal/crypto`, `internal/tui`, `.github`, Dockerfile e Makefile como âncoras reconhecíveis, mas introduz `internal/app`, `internal/domain`, `internal/generation` e `internal/adapters` para reduzir acoplamento do CLI e isolar segurança/persistência/logging.

## Decisão do usuário

- **Escolha**: 2 — Adotar topologia moderna proposta.
- **Justificativa do usuário**: aprovado conforme recomendação do Designer.
- **Decidido em**: 2026-05-08T11:31:00Z.

## Recomendação do Designer

**Opção 2 — Adotar topologia moderna proposta.**

Justificativa: a escolha anterior foi transformacional e a estratégia confirmada é Big Bang controlado com gates de paridade. Como o sistema é um CLI local pequeno/médio e sem banco, o custo de reorganização é controlado, enquanto os ganhos de testabilidade, segurança de segredos e isolamento de hot paths são relevantes.

## Mapeamento legado → novo

| Módulo / pasta legada | Bounded context novo | Tipo | Observações |
|---|---|---|---|
| `cmd/bloco-vgen/` | Experiência de Comando | renomeado | Renomear entry point para `cmd/bloco-vgen/`, mantendo compatibilidade do binário alvo. |
| `internal/cli/` | Experiência de Comando + Casos de Uso | dividido | Separar parsing/apresentação CLI de orquestração de aplicação. |
| `internal/config/` | Configuração Operacional | preservado com ajuste | Manter configuração local, mas com contratos explícitos para app/adapters. |
| `internal/crypto/` + `internal/crypto/kdf/` | Criptografia Multirede + Cofre Local | dividido | Separar geração por rede, KeyStore/KDF e mnemonic para testabilidade e segurança. |
| `internal/validation/` | Critérios e Matching | fundido | Unificar validação por rede e matching vanity no desenho de domínio/generation. |
| `internal/worker/` | Motor de Geração Vanity | preservado conceitualmente | Manter CSP/goroutines, mas isolar engine/hot path e métricas. |
| `internal/progress/` + `internal/tui/` | Interface Terminal | fundido | Tratar TUI e fallback textual como variantes de apresentação terminal. |
| `pkg/wallet/` | Domínio de Carteira | dividido | Mover invariantes para domínio; remover mistura com logging legado. |
| `pkg/logging/` | Observabilidade Segura | preservado conceitualmente | Logging seguro como adapter explícito, sem private key/mnemonic/salt. |
| `pkg/errors/` | Erros Compartilhados | preservado | Manter utilitário leve, evitando virar domínio. |
| `pkg/utils/` | Capabilities específicas | dividido | Funções de probabilidade, formatação e helpers devem migrar para o contexto que as usa. |
| `.github/workflows/` | Distribuição e Qualidade | preservado com risco aceito | Preservar escopos amplos conforme decisão humana, com documentação do risco. |
| `keystores/` | Artefatos Locais Seguros | preservado como dado | Diretório de saída, não bounded context; novos artefatos não devem sobrescrever dados do usuário. |
| Claims README não implementados | (descartado) | removido | Ver `discard_log.md`, especialmente descarte de claims divergentes do README legado. |

## Implicações pendentes para próximos passos do Designer

| Etapa do Designer | Implicação | Como honrar |
|---|---|---|
| Bounded contexts | Não copiar pacotes legados 1-para-1. | Agrupar por invariantes/capabilities: comando, geração, crypto, artefatos, terminal, observabilidade e distribuição. |
| `target_architecture` | Big Bang permite redesign interno, mas Parallel Run exige contratos comparáveis. | Desenhar portas leves para CLI, geração, filesystem, logging, TUI e crypto. |
| `target_domain_model` | Domínio deve continuar Go data-oriented, sem DDD pesado artificial. | Modelar structs/value objects e serviços de domínio com invariantes explícitas por rede. |
| `target_data_model` | Não há banco; dados são artefatos locais. | Modelar arquivos, permissões, formato, owner e compatibilidade/backup. |
| `data_migration_plan` | Não há ETL centralizado. | Definir migração/compatibilidade de arquivos locais e política de não sobrescrita. |

## Notas

A topologia moderna proposta não implica backend, banco, fila, microsserviços ou event sourcing. O produto continua um CLI local em Go. A modernização é interna e deve ser mantida idiomática: interfaces pequenas, composição simples, goroutines/context para concorrência, e testes de paridade como contrato de release.
