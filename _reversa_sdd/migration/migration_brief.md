---
schemaVersion: 1
generatedAt: 2026-05-08T10:28:00Z
reversa:
  version: "1.2.34"
kind: migration_brief
producedBy: orchestrator
hash: "sha256:4dc254d1c0117eccd00bf69b17057247264ca87195ab2f4d21623bcef07dcdb1"
---

# Migration Brief

> Documento de critério de migração coletado em entrevista no início do `/reversa-migrate`.  
> Consumido pelos cinco agentes do Time de Migração. Não pergunta paradigma nem apetite.

## Objetivo da migração

Migrar/reconstruir o legado como `bloco-vanity-generator`, preservando o valor principal: geração local de carteiras vanity multi-rede com alta performance, CLI/TUI utilizável e higiene forte de segredos. O binário alvo deve manter compatibilidade como `bloco-vgen`.

## Métricas de sucesso

- Paridade funcional para geração vanity em Ethereum, Bitcoin e Solana.
- Validação de prefixo/sufixo por rede: Ethereum hex/EIP-55, Bitcoin Base58/bech32 quando aplicável, Solana Base58.
- Nenhum vazamento de private key, mnemonic, password ou salt em logs/stdout indevido.
- Persistência Solana segura, sem `.key` bruto.
- Performance comparável ou melhor que o worker pool legado em cenários equivalentes.
- Testes automatizados cobrindo CLI, crypto, KDF, logging, persistência e concorrência.

## Restrições

- **Prazo**: não informado.
- **Orçamento**: não informado.
- **Técnicas**: aplicação local; sem backend obrigatório; sem banco; preservar KeyStore V3/AES-128-CTR/KDF; manter UX CLI/TUI.
- **Operacionais**: artefatos sensíveis devem ficar em filesystem local com permissões restritas.

## Fatores de risco conhecidos

- Regressão criptográfica em geração de chaves/endereço por rede.
- Degradação de performance no hot path de geração.
- Persistência insegura de segredos em logs, stdout ou arquivos.
- Divergência entre documentação antiga e comportamento real da CLI.
- TUI/progress textual reintroduzindo deadlocks.

## Stakeholders

| Nome / papel | Responsabilidade na migração |
|---|---|
| Italo / owner técnico | Aprovar escopo, decisões de produto e trade-offs |
| Usuário CLI | Validar UX, flags, saída e compatibilidade operacional |
| Revisor de segurança | Validar higiene de segredos e criptografia |
| Mantenedor/release | Validar build, Docker, CI e distribuição |

## Stack alvo

- **Linguagem**: Go moderno, compatível com a toolchain atual do projeto.
- **Framework**: Cobra/Fang para CLI; Bubble Tea/Bubbles/Lip Gloss para TUI, salvo decisão posterior.
- **Banco**: nenhum.
- **Mensageria**: nenhuma.
- **Infra**: execução local por binário; Docker multi-stage; GitHub Actions para CI/release.
- **Outros componentes relevantes**: logging seguro/sanitizado local, sem tracing remoto obrigatório.

## Escopo declarado

- **Incluído**: CLI, TUI, geração Ethereum/Bitcoin/Solana, validação vanity por rede, worker pool, stats/benchmark, KeyStore/KDF, mnemonic, logging seguro, Docker/CI/release.
- **Excluído**: backend HTTP/RPC, banco de dados, filas, RBAC de aplicação, serviços remotos obrigatórios.

## Notas livres

README legado deve ser tratado como fonte secundária quando divergir do Cobra/código. Decisões humanas já consolidadas pelo Revisor devem orientar o alvo: nomenclatura `bloco-vanity-generator`/`bloco-vgen`, validação por rede, Solana sem `.key` bruto, logging seguro, progress textual corrigido e `EncryptPrivateKeyWithKDF()` como detalhe interno.
