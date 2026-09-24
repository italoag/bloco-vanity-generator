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
| Q-CLI-011 | O nome de produto exibido deve ser `Bloco Vanity Generator`, `bloco-vgen`, `Bloco Vanity Generator` ou `bloco-wallet-generator`? | O código e artefatos anteriores registram nomes inconsistentes. | Afeta ajuda CLI, releases, Docker, documentação e onboarding. | 🟢 |
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

## Respostas Consolidadas — 2026-05-14

### Perguntas Críticas

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-CLI-001 | `--case-sensitive` deve alterar `GenerationCriteria`; não é apenas contrato morto. | Manter `GenerationCriteria.CaseSensitive`, ler a flag no parser CLI e validar que ela só é aceita com `--checksum` em Ethereum. Para Ethereum EIP-55, comparar contra o endereço checksum calculado. | Respondida |
| Q-CLI-002 | Para fidelidade ao código atual, `--output` e `--format` ainda não geram arquivo/JSON/CSV; para produto alvo, flags públicas devem funcionar ou ser removidas. | Melhor abordagem: implementar `--output` e `--format` de ponta a ponta, com `text`, `json` e `csv`, garantindo tratamento seguro de segredos. Enquanto não implementado, README/help devem indicar a limitação ou as flags devem ser ocultadas/deprecadas. | Respondida |
| Q-CLI-003 | O README não deve anunciar `benchmark --pattern` enquanto o comando não implementar essa flag. | Tratar o README legado como desatualizado quando divergir do Cobra/código. Reintroduzir `--pattern` somente com implementação, testes e documentação correspondentes. | Respondida |
| Q-CLI-004 | O progress manager textual deve ser corrigido e reativado como fallback. | Não preservar deadlock como comportamento alvo. Redesenhar progress textual com ciclo de vida claro, cancelamento por contexto, fechamento seguro de canais e testes com `-race`. | Respondida |
| Q-CLI-005 | O benchmark deve submeter trabalho real ao worker pool; preservar o TODO não é comportamento alvo. | Implementar benchmark sobre o mesmo motor de geração usado pela CLI e coletar métricas reais. Até lá, documentar o benchmark como limitado e não usá-lo como prova de performance. | Respondida |
| Q-CLI-006 | Persistência Solana deve usar formato seguro/criptografado ou ser explicitamente limitada; não deve salvar `.key` bruto por padrão. | Substituir arquivo raw por formato criptografado/seguro. Se a compatibilidade Solana completa ainda não existir, falhar com mensagem clara ou documentar suporte parcial, evitando falsa sensação de backup seguro. | Respondida |
| Q-CLI-007 | O fluxo single pode continuar exibindo private key/mnemonic por padrão por compatibilidade, desde que haja aviso de segurança claro. | Decisão humana anterior preservou a exibição por compatibilidade. A documentação deve tratar stdout como sensível, e o usuário deve ter modo explícito para ocultar segredos em automações. | Respondida |
| Q-CLI-008 | Falhas individuais em geração múltipla devem continuar sem abortar todo o lote quando for possível continuar. | Manter semântica de sucesso parcial: registrar erro individual, continuar próximas carteiras e exibir resumo final. Exit code deve distinguir falha total de sucesso parcial quando houver modo estruturado/automação. | Respondida |
| Q-CLI-009 | Falha de keystore não deve ser silenciosa; como padrão atual, pode ser warning quando a carteira foi gerada e o segredo foi exibido ao usuário. | Melhor abordagem: manter warning no modo interativo legado, mas adicionar/usar modo estrito para automação quando keystore estiver habilitado e a persistência for requisito. Em quiet/automação, falha de backup deve resultar em erro claro para evitar perda de segredo. | Respondida |
| Q-CLI-010 | `quiet` deve ocultar segredos também no fluxo single texto. | Corrigir a inconsistência: `--quiet` deve suprimir private key/mnemonic em qualquer fluxo textual. Se o usuário precisar dos segredos em stdout, deve usar modo normal ou flag explícita futura. | Respondida |

