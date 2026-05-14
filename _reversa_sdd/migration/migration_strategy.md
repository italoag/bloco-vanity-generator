---
schemaVersion: 1
generatedAt: 2026-05-08T11:18:00Z
reversa:
  version: "1.2.34"
kind: migration_strategy
producedBy: strategist
hash: "sha256:3a0abe185a7ed72721f3425bd4e6ca7a2861739f983a136178eec72e11c40541"
---

# Migration Strategy

> Avaliação de estratégias de migração para reconstruir o legado como `bloco-vanity-generator`, preservando comportamento externo e permitindo desenho interno Go idiomático.

## Contexto sintetizado

| Dimensão | Síntese | Fonte |
|---|---|---|
| Tamanho do legado | Pequeno/médio: 61 arquivos Go, 13 módulos/pacotes relevantes, 1 entry point CLI, 4 comandos principais. | `_reversa_sdd/inventory.md` |
| Topologia | Monólito CLI local, sem backend, sem banco, sem fila, sem API HTTP/RPC. | `_reversa_sdd/architecture.md` |
| Dados persistidos | Artefatos locais de filesystem: keystore, password, mnemonic, logs e assets de release; sem schema de banco a migrar. | `_reversa_sdd/database/*`, `_reversa_sdd/migration/migration_brief.md` |
| Stack alvo | Go moderno, Cobra/Fang, Bubble Tea/Bubbles/Lip Gloss, Docker multi-stage e GitHub Actions. | `_reversa_sdd/migration/migration_brief.md` |
| Apetite derivado | `transformational`. | `_reversa_sdd/migration/paradigm_decision.md` |
| Gap de paradigma | Baixo: Go CLI local continua; transformação é interna, rumo a Go mais idiomático. | `_reversa_sdd/migration/paradigm_decision.md` |
| Restrições | Sem backend obrigatório, sem banco, sem serviços remotos obrigatórios, manter KeyStore V3/AES-128-CTR/KDF e UX CLI/TUI. | `_reversa_sdd/migration/migration_brief.md` |
| Regras críticas | Crypto multirede, validação por rede, worker pool/performance, persistência segura, logging sem segredos, TUI/progress e CI/release. | `_reversa_sdd/migration/target_business_rules.md` |

## Estratégias filtradas

### Descartadas como estratégia principal

| Estratégia | Motivo de descarte |
|---|---|
| Strangler Fig | Não há backend, roteador, API gateway ou tráfego remoto para estrangular incrementalmente. O produto é um binário CLI local. |

### Mantidas para avaliação

| Estratégia | Aplicabilidade |
|---|---|
| Big Bang controlado | Aplicável porque o sistema é local, pequeno/médio, sem banco, sem integrações runtime remotas e com apetite transformacional. |
| Parallel Run offline | Aplicável para provar equivalência de crypto, CLI, persistência, logging e performance antes de publicar release. |
| Branch by Abstraction seletivo | Aplicável se a equipe decidir migrar pacote a pacote dentro do mesmo repositório, mantendo contratos temporários. |

## Avaliação comparativa

### Estratégia A — Big Bang controlado com gates de paridade

- **Adequação ao apetite**: alta; favorece redesign interno Go idiomático.
- **Adequação ao gap de paradigma**: alta; gap baixo permite reconstrução sem grande fricção conceitual.
- **Custo**: baixo/médio.
- **Risco**: alto sem gates; médio com paridade automatizada.
- **Tempo**: curto/médio.
- **Prós específicos**:
  - Evita carregar acoplamento histórico de `internal/cli`.
  - Permite redesenhar validação por rede, persistência Solana segura e progress textual sem dívida legada.
  - Não exige coexistência em produção porque não há serviço remoto nem banco compartilhado.
  - Facilita renomeação para `bloco-vanity-generator`/`bloco-vgen` de forma coesa.
- **Contras específicos**:
  - Maior risco de regressão criptográfica e de UX CLI se os testes de paridade forem insuficientes.
  - Requer disciplina forte de golden tests para stdout/stderr, arquivos gerados, logs e performance.
  - Requer plano de rollback por release/binário, não por tráfego gradual.

### Estratégia B — Parallel Run offline como estratégia principal

