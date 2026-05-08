# Módulo pkg/logging, Design Técnico

> Spec gerada pelo Reversa Writer.  
> Escala: 🟢 CONFIRMADO no código | 🟡 INFERIDO | 🔴 LACUNA

## Interface

A unit `pkg/logging` é implementada no legado principalmente em `pkg/logging/secure_logger.go`, `pkg/logging/types.go`. Sua interface é formada por funções, structs e contratos internos consumidos pelos demais módulos do monólito CLI. 🟢

| Símbolo | Assinatura / Entrada | Retorno | Observação |
|---------|----------------------|---------|------------|
| `NewSecureLogger` | parâmetros: config *LogConfig | `SecureLogger, error` | 🟢 |
| `LogOperationStart` | parâmetros: operation string, params map[string]interface{} | `error` | 🟢 |
| `LogError` | parâmetros: operation string, err error, context map[string]interface{} | `error` | 🟢 |

## Fluxo Principal

```mermaid
flowchart TD
  A[Entrada da unit] --> B[validar/preparar dados]
  B --> C[executar algoritmo principal]
  C --> D{erro?}
  D -- sim --> E[retornar erro contextual]
  D -- não --> F[retornar resultado/estado]
```

1. A unit recebe dados do módulo consumidor ou da configuração runtime. 🟢
2. Entradas são normalizadas ou validadas conforme regras documentadas em `requirements.md`. 🟢
3. O algoritmo principal executa: Sanitização por whitelist, Redaction de erros e paths, Rotação de arquivos por tamanho, Buffer assíncrono com Flush/Close. 🟢
4. Em erro, a unit retorna erro estruturado, erro contextual ou valor de falha conforme contrato do legado. 🟢
5. Em sucesso, a unit entrega resultado para o próximo componente arquitetural. 🟢

## Fluxos Alternativos

- **Entrada inválida:** validação interrompe o processamento e retorna erro compatível com o legado. 🟢
- **Configuração ausente ou default:** a unit aplica defaults quando o código legado confirma fallback. 🟢
- **Integração indisponível:** erros de dependências são propagados ou convertidos em warning conforme contrato local. 🟡

## Dependências

- `os`: dependência usada pela unit conforme análise do legado. 🟢
- `io`: dependência usada pela unit conforme análise do legado. 🟢
- `path/filepath`: dependência usada pela unit conforme análise do legado. 🟢
- `regexp`: dependência usada pela unit conforme análise do legado. 🟢
- `sync`: dependência usada pela unit conforme análise do legado. 🟢
- `time`: dependência usada pela unit conforme análise do legado. 🟢

## Decisões de Design Identificadas

| Decisão | Evidência no código | Confiança |
|---------|---------------------|-----------|
| A unit permanece encapsulada em `pkg/logging` e é consumida por contratos internos. | `pkg/logging/secure_logger.go`, `pkg/logging/types.go` | 🟢 |
| Regras de validação são explícitas e devem ser preservadas na reimplementação. | `.reversa/context/modules.json` | 🟢 |
| Lacunas conhecidas não devem ser corrigidas sem decisão humana quando alteram comportamento. | `_reversa_sdd/domain.md` | 🟡 |

## Estado Interno

| Estado | Campo | Papel | Confiança |
|---|---|---|---:|
| `LogEntry` | `Timestamp: time.Time` | obrigatório no contrato legado | 🟢 |
| `LogEntry` | `Level: LogLevel` | obrigatório no contrato legado | 🟢 |
| `LogEntry` | `Message: string` | obrigatório no contrato legado | 🟢 |
| `LogEntry` | `Fields: map[string]interface{}` | opcional no contrato legado | 🟢 |

## Observabilidade

- Eventos, erros ou warnings devem manter mensagens e categorias equivalentes quando existentes no legado. 🟢
- Quando a unit opera com dados sensíveis, logs devem permanecer sanitizados e sem segredos. 🟢
- Métricas de performance devem ser preservadas quando a unit expõe stats ou collectors. 🟡

## Riscos e Lacunas

- 🟡 Arquivos legados wallets-*.log podem conter termos sensíveis.
- 🟡 Overflow de buffer deve cair para escrita síncrona.
