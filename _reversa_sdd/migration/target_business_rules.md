---
schemaVersion: 1
generatedAt: 2026-05-08T10:36:00Z
reversa:
  version: "1.2.34"
kind: target_business_rules
producedBy: curator
hash: "sha256:9e7c49c9387d80b5daadf6afca34522945dcb28e4f509d560721c991f71e85d8"
---

# Target Business Rules

> Catálogo das regras de negócio do legado com decisão de migração: MIGRAR, DESCARTAR ou DECISÃO HUMANA.  
> Cada item rastreia para a origem em `_reversa_sdd/` e respeita o `paradigm_decision.md`.

## Resumo

- Total de regras consolidadas analisadas: 41
- MIGRAR: 30
- DESCARTAR: 9 (detalhe em `discard_log.md`)
- DECISÃO HUMANA: 2
- Apetite aplicado: `transformational`

## Critério de curadoria aplicado

A decisão transformacional permite reorganizar internamente o sistema em Go idiomático, mas não autoriza mudar o produto para backend, banco, fila ou serviço remoto. Portanto:

- Regras de comportamento externo confirmado foram migradas.
- Mecanismos acidentais do legado foram descartados quando decisões humanas já os classificaram como dívida.
- Itens de segurança/operabilidade sem decisão explícita foram mantidos como DECISÃO HUMANA.

## Regras MIGRAR

### BR-MIGRAR-001 — Produto CLI local sem backend obrigatório

- **Origem**: `_reversa_sdd/architecture.md` § Sumário executivo; `_reversa_sdd/migration/migration_brief.md` § Escopo declarado
- **Confiança original**: 🟢
- **Descrição**: O alvo continua sendo uma aplicação CLI local para geração de carteiras vanity, sem backend HTTP/RPC, banco de dados, filas ou serviços remotos obrigatórios.
- **Justificativa de migração**: É o formato operacional confirmado do produto e também restrição explícita do brief.
- **Compatibilidade com paradigma alvo**: Compatível com Go idiomático, mantendo execução local síncrona e concorrência interna.

### BR-MIGRAR-002 — Nome canônico do produto e binário compatível

- **Origem**: `_reversa_sdd/questions.md` Pergunta 1; `_reversa_sdd/gaps.md` GAP-RV-001
- **Confiança original**: 🟢
- **Descrição**: Produto/repositório/documentação devem usar `bloco-vanity-generator`; binário compatível deve ser `bloco-vgen`.
- **Justificativa de migração**: Decisão humana consolidada pelo Revisor.
- **Compatibilidade com paradigma alvo**: Não afeta paradigma; deve ser refletido em CLI, Docker, release e documentação.

### BR-MIGRAR-003 — Inicialização com configuração validada

- **Origem**: `_reversa_sdd/cmd/bloco-vgen/inicializacao-da-cli/requirements.md` § Regras de Negócio; `_reversa_sdd/internal/config/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Configuração padrão deve ser criada antes de variáveis de ambiente/flags, e configuração inválida deve impedir a inicialização da CLI.
- **Justificativa de migração**: Protege execução contra parâmetros inválidos antes de acionar crypto/worker.
- **Compatibilidade com paradigma alvo**: Pode ser reimplementada com um pacote de config mais idiomático, preservando comportamento.

### BR-MIGRAR-004 — Cancelamento gracioso por contexto

- **Origem**: `_reversa_sdd/cmd/bloco-vgen/requirements.md`; `_reversa_sdd/domain.md` BR-036
- **Confiança original**: 🟢
- **Descrição**: `os.Interrupt` e `SIGTERM` devem cancelar o contexto de execução para desligamento gracioso.
- **Justificativa de migração**: Essencial para geração concorrente longa e segura.
- **Compatibilidade com paradigma alvo**: Go idiomático via `context.Context`.

### BR-MIGRAR-005 — Erros estruturados e debug controlado

- **Origem**: `_reversa_sdd/cmd/bloco-vgen/requirements.md`; `_reversa_sdd/pkg/errors/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Erros estruturados devem preservar tipo/contexto; stack trace só deve aparecer com `BLOCO_DEBUG`.
- **Justificativa de migração**: Mantém UX operacional e evita exposição indevida em execução normal.
- **Compatibilidade com paradigma alvo**: Pode usar erros Go idiomáticos com wrapping e tipos categorizados.

### BR-MIGRAR-006 — Flags e overrides explícitos

