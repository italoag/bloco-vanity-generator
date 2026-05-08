# Caso de Uso: Analisar Compatibilidade KDF, Tarefas de Implementação

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Pré-requisitos

- [ ] Dependências da unit listadas em `design.md` estão disponíveis. 🟢
- [ ] Entidades de domínio documentadas em `_reversa_sdd/data-dictionary.md` foram recriadas ou adaptadas. 🟢
- [ ] Regras de negócio documentadas em `requirements.md` foram revisadas antes da implementação. 🟢

## Tarefas

> Cada tarefa referencia o arquivo do legado de onde o comportamento foi extraído.


- [ ] T-01, Implementar o contrato principal de `internal/crypto/kdf/analisar-compatibilidade`.
  - Origem no legado: `internal/crypto/kdf/service.go`
  - Critério de pronto: a unit executa o fluxo principal descrito em `design.md`.
  - Confiança: 🟢
- [ ] T-02, Reimplementar `DeriveKey`.
  - Origem no legado: `internal/crypto/kdf/service.go`
  - Critério de pronto: assinatura, entradas, retornos e erros seguem o comportamento mapeado.
  - Confiança: 🟢
- [ ] T-03, Reimplementar `ValidateParams`.
  - Origem no legado: `internal/crypto/kdf/service.go`
  - Critério de pronto: assinatura, entradas, retornos e erros seguem o comportamento mapeado.
  - Confiança: 🟢
- [ ] T-04, Reimplementar `AnalyzeKeystore`.
  - Origem no legado: `internal/crypto/kdf/analyzer.go`
  - Critério de pronto: assinatura, entradas, retornos e erros seguem o comportamento mapeado.
  - Confiança: 🟢
- [ ] T-05, Preservar regras de negócio e lacunas documentadas.
  - Origem no legado: `.reversa/context/modules.json`; `_reversa_sdd/domain.md`
  - Critério de pronto: testes cobrem regras confirmadas e pontos 🟡/🔴 permanecem explícitos.
  - Confiança: 🟡

## Tarefas de Teste

- [ ] TT-01, Teste do happy path do fluxo principal.  
  - Critério de pronto: entrada válida retorna resultado equivalente ao legado. 🟢
- [ ] TT-02, Teste de entrada inválida ou configuração fora dos limites.  
  - Critério de pronto: erro/retorno segue contrato documentado. 🟢
- [ ] TT-03, Teste de integração com módulo consumidor.  
  - Critério de pronto: chamada a partir do fluxo arquitetural principal permanece compatível. 🟡

## Tarefas de Migração de Dados

Não aplicável, salvo quando a unit explicitamente cria artefatos locais documentados em `requirements.md`. 🟢

## Ordem Sugerida

1. Recriar tipos e contratos públicos usados pelos consumidores. 🟢
2. Implementar funções principais com validações e erros equivalentes. 🟢
3. Integrar dependências diretas e preservar observabilidade. 🟢
4. Executar testes de happy path, erro e integração. 🟢

## Lacunas Pendentes (🔴)

- 🟡 Validar pontos marcados em `questions.md` quando existir arquivo opcional da unit.
- 🟡 Não corrigir lacunas conhecidas do legado sem decisão explícita, pois isso altera paridade comportamental.
