# User Story: Analisar Performance

> Spec global gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## História

Como **Operador CLI**, eu quero **ver estatísticas e benchmark de geração**, para **estimar tempo, dificuldade e eficiência dos workers**. 🟢

## Critérios de Aceite

```gherkin
Dado que a CLI está instalada e configurada
Quando o usuário executa o fluxo relacionado
Então o comportamento deve seguir as specs de módulo geradas em `_reversa_sdd/`
```

## Rastreabilidade

| Área | Specs relacionadas | Confiança |
|---|---|---:|
| CLI | `internal/cli/`, `cmd/bloco-eth/` | 🟢 |
| Domínio | `pkg/wallet/`, `internal/worker/` | 🟢 |
| Segurança | `internal/crypto/`, `internal/crypto/kdf/`, `pkg/logging/` | 🟢 |
