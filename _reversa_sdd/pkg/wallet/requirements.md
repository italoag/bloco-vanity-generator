# Módulo pkg/wallet

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

A unit `pkg/wallet` define entidades de domínio de carteira, critérios, resultados, estatísticas e logger legado. A especificação é baseada nos artefatos Archaeologist/Detective/Architect e nos arquivos principais `pkg/wallet/types.go`, `pkg/wallet/logger.go`. 🟢

## Responsabilidades

- Manter o contrato operacional descrito para `pkg/wallet` no legado. 🟢
- Validar entradas e preservar os limites documentados nas regras de negócio. 🟢
- Retornar erros ou resultados no formato consumido pelos módulos dependentes. 🟢
- Preservar os comportamentos de segurança, performance e observabilidade confirmados para a unit. 🟢

## Regras de Negócio

- GenerationCriteria limita padrão total a 20 caracteres. 🟢
- No legado, prefixo e sufixo são validados como hexadecimais. 🟢
- Por decisão humana, a reimplementação deve validar prefixo/sufixo por `Network`: Ethereum hexadecimal/EIP-55, Bitcoin Base58/bech32 quando aplicável e Solana Base58. 🟢
- No legado, `Wallet.IsValid()` é centrado em formatos Ethereum. 🟢
- Por decisão humana, `Wallet.IsValid()` deve evoluir para validação por `Network`. 🟢
- `WalletLogger` legado deve migrar para logging seguro e não persistir private key em claro. 🟢
- GenerationCriteria limita padrão total a 20 caracteres. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Reimplementar a responsabilidade principal da unit `pkg/wallet`. | Must | Chamadas equivalentes às funções mapeadas em `pkg/wallet/types.go`, `pkg/wallet/logger.go` produzem resultados compatíveis. 🟢 |
| RF-02 | Preservar validações e limites confirmados. | Must | Entradas inválidas retornam erro ou rejeição conforme regras documentadas. 🟢 |
| RF-03 | Manter integração com dependências diretas. | Must | Módulos consumidores conseguem chamar a unit sem alterar contratos públicos. 🟢 |
| RF-04 | Preservar observabilidade e tratamento de erro aplicáveis. | Should | Logs, warnings ou erros estruturados mantêm semântica equivalente. 🟢 |
| RF-05 | Documentar lacunas sem introduzir comportamento novo implicitamente. | Should | Pontos 🟡/🔴 permanecem rastreados para decisão humana. 🟡 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Manutenibilidade | A unit deve preservar separação modular em `pkg/wallet`. | `pkg/wallet/types.go`, `pkg/wallet/logger.go` | 🟢 |
| Segurança | Dados sensíveis devem seguir logging seguro; `WalletLogger` legado deve ser migrado/sanitizado para não gravar private key em claro. | `_reversa_sdd/domain.md`; resposta do Revisor | 🟢 |
| Performance | Algoritmos e estruturas de concorrência/pooling devem manter características do legado quando presentes. | `_reversa_sdd/code-analysis.md` | 🟢 |
| Operabilidade | Erros e estados devem permanecer rastreáveis por mensagens ou structs de domínio. | `pkg/wallet/types.go`, `pkg/wallet/logger.go` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado uma entrada válida para `pkg/wallet`
Quando a funcionalidade principal é executada
Então o resultado deve ser equivalente ao comportamento confirmado no legado

Dado uma entrada inválida ou configuração fora dos limites
Quando a funcionalidade é executada
Então a unit deve retornar erro, warning ou rejeição conforme regra documentada

Dado que um módulo consumidor depende de `pkg/wallet`
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
| `pkg/wallet/types.go` | `Validate` | 🟢 |
| `pkg/wallet/types.go` | `Update` | 🟢 |
| `pkg/wallet/logger.go` | `LogWallet` | 🟢 |
