# Módulo internal/tui, Fluxos Operacionais

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Fluxo Principal

```mermaid
flowchart TD
  A[receber entrada] --> B[validar/normalizar]
  B --> C[executar responsabilidade principal]
  C --> D{erro?}
  D -- sim --> E[retornar erro contextual]
  D -- não --> F[retornar resultado]
```

## Fluxos Técnicos Confirmados

- State machine Bubble Tea por mensagens 🟢
- Renderização adaptativa por tamanho de terminal 🟢
- Tabela rolável de resultados 🟢

## Fluxos Alternativos

- **Entrada inválida:** retornar erro conforme contrato da unit. 🟢
- **Dependência falha:** propagar erro ou warning conforme `design.md`. 🟡
- **Dados sensíveis:** aplicar sanitização/ocultação quando a unit manipula chaves, mnemonic, password ou salt. 🟢