- **Origem**: `_reversa_sdd/internal/cli/requirements.md` § Regras de Negócio; `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Flags devem sobrescrever configuração apenas quando explicitamente alteradas, incluindo threads, keystore, diretório, KDF, parâmetros KDF e segurança.
- **Justificativa de migração**: Evita alterações implícitas de comportamento e preserva controle do operador.
- **Compatibilidade com paradigma alvo**: Pode ser isolada em camada de parsing CLI e mapeamento para configuração.

### BR-MIGRAR-007 — Threads configuráveis e limites operacionais

- **Origem**: `_reversa_sdd/internal/cli/requirements.md`; `_reversa_sdd/internal/config/requirements.md`; `_reversa_sdd/internal/worker/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: `--threads=0` autodetecta CPU; configuração deve respeitar `1..128`; dentro do pool, valores `<=0` viram `1`.
- **Justificativa de migração**: Mantém controle de performance e evita configuração inválida.
- **Compatibilidade com paradigma alvo**: Deve ser unificada para reduzir inconsistências entre config e worker.

### BR-MIGRAR-008 — Critérios de geração vanity

- **Origem**: `_reversa_sdd/domain.md` BR-001..BR-006; `_reversa_sdd/pkg/wallet/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Usuário pode buscar por prefixo, sufixo ou ambos; padrão total deve continuar limitado a 20 caracteres; `MaxAttempts` não pode ser negativo.
- **Justificativa de migração**: Regras centrais do domínio de vanity address.
- **Compatibilidade com paradigma alvo**: Expressar como validações puras por rede.

### BR-MIGRAR-009 — Validação vanity específica por rede

- **Origem**: `_reversa_sdd/questions.md` Pergunta 3; `_reversa_sdd/gaps.md` GAP-RV-003; `_reversa_sdd/domain.md` BR-002
- **Confiança original**: 🟢
- **Descrição**: Ethereum deve validar hexadecimal/EIP-55; Bitcoin deve validar Base58/bech32 quando aplicável; Solana deve validar Base58.
- **Justificativa de migração**: Decisão humana corrige limitação do legado sem perder intenção de negócio.
- **Compatibilidade com paradigma alvo**: Deve virar estratégia ou função por rede, não regra global hex-only.

### BR-MIGRAR-010 — `Wallet.IsValid()` por rede

- **Origem**: `_reversa_sdd/questions.md` Pergunta 4; `_reversa_sdd/gaps.md` GAP-RV-004; `_reversa_sdd/pkg/wallet/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Validação de carteira deve considerar `Network` e formatos Ethereum, Bitcoin e Solana.
- **Justificativa de migração**: Decisão humana consolidada; necessária para multirede real.
- **Compatibilidade com paradigma alvo**: Pode ser interface/registry de validadores ou funções por rede.

### BR-MIGRAR-011 — Geração single e múltipla

- **Origem**: `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md` RF-GCV-06..RF-GCV-07
- **Confiança original**: 🟢
- **Descrição**: `count == 1` executa fluxo single; valores diferentes de `1` executam fluxo múltiplo.
- **Justificativa de migração**: Comportamento externo confirmado da CLI.
- **Compatibilidade com paradigma alvo**: Pode ser separado em comandos/serviços internos mantendo contrato de flags.

### BR-MIGRAR-012 — Geração múltipla resiliente a erro individual

- **Origem**: `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md` § Regras de Negócio
- **Confiança original**: 🟢
- **Descrição**: Em geração múltipla texto, erro em uma carteira não deve abortar todo o lote.
- **Justificativa de migração**: Preserva sucesso parcial e UX operacional.
- **Compatibilidade com paradigma alvo**: Modelar resultado por item com resumo agregado.

### BR-MIGRAR-013 — Worker pool concorrente com contexto e estatísticas

- **Origem**: `_reversa_sdd/code-analysis.md` § Loop concorrente de geração; `_reversa_sdd/internal/worker/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Geração deve usar concorrência Go, contexto cancelável, estatísticas por worker e primeiro resultado vencedor.
- **Justificativa de migração**: Mecanismo central de performance do produto.
- **Compatibilidade com paradigma alvo**: Expressar com goroutines/channels/context de forma mais idiomática e testável.

### BR-MIGRAR-014 — Ethereum: secp256k1, Keccak e EIP-55

- **Origem**: `_reversa_sdd/code-analysis.md` § Geração de endereço Ethereum; `_reversa_sdd/domain.md` BR-013 e BR-005
- **Confiança original**: 🟢
- **Descrição**: Ethereum usa private key de 32 bytes, secp256k1, endereço por Keccak e checksum EIP-55.
- **Justificativa de migração**: Regra criptográfica essencial.
- **Compatibilidade com paradigma alvo**: Deve ter testes determinísticos e vetores de validação.

### BR-MIGRAR-015 — Bitcoin: endereço P2PKH e mnemonic para backup

- **Origem**: `_reversa_sdd/domain.md` BR-014 e BR-017; `_reversa_sdd/internal/cli/salvar-keystore/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Bitcoin gera endereço P2PKH com chave pública comprimida e salva somente mnemonic para backup; carteira Bitcoin sem mnemonic deve erro de backup.
- **Justificativa de migração**: Comportamento confirmado e regra de persistência específica por rede.
- **Compatibilidade com paradigma alvo**: Persistência por rede deve ser explícita.

