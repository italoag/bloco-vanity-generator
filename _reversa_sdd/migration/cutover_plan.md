---
schemaVersion: 1
generatedAt: 2026-05-08T11:18:00Z
reversa:
  version: "1.2.34"
kind: cutover_plan
producedBy: strategist
hash: "sha256:55ac50033575b10b6522b7e5a3997ebe286ff3c49b13c1a73910c6984aebcad2"
---

# Cutover Plan

> Plano base para a estratégia recomendada: Big Bang controlado com Parallel Run offline obrigatório. A estratégia final depende da escolha humana.

## Estratégia base

- **Estratégia**: Big Bang controlado.
- **Gate obrigatório**: Parallel Run offline antes de release público.
- **Produto alvo**: `bloco-vanity-generator`.
- **Binário compatível**: `bloco-vgen`.
- **Tipo de cutover**: release de binário/artefatos, não troca de tráfego runtime.
- **Janela sugerida**: 1 dia útil para release, validação e monitoramento manual.
- **Duração estimada da janela**: 4 a 8 horas após artefatos candidatos prontos.

## Pré-requisitos

| ID | Pré-requisito | Owner | Evidência esperada |
|---|---|---|---|
| PRE-001 | Estratégia escolhida pelo usuário e registrada. | Owner técnico | `migration_strategy.md` atualizado/aceito. |
| PRE-002 | Arquitetura alvo aprovada pelo Designer. | Designer / Owner técnico | `target_architecture.md` e topologia aprovada. |
| PRE-003 | Parity specs aprovadas pelo Inspector. | Inspector | `parity_specs.md` e features Gherkin. |
| PRE-004 | Testes automatizados verdes. | Mantenedor técnico | `go test`, race, lint, security scans. |
| PRE-005 | Golden tests CLI/stdout/stderr aprovados. | Usuário CLI validador | Matriz de flags e snapshots revisados. |
| PRE-006 | Testes de crypto por rede aprovados. | Revisor de segurança | Vetores Ethereum/Bitcoin/Solana aprovados. |
| PRE-007 | Testes de não vazamento aprovados. | Revisor de segurança | Logs/stdout/arquivos sem segredo indevido. |
| PRE-008 | Benchmarks comparáveis ou melhores que legado. | Engenheiro de performance | Relatório de throughput por cenário. |
| PRE-009 | Artefatos Docker/release gerados e verificados. | Mantenedor/release | Checksums, binários e imagem disponíveis. |
| PRE-010 | Plano de rollback comunicado. | Owner técnico | Release legado/binário anterior disponível. |

## Go/No-Go

### Critérios de Go

- Todos os pré-requisitos `PRE-*` estão cumpridos.
- Nenhum risco CRITICAL aberto sem mitigação aceita.
- Nenhum teste de redaction acusa private key, mnemonic, password ou salt em logs.
- Solana não gera `.key` bruto.
- Validação por rede funciona para Ethereum, Bitcoin e Solana.
- `Wallet.IsValid()` por rede passa nos cenários de paridade.
- `bloco-vgen` executa comandos principais esperados.
- Performance está dentro do limite aceito pelo owner técnico.

### Critérios de No-Go

- Qualquer divergência criptográfica não explicada.
- Qualquer vazamento de segredo em log ou arquivo não autorizado.
- Deadlock, race ou timeout em worker/progress/TUI.
- Artefatos de release sem checksum verificável.
- Falha de persistência/recovery de keystore/mnemonic em cenários principais.

## Plano de execução da janela

