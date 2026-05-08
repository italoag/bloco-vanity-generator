# Plano de Exploração — bloco-wallet-generator

> Criado pelo Reversa em 2026-05-08
> Marque cada tarefa com ✅ quando concluída.
> Você pode editar este plano antes de iniciar: adicione, remova ou reordene tarefas conforme necessário.

---

## Fase 1: Reconhecimento 🔍

- [x] ✅ **Scout** — Mapeamento de estrutura de pastas e tecnologias
- [x] ✅ **Scout** — Análise de dependências e gerenciadores de pacotes
- [x] ✅ **Scout** — Identificação de entry points, CI/CD e configurações

## Decisão de organização das specs 🗂️

> Entre o Scout e o Arqueólogo, o Reversa pergunta como você quer organizar as specs (por módulo, caso de uso, endpoint, híbrida, por features ou customizada). A escolha fica persistida em `.reversa/config.toml` na seção `[specs]` e não será reperguntada em execuções futuras. Para reapresentar o menu, remova manualmente a seção.

## Fase 2: Escavação 🏗️

> Preenchido pelo Reversa após o Scout concluir o reconhecimento.

- [x] ✅ **Archaeologist** — Análise do módulo `cmd/bloco-eth`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/cli`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/config`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/crypto`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/crypto/kdf`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/progress`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/tui`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/validation`
- [x] ✅ **Archaeologist** — Análise do módulo `internal/worker`
- [x] ✅ **Archaeologist** — Análise do módulo `pkg/errors`
- [x] ✅ **Archaeologist** — Análise do módulo `pkg/logging`
- [x] ✅ **Archaeologist** — Análise do módulo `pkg/utils`
- [x] ✅ **Archaeologist** — Análise do módulo `pkg/wallet`

## Fase 3: Interpretação 🧠

- [x] ✅ **Detetive** — Arqueologia Git e ADRs retroativos
- [x] ✅ **Detetive** — Regras de negócio implícitas e máquinas de estado
- [x] ✅ **Detetive** — Matriz de permissões (RBAC/ACL)
- [x] ✅ **Arquiteto** — Diagramas C4 (Contexto, Containers, Componentes)
- [x] ✅ **Arquiteto** — ERD completo e integrações externas
- [x] ✅ **Arquiteto** — Spec Impact Matrix

## Fase 4: Geração 📝

- [x] ✅ **Redator** — Specs SDD por componente
- [x] ✅ **Redator** — OpenAPI não aplicável (sem API HTTP/RPC)
- [x] ✅ **Redator** — User Stories (se aplicável)
- [x] ✅ **Redator** — Code/Spec Matrix

## Fase 5: Revisão ✅

- [x] ✅ **Revisor** — Revisão cruzada de specs
- [x] ✅ **Revisor** — Resolução de lacunas com o usuário
- [x] ✅ **Revisor** — Relatório de confiança final

---

## Agentes Independentes

> Execute estes agentes quando os recursos estiverem disponíveis — podem rodar em qualquer fase.

- [ ] **Visor** — Análise de interface via screenshots
- [x] ✅ **Data Master** — Análise completa do banco de dados
- [ ] **Design System** — Extração de tokens de design
- [ ] **Tracer** — Análise dinâmica (requer sistema acessível)

---

## Próximo passo

Após o Time de Descoberta concluir e o `_reversa_sdd/` estar populado, você pode disparar um dos fluxos seguintes:

- `/reversa-migrate`: orquestrador do **Time de Migração** (Paradigm Advisor → Curator → Strategist → Designer → Inspector). Gera as specs do sistema novo. Saída em `_reversa_sdd/migration/`.
- `/reversa-reconstructor`: gera plano bottom-up para reimplementar o software a partir das specs do legado (uma tarefa por sessão).
