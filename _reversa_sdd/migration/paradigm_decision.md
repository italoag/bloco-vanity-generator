---
schemaVersion: 1
generatedAt: 2026-05-08T10:31:00Z
reversa:
  version: "1.2.34"
kind: paradigm_decision
producedBy: paradigm_advisor
hash: "sha256:2a062a7cb52da79bfd797f29070d5ee0b1832d2389305092610871c984839aa5"
---

# Paradigm Decision

> Decisão consciente sobre como tratar a mudança ou ausência de mudança de paradigma entre o legado e a stack alvo.  
> Este artefato é leitura obrigatória primeiro para qualquer agente posterior e para o agente de codificação.

## Paradigma do legado detectado

- **Paradigma principal**: híbrido: procedural estruturado + CSP/goroutines Go.
- **Confiança**: 🟢 CONFIRMADO para topologia Go/CLI/worker pool; 🟡 INFERIDO para a classificação como híbrido.
- **Evidências**:
  - `_reversa_sdd/architecture.md` descreve aplicação CLI monolítica local em Go, modularizada por pacotes e sem backend, banco, API HTTP ou fila.
  - `_reversa_sdd/architecture.md` classifica a concorrência como worker pool com goroutines e stats collector.
  - `_reversa_sdd/code-analysis.md` descreve fluxo linear `main -> config -> CLI -> worker -> crypto -> filesystem`.
  - `_reversa_sdd/domain.md` modela o domínio por structs e regras simples: `Wallet`, `GenerationCriteria`, `GenerationResult`, KeyStore/KDF e logging seguro.
- **Variações observadas**:
  - CLI/configuração: procedural estruturado, com comandos e handlers orquestrando fluxo síncrono local.
  - Worker/performance: CSP/goroutines, com concorrência controlada por contexto, canais e estatísticas.
  - Domínio: data-oriented Go, com structs e validações, sem DDD/OO pesado.

## Stack alvo declarada

- Linguagem: Go moderno, compatível com a toolchain atual do projeto.
- Framework: Cobra/Fang para CLI; Bubble Tea/Bubbles/Lip Gloss para TUI, salvo decisão posterior.
- Infra: execução local por binário; Docker multi-stage; GitHub Actions para CI/release.

## Paradigma natural inferido

- **Paradigma**: CSP/goroutines Go com procedural estruturado e interfaces leves.
- **Justificativa**: o catálogo mapeia Go como naturalmente orientado a goroutines/channels, com alternativa procedural estruturada. A stack alvo permanece Go e CLI local, então o paradigma natural continua próximo ao legado, mas permite reorganização idiomática.
- **Alternativas viáveis**: OO com DI leve por interfaces Go, útil em fronteiras como crypto, filesystem, logging e validação; funcional leve para funções puras de validação/cálculo; event-driven externo não é indicado porque não há backend, fila ou processo assíncrono distribuído.

## Gap identificado

- **Severidade**: baixo.
- **Implicações concretas**:
  - O fluxo principal `flags -> criteria -> worker pool -> resultado -> filesystem` deve continuar local e síncrono, mas pode ser reorganizado em pacotes menores para reduzir acoplamento de `internal/cli`.
  - A concorrência de geração deve continuar baseada em goroutines/context/canais, preservando cancelamento e coleta de estatísticas descritos em `_reversa_sdd/code-analysis.md`.
  - As entidades `Wallet`, `GenerationCriteria` e `GenerationResult` podem continuar como structs Go simples, mas validações devem ser redesenhadas por rede conforme decisão do Revisor em `_reversa_sdd/confidence-report.md`.
  - A segurança de persistência deve ser modernizada dentro do mesmo paradigma: Solana sem `.key` bruto, logs sanitizados e uso de serviços de alto nível para KeyStore/KDF.
  - A TUI e o progress textual devem preservar UX terminal, mas o progress textual precisa ser redesenhado para evitar os deadlocks documentados nas specs.

## Opções apresentadas ao usuário

1. **Adotar paradigma natural da stack** (transformacional)
   - Consequências: permitir reorganização interna Go mais idiomática, reduzir acoplamento, reforçar interfaces de fronteira e preservar paridade comportamental como contrato.
2. **Forçar paradigma similar ao legado** (conservador)
   - Consequências: preservar estrutura atual de pacotes e fluxos, corrigindo apenas lacunas críticas, com menor risco imediato e maior chance de carregar dívida estrutural.
3. **Híbrido** (equilibrado)
   - Consequências: manter o modelo mental Go/CLI/worker pool, modernizando seletivamente nomes, validação por rede, logging, Solana, progress textual e separação do CLI.

## Decisão do usuário

- **Escolha**: 1.
- **Justificativa do usuário**: Aceitamos reorganizar internamente para um desenho Go mais idiomático, mantendo paridade comportamental.
- **Decidido em**: 2026-05-08T10:31:00Z.

## Apetite derivado

- `derived_appetite`: transformational

## Implicações pendentes para próximos agentes

| Agente | Implicação | Como honrar |
|---|---|---|
| Curator | A migração pode descartar estrutura interna acidental do legado. | Preservar regras e comportamentos confirmados, não a organização atual quando ela for dívida. |
| Curator | Claims do README divergentes não são contrato confirmado. | Curar somente comportamento confirmado por specs/código ou decisões humanas. |
| Strategist | Apetite transformacional permite reconstrução mais limpa. | Recomendar estratégia que maximize redesign controlado com gates de paridade. |
| Designer | Arquitetura alvo deve ser Go idiomática, não cópia de pacotes antigos. | Separar CLI, domínio, geração, persistência, logging e validação com interfaces leves. |
| Inspector | Paridade deve validar comportamento externo, não estrutura interna. | Criar testes focados em CLI, artefatos, segurança, validação por rede e performance. |

## Notas

A decisão transformacional não autoriza mudança de produto para backend, banco ou fila. O alvo continua sendo uma aplicação CLI local em Go. A transformação é interna: desenho Go mais idiomático, maior testabilidade, segurança de segredos e redução de acoplamento, mantendo paridade comportamental e compatibilidade operacional.
