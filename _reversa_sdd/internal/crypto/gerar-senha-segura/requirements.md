# Caso de Uso: Gerar Senha Segura

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Visão Geral

A unit `internal/crypto/gerar-senha-segura` gera senha criptograficamente segura com mínimo e classes obrigatórias. A especificação é baseada nos artefatos Archaeologist/Detective/Architect e nos arquivos principais `internal/crypto/ethereum.go`, `internal/crypto/bitcoin.go`, `internal/crypto/solana.go`, `internal/crypto/checksum.go`, `internal/crypto/keystore.go`, `internal/crypto/password.go`, `internal/crypto/pools.go`, `internal/crypto/random.go`, `internal/crypto/validation.go`, `internal/crypto/generator.go`. 🟢

## Responsabilidades

- Manter o contrato operacional descrito para `internal/crypto/gerar-senha-segura` no legado. 🟢
- Validar entradas e preservar os limites documentados nas regras de negócio. 🟢
- Retornar erros ou resultados no formato consumido pelos módulos dependentes. 🟢
- Preservar os comportamentos de segurança, performance e observabilidade confirmados para a unit. 🟢

## Regras de Negócio

- Ethereum private key deve ter 32 bytes. 🟢
- KeyStore usa AES-128-CTR, versão 3 e KDF scrypt/PBKDF2. 🟢
- Senha segura tem mínimo 12 caracteres e quatro classes de caracteres. 🟢
- Ethereum private key deve ter 32 bytes; Solana aceita 64-byte private key ou 32-byte seed. 🟢
- KeyStore usa aes-128-ctr, versão 3, KDF scrypt ou pbkdf2. 🟢
- Senha gerada tem mínimo 12 caracteres e ao menos uma letra minúscula, maiúscula, número e especial. 🟢

## Requisitos Funcionais

| ID | Requisito | Prioridade | Critério de Aceite |
|----|-----------|-----------|-------------------|
| RF-01 | Reimplementar a responsabilidade principal da unit `internal/crypto/gerar-senha-segura`. | Must | Chamadas equivalentes às funções mapeadas em `internal/crypto/ethereum.go`, `internal/crypto/bitcoin.go`, `internal/crypto/solana.go`, `internal/crypto/checksum.go`, `internal/crypto/keystore.go`, `internal/crypto/password.go`, `internal/crypto/pools.go`, `internal/crypto/random.go`, `internal/crypto/validation.go`, `internal/crypto/generator.go` produzem resultados compatíveis. 🟢 |
| RF-02 | Preservar validações e limites confirmados. | Must | Entradas inválidas retornam erro ou rejeição conforme regras documentadas. 🟢 |
| RF-03 | Manter integração com dependências diretas. | Must | Módulos consumidores conseguem chamar a unit sem alterar contratos públicos. 🟢 |
| RF-04 | Preservar observabilidade e tratamento de erro aplicáveis. | Should | Logs, warnings ou erros estruturados mantêm semântica equivalente. 🟢 |
| RF-05 | Documentar lacunas sem introduzir comportamento novo implicitamente. | Should | Pontos 🟡/🔴 permanecem rastreados para decisão humana. 🟡 |

## Requisitos Não Funcionais

| Tipo | Requisito inferido | Evidência no código | Confiança |
|------|--------------------|---------------------|-----------|
| Manutenibilidade | A unit deve preservar separação modular em `internal/crypto`. | `internal/crypto/ethereum.go`, `internal/crypto/bitcoin.go`, `internal/crypto/solana.go`, `internal/crypto/checksum.go`, `internal/crypto/keystore.go`, `internal/crypto/password.go`, `internal/crypto/pools.go`, `internal/crypto/random.go`, `internal/crypto/validation.go`, `internal/crypto/generator.go` | 🟢 |
| Segurança | Dados sensíveis devem seguir regras de sanitização/backup quando aplicável. | `_reversa_sdd/domain.md`; `_reversa_sdd/data-dictionary.md` | 🟡 |
| Performance | Algoritmos e estruturas de concorrência/pooling devem manter características do legado quando presentes. | `_reversa_sdd/code-analysis.md` | 🟢 |
| Operabilidade | Erros e estados devem permanecer rastreáveis por mensagens ou structs de domínio. | `internal/crypto/ethereum.go`, `internal/crypto/bitcoin.go`, `internal/crypto/solana.go`, `internal/crypto/checksum.go`, `internal/crypto/keystore.go`, `internal/crypto/password.go`, `internal/crypto/pools.go`, `internal/crypto/random.go`, `internal/crypto/validation.go`, `internal/crypto/generator.go` | 🟢 |

> Inferido a partir do código. Validar com equipe de operações.

## Critérios de Aceitação

```gherkin
Dado uma entrada válida para `internal/crypto/gerar-senha-segura`
Quando a funcionalidade principal é executada
Então o resultado deve ser equivalente ao comportamento confirmado no legado

Dado uma entrada inválida ou configuração fora dos limites
Quando a funcionalidade é executada
Então a unit deve retornar erro, warning ou rejeição conforme regra documentada

Dado que um módulo consumidor depende de `internal/crypto/gerar-senha-segura`
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
| `internal/crypto/ethereum.go` | `GenerateAddressFromPrivateKey` | 🟢 |
| `internal/crypto/ethereum.go` | `OptimizedAddressGeneration` | 🟢 |
| `internal/crypto/checksum.go` | `ToChecksumAddress` | 🟢 |
| `internal/crypto/keystore.go` | `EncryptPrivateKeyWithKDF` | 🟢 |
| `internal/crypto/password.go` | `GenerateSecurePassword` | 🟢 |
