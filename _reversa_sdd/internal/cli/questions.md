# Módulo internal/cli, Perguntas Pendentes

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Status do Revisor

Triado em 2026-05-08T09:54:10Z. Este arquivo contém 20 pergunta(s) unitária(s) do Writer e 0 lacuna(s) crítica(s)/vermelha(s). Perguntas já escaladas e resolvidas no arquivo transversal: Q-CLI-003, Q-CLI-004, Q-CLI-006 e Q-CLI-011. Os demais itens permanecem como decisões de produto/UX ou dívida técnica não bloqueante para uma reimplementação fiel.


## Visão Geral

Este arquivo reúne perguntas de validação humana para pontos da CLI em que o contrato de produto, a documentação pública ou a intenção de implementação não estão totalmente confirmados pelo código lido. 🟢

## Perguntas Críticas

| ID | Pergunta | Contexto | Evidência | Impacto | Confiança |
|---|---|---|---|---|---:|
| Q-CLI-001 | A flag `--case-sensitive` deve alterar `GenerationCriteria` ou é apenas um contrato não implementado? | A flag é registrada, mas `getGenerationCriteria()` não lê `case-sensitive`. | `internal/cli/commands.go:85`; `internal/cli/commands.go:1352-1368` | Pode afetar matching de prefixo/sufixo, especialmente com `--checksum`. | 🟡 |
| Q-CLI-002 | As flags `--output` e `--format` devem gerar arquivo, JSON ou CSV em uma implementação fiel? | As flags são registradas, mas o fluxo de display lido imprime em stdout e não confirmou uso delas. | `internal/cli/commands.go:95-99`; `internal/cli/commands.go:1395-1505` | Pode haver divergência entre contrato CLI anunciado e comportamento real. | 🟡 |
| Q-CLI-003 | O README deve continuar anunciando benchmark `--pattern` ou o comando deve implementar essa flag? | O subcomando `benchmark` lido declara `attempts`, `duration` e `detailed`, sem `pattern`. | `internal/cli/commands.go:836-850`; `_reversa_sdd/domain.md:GAP-002` | Pode confundir usuários e testes de CLI. | 🟡 |
| Q-CLI-004 | O progress manager texto deve permanecer desabilitado ou deve ser corrigido? | O código declara que o progress manager está desabilitado para evitar deadlocks. | `internal/cli/commands.go:396-398`; `internal/cli/commands.go:700-701`; `_reversa_sdd/domain.md:BR-035` | Define se a reimplementação deve preservar a lacuna ou resolver comportamento de progresso. | 🟢 |
| Q-CLI-005 | O benchmark deve submeter `WorkItem` ao worker pool real ou preservar o TODO atual? | Os loops de benchmark criam `WorkItem`, mas não submetem trabalho ao pool. | `internal/cli/commands.go:1629-1630`; `internal/cli/commands.go:1753-1754`; `_reversa_sdd/domain.md:GAP-007` | Métricas de benchmark podem ser incompletas se o TODO for preservado. | 🟢 |
| Q-CLI-006 | A persistência Solana deve usar formato próprio real, KeyStore V3 compatível ou ser documentada como suporte parcial? | O fluxo agrupa Ethereum e Solana em keystore/formato específico, mas a análise anterior marcou simplificação/placeholder. | `internal/cli/commands.go:1979-2063`; `_reversa_sdd/domain.md:GAP-004` | Afeta segurança e restauração de carteiras Solana. | 🟡 |
| Q-CLI-007 | O fluxo single deve continuar imprimindo private key e mnemonic em stdout por padrão? | `displayWalletResult()` imprime private key sempre e mnemonic quando presente. | `internal/cli/commands.go:1395-1404` | Risco operacional se stdout for salvo, copiado ou logado externamente. | 🟢 |
| Q-CLI-008 | Em geração múltipla texto, falhas individuais devem continuar sem abortar todo o lote? | O loop imprime erro individual e continua para a próxima carteira. | `internal/cli/commands.go:703-713` | Define semântica de sucesso parcial e critérios de exit code. | 🟢 |
| Q-CLI-009 | Falha de keystore deve ser apenas warning ou deve tornar a geração uma falha? | Exibição de resultado trata erro de keystore como warning, mas helper de persistência retorna erro ao chamador. | `internal/cli/commands.go:1406-1415`; `internal/cli/commands.go:1453-1464`; `internal/cli/commands.go:2039-2049` | Afeta confiabilidade de backup e automações que dependem de exit code. | 🟢 |
| Q-CLI-010 | `quiet` deve ocultar segredos também no fluxo single texto? | Em múltiplos resultados, `QuietMode` oculta private key/mnemonic; no single texto, esses campos são sempre impressos. | `internal/cli/commands.go:1395-1404`; `internal/cli/commands.go:1438-1444` | Pode haver inconsistência de segurança/UX entre geração single e múltipla. | 🟡 |