| Passo | Ação | Owner | Duração estimada | Go/No-Go local |
|---|---|---|---:|---|
| CUT-001 | Congelar branch/release candidate. | Owner técnico | 15 min | Sem commits pendentes críticos. |
| CUT-002 | Rodar suíte completa local/CI. | Mantenedor técnico | 60-120 min | Testes, race, lint, scans verdes. |
| CUT-003 | Executar Parallel Run offline contra cenários definidos pelo Inspector. | Inspector | 60-120 min | Paridade aceita ou divergência intencional documentada. |
| CUT-004 | Rodar benchmarks comparativos. | Engenheiro de performance | 30-60 min | Throughput dentro do limiar aceito. |
| CUT-005 | Validar segurança de artefatos locais. | Revisor de segurança | 30-60 min | Sem segredo indevido; permissões corretas. |
| CUT-006 | Gerar binários, checksums e imagem Docker. | Mantenedor/release | 30-60 min | Checksums e imagem verificados. |
| CUT-007 | Publicar release/tag. | Mantenedor/release | 15-30 min | Release visível e assets baixáveis. |
| CUT-008 | Smoke test pós-release com binário baixado. | Usuário CLI validador | 30-60 min | Comandos principais funcionam. |
| CUT-009 | Atualizar documentação de instalação/uso. | Mantenedor/release | 30-60 min | Docs refletem comportamento real. |
| CUT-010 | Encerrar janela ou acionar rollback. | Owner técnico | 15 min | Decisão final registrada. |

## Smoke tests mínimos pós-release

| ID | Cenário | Resultado esperado |
|---|---|---|
| SMOKE-001 | `bloco-vgen version` | Exibe versão do release. |
| SMOKE-002 | Geração Ethereum com prefixo simples e `--count=1` | Endereço válido, warning de segredo em stdout quando aplicável, keystore salvo se habilitado. |
| SMOKE-003 | Geração Bitcoin com mnemonic | Endereço P2PKH válido e mnemonic salvo para backup. |
| SMOKE-004 | Geração Solana | Endereço Base58 válido e persistência segura sem `.key` bruto. |
| SMOKE-005 | `--quiet` em geração múltipla | Não exibe private key/mnemonic por carteira. |
| SMOKE-006 | TUI indisponível ou desabilitada | Fallback texto funciona sem deadlock. |
| SMOKE-007 | Logging habilitado | Logs contêm eventos operacionais, sem segredo. |
| SMOKE-008 | Docker image | Container executa comando de versão/ajuda. |

## Plano de rollback

### Gatilhos

- Falha CRITICAL em crypto, segurança ou persistência.
- Regresso grave de CLI impedindo uso principal.
- Vazamento de segredo em release público.
- Binários ou imagem Docker inválidos.

### Ações

| Passo | Ação | Owner | Tempo alvo |
|---|---|---|---:|
| RB-001 | Marcar release alvo como problemático nas notas. | Mantenedor/release | 15 min |
| RB-002 | Reapontar documentação para última versão estável. | Mantenedor/release | 30 min |
| RB-003 | Remover ou substituir assets se houver risco de segredo/artefato inválido. | Revisor de segurança / release | 30-60 min |
| RB-004 | Publicar hotfix ou nova tag com correção. | Mantenedor técnico | conforme severidade |
| RB-005 | Registrar post-mortem e atualizar risk register. | Owner técnico | 1 dia útil |

## Plano de comunicação

| Público | Mensagem | Canal |
|---|---|---|
| Usuários CLI | Nova versão `bloco-vanity-generator` com binário compatível `bloco-vgen`; mudanças intencionais de segurança documentadas. | README/release notes |
| Mantenedores | Checklist de release, rollback e riscos residuais. | PR/release issue |
| Segurança | Confirmação de testes sem vazamento e persistência Solana segura. | Relatório/review |

## Critérios de sucesso pós-cutover

- Release disponível com checksums e imagem Docker.
- Comandos principais funcionam em Linux/macOS amd64/arm64.
- Nenhum vazamento de segredo em smoke tests.
- Nenhum deadlock em TUI/progress textual.
- Benchmarks dentro do limiar aceito.
- Documentação não anuncia flags inexistentes.

## Observações

Como não há banco de dados nem serviço remoto, não existe migração de dados centralizada nem roteamento gradual de tráfego. O rollback é operacional: manter/retornar para binário anterior e release anterior. Artefatos locais de usuário não devem ser sobrescritos automaticamente.
