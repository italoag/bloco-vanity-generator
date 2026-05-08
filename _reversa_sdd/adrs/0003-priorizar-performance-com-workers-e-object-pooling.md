# ADR 0003 — Priorizar performance com workers e object pooling

## Status

Aceito retroativamente.

## Contexto

Geração vanity é busca probabilística potencialmente longa. O README enfatiza alta performance, multi-threading, uso de todos os CPUs, estatísticas em tempo real e arquitetura de pool. O commit `8bbb174` refatora geração de endereço com object pooling.

## Decisão

Usar pool concorrente de workers e otimizações no hot path: reutilização de buffers/objetos criptográficos, geração otimizada de endereço e reconstrução completa da wallet apenas após match.

## Evidências

- README: High-performance, multi-threading, real-time stats.
- `internal/worker/pool.go`: fan-out por thread e primeiro resultado vencedor.
- `internal/crypto/pools.go`: `sync.Pool` para recursos criptográficos/buffers.
- Git: `8bbb174 Refactor: Optimize address generation using object pooling`.

## Consequências

- Maior throughput de tentativas por segundo.
- Mais complexidade de concorrência, métricas e limpeza de memória.
- Necessidade de race detector e testes de concorrência no CI.

## Confiança

🟢 CONFIRMADO.
