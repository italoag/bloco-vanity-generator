# bloco-wallet-generator Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-09-29

## Active Technologies
- (001-animated-banner-create)
- Go 1.24 (module already targets latest) + Charmbracelet Harmonica (animation), existing `internal/tui` toolkit, secure logger (001-animated-banner-create)
- N/A (in-memory rendering only) (001-animated-banner-create)

## Project Structure
```
backend/
frontend/
tests/
```

## Commands
# Add commands for 

## Code Style
: Follow standard conventions

## Recent Changes
- 001-animated-banner-create: Added Go 1.24 (module already targets latest) + Charmbracelet Harmonica (animation), existing `internal/tui` toolkit, secure logger
- 001-animated-banner-create: Added

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->


---

# Reversa

> Framework de Engenharia Reversa instalado neste projeto.

## Como usar

Digite `/reversa` para ativar o Reversa e iniciar ou retomar a análise do projeto.

## Comportamento ao ativar

Quando o usuário digitar `/reversa` ou a palavra `reversa` sozinha em uma mensagem:

1. Ative o skill `reversa` disponível em `.agents/skills/reversa/SKILL.md`
2. Leia o SKILL.md na íntegra e siga exatamente as instruções do Reversa

## Regra não-negociável

Nunca apague, modifique ou sobrescreva arquivos pré-existentes do projeto legado.
O Reversa escreve **apenas** em `.reversa/` e `_reversa_sdd/`.
