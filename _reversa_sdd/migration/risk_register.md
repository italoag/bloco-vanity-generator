---
schemaVersion: 1
generatedAt: 2026-05-08T11:18:00Z
reversa:
  version: "1.2.34"
kind: risk_register
producedBy: strategist
hash: "sha256:de0b1148f9cda0faeba678e9856b16917a0bad71a1dbf82a1053e56704b6057d"
---

# Risk Register

> Registro de riscos da estratégia recomendada: Big Bang controlado com Parallel Run offline como gate obrigatório.

## Escala

| Campo | Valores |
|---|---|
| Probabilidade | baixa, média, alta |
| Impacto | baixo, médio, alto, crítico |
| Severidade | LOW, MEDIUM, HIGH, CRITICAL |

## Riscos

### RISK-001 — Regressão criptográfica multirede

- **Categoria**: técnica / segurança
- **Descrição**: A reconstrução pode gerar endereços, checksums, chaves ou artefatos de rede divergentes para Ethereum, Bitcoin ou Solana.
- **Fonte**: `_reversa_sdd/domain.md` BR-012..BR-018; `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-014..016.
- **Probabilidade**: média
- **Impacto**: crítico
- **Severidade**: CRITICAL
- **Mitigação**: usar vetores determinísticos, testes por rede, comparação com bibliotecas de referência e revisão de segurança para crypto/KDF.
- **Contingência**: bloquear release; manter binário legado como distribuição recomendada até correção.
- **Owner**: Revisor de segurança / mantenedor técnico.

### RISK-002 — Vazamento de segredos em logs, stdout ou arquivos

- **Categoria**: segurança operacional
- **Descrição**: Private key, mnemonic, password ou salt podem aparecer indevidamente em logs, stdout capturado ou arquivos locais.
- **Fonte**: `_reversa_sdd/migration/migration_brief.md` métricas; `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-023..025; decisão BR-HUMANA-001.
- **Probabilidade**: média
- **Impacto**: crítico
- **Severidade**: CRITICAL
- **Mitigação**: testes negativos de redaction, snapshots de stdout/stderr, lint/review de logging e permissões restritas em filesystem.
- **Contingência**: revogar release, publicar aviso de segurança, remover artefatos afetados e corrigir sanitização.
- **Owner**: Revisor de segurança.

### RISK-003 — Solana persistida em formato inseguro ou não recuperável

- **Categoria**: dados / segurança
- **Descrição**: A substituição do `.key` bruto por formato seguro pode gerar artefato não importável, não recuperável ou insuficientemente protegido.
- **Fonte**: `_reversa_sdd/questions.md` Pergunta 5; `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-016.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: Designer deve especificar formato, threat model mínimo e testes de round-trip para Solana.
- **Contingência**: marcar persistência Solana como experimental/desabilitada até formato aprovado.
- **Owner**: Designer / Revisor de segurança.

### RISK-004 — Performance abaixo do worker pool legado

- **Categoria**: performance
- **Descrição**: Redesign interno pode degradar throughput de geração vanity, especialmente no hot path concorrente.
- **Fonte**: `_reversa_sdd/migration/migration_brief.md` métricas; `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-013 e BR-MIGRAR-029.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: benchmarks comparativos, limites mínimos de throughput por rede/cenário e profiling antes do release.
- **Contingência**: adiar release; otimizar hot path ou reter worker engine legado atrás de interface temporária.
- **Owner**: Mantenedor técnico / engenheiro de performance.

### RISK-005 — Paridade CLI quebrada