### Perguntas de Produto e UX

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-CLI-011 | Nome canônico: produto/repo/docs como `bloco-vanity-generator`; binário compatível como `bloco-vgen`. | Help, README, CI, Docker e releases devem convergir para `bloco-vgen` como executável. Nome textual de produto pode usar `Bloco Vanity Generator`; evitar `Bloco Vanity Generator` como nome principal porque o produto é multirede. | Respondida |
| Q-CLI-012 | A TUI não deve abrir apenas porque o terminal suporta; deve depender de intenção de progresso/interatividade. | Manter `--tui` como preferência/capacidade e usar TUI quando `--progress` estiver ativo e o ambiente suportar. Sem `--progress`, manter saída textual simples e previsível. | Respondida |
| Q-CLI-013 | Warnings de keystore devem ir para stderr. | stdout deve permanecer reservado para resultado consumível pelo usuário/automação. Warnings, diagnósticos e falhas de persistência devem ir para stderr ou log operacional seguro. | Respondida |
| Q-CLI-014 | `stats` deve respeitar `--network` no alvo multirede. | Adicionar flags/reuso de flags globais para rede e calcular dificuldade conforme alfabeto/formato da rede. Enquanto não implementado, documentar `stats` como estimativa Ethereum/hex. | Respondida |
| Q-CLI-015 | O benchmark deve aceitar rede, prefixo e sufixo customizados para medir cenários reais. | Evoluir benchmark para aceitar critérios equivalentes aos da geração (`--network`, `--prefix`, `--suffix`, `--checksum`, threads/duração/tentativas), evitando métricas artificiais que não representam vanity real. | Respondida |

### Perguntas Técnicas

| ID | Resposta | Abordagem recomendada | Status |
|---|---|---|---|
| Q-CLI-016 | `--kdf-params` deve ser estrito com chaves canônicas, não aceitar aliases silenciosos. | Aceitar `n`, `r`, `p`, `dklen`, `c`, `prf`; rejeitar chaves desconhecidas com erro claro. Isso evita typos silenciosos em parâmetros de segurança. | Respondida |
| Q-CLI-017 | Limites de scrypt devem ser alinhados ao módulo KDF completo. | A CLI não deve manter ranges próprios divergentes; deve delegar validação ao serviço KDF central e apenas formatar mensagens. | Respondida |
| Q-CLI-018 | Valor desconhecido de security level deve falhar na validação final, não cair silenciosamente para medium como contrato. | Manter fallback interno apenas como defesa inatingível após `Config.Validate()`. Para entrada de usuário/ambiente inválida, retornar erro ou warning explícito com default documentado. | Respondida |
| Q-CLI-019 | Resultados parciais devem ser persistidos conforme são gerados, inclusive em lote/TUI, para sobreviver a cancelamento. | Persistir cada carteira imediatamente após geração bem-sucedida quando keystore estiver habilitado. Em cancelamento, mostrar resumo de resultados persistidos, pendentes e falhas. | Respondida |
| Q-CLI-020 | O adapter deve expor mais métricas quando a TUI precisar de observabilidade real. | Expandir gradualmente para tentativas totais, velocidade média/pico, erros, workers ativos, duração e ETA. Manter interface mínima apenas enquanto a TUI não renderizar esses dados. | Respondida |

## Recomendações para Validação Humana

- Validar primeiro `Q-CLI-001`, `Q-CLI-002`, `Q-CLI-003` e `Q-CLI-010`, porque afetam contrato público da CLI. 🟡
- Decidir se o objetivo da reimplementação é preservar lacunas (`progress` texto e benchmark TODO) ou corrigi-las com comportamento novo. 🟢
- Revisar política de exposição de private key/mnemonic em stdout antes de usar a CLI em automações. 🟢
- Unificar naming de produto/binário antes de gerar documentação final de usuário. 🟢

## Status

Esta unit possui lacunas suficientes para exigir validação humana antes de uma reimplementação com mudanças de comportamento. Uma reimplementação estritamente fiel pode preservar os comportamentos confirmados, mas deve marcar como dívida as flags sem uso confirmado e o benchmark parcial. 🟢
