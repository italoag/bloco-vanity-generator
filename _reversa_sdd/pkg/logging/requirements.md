# Módulo pkg/logging

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

A unit `pkg/logging` fornece logging seguro com sanitização, formatos, buffering assíncrono e rotação. A especificação é baseada nos artefatos Archaeologist/Detective/Architect e nos arquivos principais `pkg/logging/secure_logger.go`, `pkg/logging/types.go`. 🟢

## Responsabilidades

- Manter o contrato operacional descrito para `pkg/logging` no legado. 🟢
- Validar entradas e preservar os limites documentados nas regras de negócio. 🟢
- Retornar erros ou resultados no formato consumido pelos módulos dependentes. 🟢
- Preservar os comportamentos de segurança, performance e observabilidade confirmados para a unit. 🟢

## Regras de Negócio

- Parâmetros sensíveis são filtrados por whitelist. 🟢
- Salt e segredos não devem aparecer nos logs. 🟢
- Parâmetros sensíveis são filtrados por whitelist; salt e segredos não devem aparecer nos logs. 🟢
- Por decisão humana, o `WalletLogger` legado deve migrar para logging seguro/sanitizado e não gravar private key em claro. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Reimplementar a responsabilidade principal da unit `pkg/logging`. | Must | Chamadas equivalentes às funções mapeadas em `pkg/logging/secure_logger.go`, `pkg/logging/types.go` produzem resultados compatíveis. 🟢 |
| RF-02 | Preservar validações e limites confirmados. | Must | Entradas inválidas retornam erro ou rejeição conforme regras documentadas. 🟢 |
| RF-03 | Manter integração com dependências diretas. | Must | Módulos consumidores conseguem chamar a unit sem alterar contratos públicos. 🟢 |
| RF-04 | Preservar observabilidade e tratamento de erro aplicáveis. | Should | Logs, warnings ou erros estruturados mantêm semântica equivalente. 🟢 |
| RF-05 | Documentar lacunas sem introduzir comportamento novo implicitamente. | Should | Pontos 🟡/🔴 permanecem rastreados para decisão humana. 🟡 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Manutenibilidade | A unit deve preservar separação modular em `pkg/logging`. | `pkg/logging/secure_logger.go`, `pkg/logging/types.go` | 🟢 |
| Segurança | Dados sensíveis devem seguir sanitização estrita; logging legado com private key deve migrar para logging seguro. | `_reversa_sdd/domain.md`; resposta do Revisor | 🟢 |
| Performance | Algoritmos e estruturas de concorrência/pooling devem manter características do legado quando presentes. | `_reversa_sdd/code-analysis.md` | 🟢 |
| Operabilidade | Erros e estados devem permanecer rastreáveis por mensagens ou structs de domínio. | `pkg/logging/secure_logger.go`, `pkg/logging/types.go` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado uma entrada válida para `pkg/logging`
Quando a funcionalidade principal é executada
Então o resultado deve ser equivalente ao comportamento confirmado no legado

Dado uma entrada inválida ou configuração fora dos limites
Quando a funcionalidade é executada
Então a unit deve retornar erro, warning ou rejeição conforme regra documentada

Dado que um módulo consumidor depende de `pkg/logging`
Quando a unit é reimplementada
Então o contrato usado pelo consumidor deve permanecer compatível
```

## Prioridade (MoSCoW)

| Requisito | MoSCoW | Justificativa |
|-----------|--------|---------------|
| Contrato funcional principal | Must | Caminho crítico para reimplementar o comportamento legado. 🟢 |
| Validações e limites | Must | Evitam comportamento divergente e falhas de segurança. 🟢 |
| Observabilidade e mensagens | Should | Importante para operação, mas pode ter fallback. 🟢 |
| Lacunas documentadas | Should | Necessárias para evitar decisões implícitas. 🟡 |

> Prioridade inferida por frequência de chamada e posição na cadeia de dependências.

## Rastreabilidade de Código

| Arquivo | Função / Classe | Cobertura |
|---------|-----------------|-----------|
| `pkg/logging/secure_logger.go` | `NewSecureLogger` | 🟢 |
| `pkg/logging/secure_logger.go` | `LogOperationStart` | 🟢 |
| `pkg/logging/secure_logger.go` | `LogError` | 🟢 |