- **Categoria**: produto / UX
- **Descrição**: Flags, códigos de saída, mensagens, quiet/verbose, warnings e fallback podem divergir do comportamento esperado.
- **Fonte**: `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-003..012 e BR-MIGRAR-026..028.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: golden tests de CLI, matriz de flags e testes de códigos de saída.
- **Contingência**: manter release candidato privado até correção; documentar mudanças intencionais.
- **Owner**: Owner técnico / usuário CLI validador.

### RISK-006 — TUI/progress textual reintroduzir deadlocks

- **Categoria**: concorrência / UX
- **Descrição**: Reativação do progress manager textual pode introduzir deadlocks ou interferência com TUI/stdout.
- **Fonte**: `_reversa_sdd/gaps.md` GAP-RV-007; `_reversa_sdd/migration/target_business_rules.md` BR-MIGRAR-028.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: testes com timeout, cancelamento por contexto, race detector e cenários TUI/fallback.
- **Contingência**: desabilitar progress textual por flag/fallback simples até correção.
- **Owner**: Engenheiro de CLI/TUI.

### RISK-007 — Big Bang gerar regressões amplas difíceis de isolar

- **Categoria**: estratégia
- **Descrição**: Uma reconstrução concentrada pode combinar regressões de crypto, CLI, logging e release em um único corte.
- **Fonte**: `_reversa_sdd/migration/migration_strategy.md` recomendação.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: decompor implementação em marcos internos com gates, embora o cutover seja Big Bang.
- **Contingência**: converter para Branch by Abstraction seletivo nos módulos que falharem nos gates.
- **Owner**: Owner técnico.

### RISK-008 — Reorganização Go idiomática perder comportamento externo

- **Categoria**: paradigma / arquitetura
- **Descrição**: A decisão transformacional pode priorizar limpeza interna e perder detalhes externos relevantes do legado.
- **Fonte**: `_reversa_sdd/migration/paradigm_decision.md` § Implicações; `_reversa_sdd/migration/target_business_rules.md`.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: paridade baseada em comportamento externo, rastreabilidade das regras BR-MIGRAR e revisão do Inspector.
- **Contingência**: restaurar comportamento externo mesmo que a implementação interna precise ser ajustada.
- **Owner**: Designer / Inspector.

### RISK-009 — Divergência entre documentação antiga e contrato real

- **Categoria**: produto / documentação
- **Descrição**: README legado contém claims não implementados; migrá-los por engano inflaria escopo e quebraria expectativas.
- **Fonte**: `_reversa_sdd/questions.md` Pergunta 2; `_reversa_sdd/migration/discard_log.md` BR-DESCARTAR-002.
- **Probabilidade**: baixa
- **Impacto**: médio
- **Severidade**: MEDIUM
- **Mitigação**: tratar README antigo como fonte secundária e gerar documentação alvo a partir da CLI real.
- **Contingência**: mover claims não implementados para roadmap explícito.
- **Owner**: Mantenedor/release.

### RISK-010 — Escopos amplos de release preservarem risco de supply chain

- **Categoria**: segurança / CI-CD
- **Descrição**: O usuário decidiu preservar escopos amplos do release por compatibilidade operacional, mantendo risco residual de permissões excessivas.
- **Fonte**: `_reversa_sdd/migration/ambiguity_log.md` BR-HUMANA-002; `_reversa_sdd/permissions.md` PERM-GAP-004.
- **Probabilidade**: baixa
- **Impacto**: alto
- **Severidade**: MEDIUM
- **Mitigação**: documentar justificativa, revisar secrets, exigir branch protection e revisar workflows em PR.
- **Contingência**: reduzir escopos após falha ou evidência de não necessidade.
- **Owner**: Mantenedor/release / revisor de segurança.

### RISK-011 — Ausência de prazo/orçamento explícito afetar planejamento

- **Categoria**: organizacional
- **Descrição**: Brief não informa prazo nem orçamento, então a estratégia recomendada pode ser sensível à capacidade real disponível.
- **Fonte**: `_reversa_sdd/migration/migration_brief.md` § Restrições.
- **Probabilidade**: média
- **Impacto**: médio
- **Severidade**: MEDIUM
- **Mitigação**: quebrar plano em marcos de validação e estimar effort antes da execução.
- **Contingência**: reduzir escopo inicial para Ethereum + logging/keystore, mantendo Bitcoin/Solana em marco posterior se necessário.
- **Owner**: Owner técnico.

### RISK-012 — Dependências externas mudarem comportamento

- **Categoria**: dependências
- **Descrição**: Bibliotecas Go de Ethereum, Bitcoin, Solana, TUI ou KDF podem ter diferenças de versão ou comportamento em relação ao legado.
- **Fonte**: `_reversa_sdd/dependencies.md`.
- **Probabilidade**: média
- **Impacto**: médio
- **Severidade**: MEDIUM
- **Mitigação**: fixar versões compatíveis, rodar `go test`, `go mod tidy`, govulncheck e testes de vetores.
- **Contingência**: pin/downgrade de dependência ou adapter de compatibilidade.
- **Owner**: Mantenedor técnico.

### RISK-013 — Dados locais legados não terem migração automática

- **Categoria**: dados
- **Descrição**: Como não há banco, a migração depende de artefatos locais já existentes; usuários podem ter keystores/logs antigos em formatos ou permissões diversas.
- **Fonte**: `_reversa_sdd/database/*`; `_reversa_sdd/migration/migration_brief.md`.
- **Probabilidade**: baixa
- **Impacto**: médio
- **Severidade**: MEDIUM
- **Mitigação**: documentar compatibilidade/importação, não sobrescrever arquivos existentes e validar permissões em novos artefatos.
- **Contingência**: fornecer guia manual de backup/restore e ferramenta futura de verificação.
- **Owner**: Mantenedor/release.

### RISK-014 — Capacidade do time em crypto/TUI/concorrência ser insuficiente

- **Categoria**: organizacional / capacidade
- **Descrição**: A migração exige domínio de crypto, Go concurrency, TUI e segurança de filesystem.
- **Fonte**: `_reversa_sdd/architecture.md`; `_reversa_sdd/dependencies.md`.
- **Probabilidade**: média
- **Impacto**: alto
- **Severidade**: HIGH
- **Mitigação**: revisão especializada, pair programming em módulos críticos e execução obrigatória de race/security tests.
- **Contingência**: reduzir escopo ou manter componentes legados críticos temporariamente.
- **Owner**: Owner técnico.

## Riscos críticos destacados

| ID | Motivo |
|---|---|
| RISK-001 | Crypto incorreta pode gerar fundos irrecuperáveis ou endereços inválidos. |
| RISK-002 | Vazamento de segredo compromete usuários. |

## Observações

A mudança de paradigma foi classificada como baixa, mas a estratégia transformacional ainda exige risco operacional explícito porque a reconstrução Big Bang troca a organização interna de vários módulos de uma vez. O principal controle é o Parallel Run offline como gate, não coexistência runtime.