## Perguntas de Produto e UX

| ID | Pergunta | Contexto | Impacto | Confiança |
|---|---|---|---|---:|
| Q-CLI-011 | O nome de produto exibido deve ser `Bloco-ETH`, `bloco-vgen`, `Bloco Vanity Generator` ou `bloco-wallet-generator`? | O código e artefatos anteriores registram nomes inconsistentes. | Afeta ajuda CLI, releases, Docker, documentação e onboarding. | 🟢 |
| Q-CLI-012 | A TUI deve ser padrão quando terminal suporta ou depender sempre de `--progress`? | A flag `--tui` tem default `true`, mas geração só usa TUI quando `--progress` está ativo. | Afeta expectativa de UX para usuários que não passam `--progress`. | 🟢 |
| Q-CLI-013 | Warnings de keystore devem ir para stdout ou stderr? | O código usa `fmt.Printf` em alguns warnings e `fmt.Fprintf(os.Stderr, ...)` em warnings de shutdown. | Afeta automações que parseiam stdout. | 🟡 |
| Q-CLI-014 | O subcomando `stats` deve respeitar `--network`? | `showStats()` usa `getGenerationCriteria()`, mas o comando `stats` declara apenas prefix/suffix/checksum como flags próprias. | Pode afetar análise para Bitcoin/Solana se a rede for relevante. | 🟡 |
| Q-CLI-015 | O benchmark deve aceitar rede, prefixo e sufixo customizados? | O benchmark usa critérios internos simples e não declara flags de pattern/rede. | Afeta capacidade de medir cenários reais de vanity generation. | 🟡 |

## Perguntas Técnicas

| ID | Pergunta | Contexto | Impacto | Confiança |
|---|---|---|---|---:|
| Q-CLI-016 | A validação de `--kdf-params` deve aceitar aliases de maiúsculas/minúsculas para campos JSON? | O código espera chaves específicas como `n`, `r`, `p`, `dklen`, `c`, `prf`. | Afeta compatibilidade com documentação e usuários. | 🟢 |
| Q-CLI-017 | Limites de scrypt em `validateScryptParams()` devem ser alinhados ao módulo KDF completo? | Data dictionary registrou `r` até 1024 e `p` até 16, enquanto a validação CLI aceita `r` e `p` até 256. | Pode haver divergência entre validação CLI e validação KDF. | 🟡 |
| Q-CLI-018 | `parseSecurityLevel()` deve falhar para valor desconhecido ou manter fallback para medium? | O código usa fallback silencioso para `SecurityLevelMedium`. | Afeta segurança e feedback ao usuário. | 🟢 |
| Q-CLI-019 | `generateMultipleWalletsTUI()` deve persistir resultados parciais quando o contexto é cancelado? | O fluxo mantém resultados em memória e pode fechar shutdown por contexto. | Afeta comportamento em interrupção de lotes longos. | 🟡 |
| Q-CLI-020 | O adapter `StatsManagerAdapter` deve expor mais métricas para TUI? | O adapter retorna eficiência, velocidade total, thread count e peak speed. | Pode limitar observabilidade de progresso. | 🟢 |

## Recomendações para Validação Humana

- Validar primeiro `Q-CLI-001`, `Q-CLI-002`, `Q-CLI-003` e `Q-CLI-010`, porque afetam contrato público da CLI. 🟡
- Decidir se o objetivo da reimplementação é preservar lacunas (`progress` texto e benchmark TODO) ou corrigi-las com comportamento novo. 🟢
- Revisar política de exposição de private key/mnemonic em stdout antes de usar a CLI em automações. 🟢
- Unificar naming de produto/binário antes de gerar documentação final de usuário. 🟢

## Status

Esta unit possui lacunas suficientes para exigir validação humana antes de uma reimplementação com mudanças de comportamento. Uma reimplementação estritamente fiel pode preservar os comportamentos confirmados, mas deve marcar como dívida as flags sem uso confirmado e o benchmark parcial. 🟢