### BR-MIGRAR-016 — Solana: Ed25519 e persistência segura

- **Origem**: `_reversa_sdd/domain.md` BR-015 e BR-018; `_reversa_sdd/questions.md` Pergunta 5
- **Confiança original**: 🟢
- **Descrição**: Solana usa Ed25519 e endereço Base58; persistência alvo deve ser criptografada/segura e não salvar `.key` bruto.
- **Justificativa de migração**: Combina comportamento confirmado e decisão humana de segurança.
- **Compatibilidade com paradigma alvo**: Requer desenho de formato seguro no Designer e teste de paridade no Inspector.

### BR-MIGRAR-017 — KeyStore V3 Ethereum

- **Origem**: `_reversa_sdd/internal/crypto/gerar-keystore-v3/requirements.md`; `_reversa_sdd/code-analysis.md` § KeyStore V3
- **Confiança original**: 🟢
- **Descrição**: KeyStore usa versão 3, AES-128-CTR, KDF `scrypt`/PBKDF2, salt/IV aleatórios e MAC Keccak.
- **Justificativa de migração**: Contrato de interoperabilidade com ecossistema Ethereum.
- **Compatibilidade com paradigma alvo**: Expor via serviço de alto nível, não função interna de baixo nível.

### BR-MIGRAR-018 — Senha segura de keystore

- **Origem**: `_reversa_sdd/internal/crypto/gerar-senha-segura/requirements.md`; `_reversa_sdd/domain.md` BR-024
- **Confiança original**: 🟢
- **Descrição**: Senha gerada deve ter no mínimo 12 caracteres e cobrir minúscula, maiúscula, número e especial.
- **Justificativa de migração**: Requisito de segurança confirmado.
- **Compatibilidade com paradigma alvo**: Gerador deve continuar usando fonte criptográfica segura.

### BR-MIGRAR-019 — KDF universal e limites de segurança

- **Origem**: `_reversa_sdd/internal/crypto/kdf/requirements.md`; `_reversa_sdd/domain.md` BR-021..BR-023
- **Confiança original**: 🟢
- **Descrição**: KDF permitido: `scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512`; scrypt exige `N` potência de 2 e valida memória; PBKDF2 exige mínimo de iterações.
- **Justificativa de migração**: Segurança e compatibilidade de keystore.
- **Compatibilidade com paradigma alvo**: Centralizar validação e defaults, evitando duplicação.

### BR-MIGRAR-020 — Compatibilidade/análise KDF opcional

- **Origem**: `_reversa_sdd/internal/cli/salvar-keystore/requirements.md`; `_reversa_sdd/internal/crypto/kdf/analisar-compatibilidade/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Quando análise ou verbose estiverem ativos, exibir relatório de compatibilidade KDF a partir dos parâmetros completos do keystore.
- **Justificativa de migração**: Recurso operacional útil e confirmado.
- **Compatibilidade com paradigma alvo**: Pode ser serviço de análise desacoplado da escrita em disco.

### BR-MIGRAR-021 — Persistência configurável de keystore/mnemonic

- **Origem**: `_reversa_sdd/internal/cli/salvar-keystore/requirements.md`; `_reversa_sdd/database/data-dictionary.md`
- **Confiança original**: 🟢
- **Descrição**: Persistência só ocorre quando keystore está habilitado; diretório é configurável; mnemonic deve ser salvo quando presente e aplicável.
- **Justificativa de migração**: Backup local é parte do valor do produto.
- **Compatibilidade com paradigma alvo**: Designer deve definir interface de filesystem e política de permissões.

### BR-MIGRAR-022 — Falha de persistência como warning durante display

- **Origem**: `_reversa_sdd/internal/cli/salvar-keystore/requirements.md`; `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Falha ao salvar keystore durante exibição deve ser comunicada como warning/status e não impedir a exibição da carteira.
- **Justificativa de migração**: Preserva o resultado já gerado, mas informa falha operacional.
- **Compatibilidade com paradigma alvo**: Deve distinguir erro de geração vs. erro pós-geração.