- **Adequação ao apetite**: média; valida bem a equivalência, mas atrasa a transformação.
- **Adequação ao gap de paradigma**: média/alta; útil apesar do gap baixo, porque regras crypto e segurança são críticas.
- **Custo**: alto.
- **Risco**: médio.
- **Tempo**: médio.
- **Prós específicos**:
  - Compara legado e alvo em cenários idênticos antes do release.
  - Excelente para detectar regressão de CLI, stdout, artefatos e performance.
  - Reduz risco em geração Ethereum/Bitcoin/Solana e KeyStore.
- **Contras específicos**:
  - Como estratégia principal, aumenta custo sem necessidade de coexistência runtime real.
  - Algumas mudanças intencionais não terão paridade literal, como Solana sem `.key` bruto e validação por rede.
  - Exige harness de comparação que pode virar trabalho descartável.

### Estratégia C — Branch by Abstraction seletivo

- **Adequação ao apetite**: média/baixa; favorece conservação, mas pode ser usado em hot spots.
- **Adequação ao gap de paradigma**: alta tecnicamente, porque stack continua Go.
- **Custo**: baixo.
- **Risco**: baixo/médio.
- **Tempo**: médio.
- **Prós específicos**:
  - Permite substituir crypto, validação, filesystem e logging atrás de interfaces leves.
  - Reduz risco de regressão incremental no worker pool e KDF.
  - Pode aproveitar testes existentes durante a transição.
- **Contras específicos**:
  - Pode perpetuar estrutura antiga que o Paradigm Advisor autorizou descartar.
  - Menos adequada ao apetite transformacional do usuário.
  - Pode aumentar complexidade temporária em um produto pequeno.

## Estratégia recomendada

**Recomendação do Strategist: Estratégia A — Big Bang controlado com Parallel Run offline como gate obrigatório de validação.**

### Justificativa

1. O sistema é um **binário CLI local** sem estado compartilhado remoto e sem banco a migrar.
2. O legado é **pequeno/médio** e todo em Go, com **gap de paradigma baixo**.
3. O usuário escolheu apetite **transformational**, aceitando reorganização interna Go idiomática.
4. As regras críticas são testáveis por contrato externo: CLI, stdout/stderr, arquivos, logs, crypto, KDF e performance.
5. O maior risco não é cutover operacional, mas **regressão comportamental/segurança**; por isso o Big Bang deve ser bloqueado por gates de paridade offline.

### Forma concreta recomendada

- Reconstruir internamente como Go idiomático em uma linha de trabalho única.
- Criar suíte de paridade antes do release:
  - CLI flags e códigos de saída.
  - Validação por rede.
  - Vetores determinísticos de crypto quando possível.
  - KeyStore V3/KDF.
  - Persistência local e permissões.
  - Logging sem segredos.
  - TUI/fallback textual.
  - Benchmark/performance comparável.
- Publicar release somente após go/no-go verde.
- Manter release/binário legado como rollback até validação pós-release.

## Estratégia não recomendada, mas viável

**Branch by Abstraction seletivo** é alternativa aceitável se o usuário preferir reduzir risco técnico imediato. Nesse caso, usar abstrações apenas nas bordas críticas: geração por rede, validação, filesystem, logging, KDF e TUI. Não recomendado como estratégia principal porque contraria parcialmente o objetivo transformacional e pode preservar acoplamentos do legado.

## Decisão humana registrada

O usuário escolheu a opção recomendada:

**Big Bang controlado + Parallel Run offline obrigatório.**

### Opções apresentadas

1. **Big Bang controlado + Parallel Run offline obrigatório** — recomendado.
2. **Parallel Run offline como estratégia principal** — mais seguro, mais caro e mais lento.
3. **Branch by Abstraction seletivo** — mais incremental, menos transformacional.

## Implicações para próximos agentes

| Agente | Implicação |
|---|---|
| Designer | Desenhar arquitetura alvo Go idiomática, sem copiar pacotes antigos como contrato. |
| Designer | Definir fronteiras testáveis para CLI, domínio, geração, validação, KDF, filesystem, logging e TUI. |
| Inspector | Produzir parity specs que bloqueiem release se houver vazamento de segredo, regressão crypto ou queda de performance relevante. |
| Coding | Implementar em ordem que maximize gates: domínio/validação/crypto primeiro, CLI/TUI depois, release por último. |
