# ADR 0006 — Validar checksum e case sensitivity por rede

## Status

Aceito retroativamente.

## Contexto

EIP-55 usa case sensitivity no endereço Ethereum. O histórico mostra correção de checksum (`6d4de6d`) e depois refatoração para adicionar `network` a `matchesCriteria` por case sensitivity (`390898a`) com testes dedicados (`d0fddb0`).

## Decisão

A validação de padrões deve considerar rede. Ethereum pode exigir checksum/case-sensitive; outras redes não devem herdar indevidamente as regras de caixa do Ethereum.

## Evidências

- `internal/crypto/checksum.go`: EIP-55.
- `internal/validation/strategy.go`: estratégias de checksum e case-insensitive.
- Git: `6d4de6d`, `390898a`, `d0fddb0`.

## Consequências

- Corrige falso positivo/falso negativo em padrões mixed-case.
- Aumenta complexidade de validação por rede.
- Exige testes específicos por rede/case.

## Confiança

🟢 CONFIRMADO.