### BR-MIGRAR-023 — Permissões restritas para artefatos sensíveis

- **Origem**: `_reversa_sdd/database/business-rules.md`; `_reversa_sdd/permissions.md` § Restrições e controles locais
- **Confiança original**: 🟢/🟡
- **Descrição**: Arquivos sensíveis devem usar permissões restritas e escrita segura/atômica quando aplicável.
- **Justificativa de migração**: Requisito de segurança para keystores, senhas, mnemonics e chaves.
- **Compatibilidade com paradigma alvo**: Implementar por adapter de filesystem testável.

### BR-MIGRAR-024 — Logging seguro e sanitizado

- **Origem**: `_reversa_sdd/pkg/logging/requirements.md`; `_reversa_sdd/questions.md` Pergunta 6; `_reversa_sdd/domain.md` BR-027..BR-030
- **Confiança original**: 🟢
- **Descrição**: Logs não devem conter private key, public key, mnemonic, password, salt ou material criptográfico; campos devem seguir whitelist/sanitização.
- **Justificativa de migração**: Decisão humana e requisito central de segurança.
- **Compatibilidade com paradigma alvo**: Logging deve ser dependência de borda, injetável e testável contra vazamento.

### BR-MIGRAR-025 — Salt nunca deve ser logado

- **Origem**: `_reversa_sdd/internal/crypto/kdf/requirements.md`; `_reversa_sdd/domain.md` BR-026
- **Confiança original**: 🟢
- **Descrição**: Salt/KDF e dados sensíveis não devem aparecer nos logs.
- **Justificativa de migração**: Requisito específico e testável de não vazamento.
- **Compatibilidade com paradigma alvo**: Inspector deve criar cenários negativos de logging.

### BR-MIGRAR-026 — TUI condicional e fallback texto

- **Origem**: `_reversa_sdd/internal/tui/requirements.md`; `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: TUI só deve ser usada quando habilitada, progresso ativo, quiet desligado, terminal compatível; falha de TUI deve cair para texto.
- **Justificativa de migração**: Mantém UX robusta em terminais variados.
- **Compatibilidade com paradigma alvo**: TUI deve ficar isolada da lógica de geração.

### BR-MIGRAR-027 — Variáveis de terminal para visual

- **Origem**: `_reversa_sdd/internal/tui/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: `NO_COLOR` desabilita cores; `TERM=dumb` ou vazio reduz suporte visual; TUI deve ser evitada em CI/stdout redirecionado.
- **Justificativa de migração**: Preserva acessibilidade e automação.
- **Compatibilidade com paradigma alvo**: Implementar como detector de ambiente desacoplado.

### BR-MIGRAR-028 — Progress manager textual corrigido

- **Origem**: `_reversa_sdd/questions.md` Pergunta 7; `_reversa_sdd/gaps.md` GAP-RV-007; `_reversa_sdd/internal/progress/requirements.md`
- **Confiança original**: 🟢
- **Descrição**: Progress manager textual deve ser corrigido contra deadlocks e reativado como fallback textual.
- **Justificativa de migração**: Decisão humana consolidada.
- **Compatibilidade com paradigma alvo**: Deve ser redesenhado com lifecycle/cancelamento testáveis.

### BR-MIGRAR-029 — Métricas, dificuldade e probabilidade

- **Origem**: `_reversa_sdd/pkg/utils/requirements.md`; `_reversa_sdd/domain.md` BR-007..BR-010
- **Confiança original**: 🟢
- **Descrição**: Dificuldade base é `16^len(prefix+suffix)` para Ethereum; checksum multiplica por 2 para cada letra; probabilidade usa `1 - (1 - 1/difficulty)^attempts`; ETA deriva de probabilidade e velocidade.
- **Justificativa de migração**: Usuário depende de stats, benchmark e previsão de custo.
- **Compatibilidade com paradigma alvo**: Deve ser revisado para redes não-hex quando aplicável, mantendo rastreabilidade matemática.

### BR-MIGRAR-030 — CI/CD, Docker e scans de segurança

- **Origem**: `_reversa_sdd/domain.md` BR-037..BR-041; `_reversa_sdd/inventory.md` § CI/CD e Docker
- **Confiança original**: 🟢
- **Descrição**: Pipeline deve cobrir testes, race, lint, gosec/govulncheck/Semgrep, build multi-arch, Docker multi-stage, usuário não-root e releases com checksums/imagem.
- **Justificativa de migração**: Parte do contrato operacional e distribuição do produto.
- **Compatibilidade com paradigma alvo**: Pode ser modernizado, desde que mantenha cobertura e distribuição equivalentes.

## Regras DESCARTAR (resumo)

| ID | Origem | Motivo curto | Vínculo a paradigma? |
|---|---|---|---|
| BR-DESCARTAR-001 | `_reversa_sdd/questions.md` Pergunta 1 | `bloco-vgen` não é nome canônico alvo; manter no máximo compatibilidade controlada. | não |
| BR-DESCARTAR-002 | `_reversa_sdd/questions.md` Pergunta 2 | Flags/claims apenas no README desatualizado não são contrato confirmado. | não |
| BR-DESCARTAR-003 | `_reversa_sdd/pkg/wallet/requirements.md` | Validação hex global para prefixo/sufixo conflita com decisão multirede. | não |
| BR-DESCARTAR-004 | `_reversa_sdd/pkg/wallet/requirements.md` | `Wallet.IsValid()` centrado em Ethereum conflita com decisão por rede. | não |
| BR-DESCARTAR-005 | `_reversa_sdd/questions.md` Pergunta 5 | Persistência Solana em `.key` bruto é insegura. | não |
| BR-DESCARTAR-006 | `_reversa_sdd/questions.md` Pergunta 6 | `WalletLogger` com private key em claro é inseguro. | não |
| BR-DESCARTAR-007 | `_reversa_sdd/questions.md` Pergunta 7 | Progress textual desabilitado por deadlock não é comportamento alvo. | não |
| BR-DESCARTAR-008 | `_reversa_sdd/questions.md` Pergunta 8 | `EncryptPrivateKeyWithKDF()` não deve ser contrato público. | sim |
| BR-DESCARTAR-009 | `_reversa_sdd/migration/paradigm_decision.md` | Cópia rígida da organização interna do legado é incompatível com apetite transformacional. | sim |

> Detalhe completo em `discard_log.md`.

## Regras DECISÃO HUMANA

### BR-HUMANA-001 — Exibição de private key/mnemonic em stdout no fluxo single

- **Origem**: `_reversa_sdd/internal/cli/gerar-carteiras-vanity/requirements.md` § Regras de Negócio; `_reversa_sdd/permissions.md` PERM-GAP-002
- **Tipo de ambiguidade**: dependência de stakeholder / segurança operacional
- **Descrição**: No fluxo single texto legado, private key é exibida em stdout e mnemonic é exibido quando presente. Isso entrega o segredo ao usuário, mas pode ser risco quando stdout é capturado.
- **Opções**:
  1. Preservar exibição por compatibilidade, com aviso de segurança.
  2. Ocultar por padrão e exigir flag explícita para mostrar segredos.
  3. Mostrar apenas instrução de localização do keystore/mnemonic, sem segredo em stdout.
- **Recomendação do Curator**: opção 2, porque mantém capacidade explícita para o operador e reduz vazamento acidental em automação.
- **Status**: RESOLVIDA (escolha: preservar exibição por compatibilidade com aviso de segurança; decisor: usuário; data: 2026-05-08)

### BR-HUMANA-002 — Escopos amplos em workflows de release

- **Origem**: `_reversa_sdd/permissions.md` PERM-GAP-004; `_reversa_sdd/inventory.md` § CI/CD
- **Tipo de ambiguidade**: dependência de stakeholder / segurança de supply chain
- **Descrição**: Release workflow possui escopos amplos (`issues`, `pull-requests`, `id-token`) cuja necessidade não foi provada nas specs lidas.
- **Opções**:
  1. Preservar escopos do legado por compatibilidade operacional.
  2. Aplicar princípio do menor privilégio e remover escopos não comprovados.
  3. Deixar escopos parametrizados por ambiente/release.
- **Recomendação do Curator**: opção 2, com validação no pipeline alvo, porque o apetite transformacional permite modernizar segurança operacional.
- **Status**: RESOLVIDA (escolha: preservar escopos amplos do legado por compatibilidade operacional; decisor: usuário; data: 2026-05-08)

## Notas

- As regras listadas consolidam duplicatas entre 27 arquivos `requirements.md` e as decisões transversais do Revisor.
- Itens descartados por decisão humana não são perda de requisito: são substituídos por comportamento alvo mais seguro ou mais idiomático.
- O Designer deve tratar a estrutura de pacotes do legado como referência histórica, não como contrato a copiar.
