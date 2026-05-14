# Flowchart — pkg/logging

```mermaid
flowchart TD
  A[NewSecureLogger] --> B[Validate config]
  B --> C{Enabled?}
  C -- não --> D[io.Discard]
  C -- sim --> E{OutputFile?}
  E -- vazio --> F{avoid stdout?}
  F -- sim --> G[/tmp/bloco-vgen.log or discard]
  F -- não --> H[stdout]
  E -- definido --> I[file writer]
  I --> J{BufferSize > 0?}
  J -- sim --> K[async bufferWorker]
  J -- não --> L[sync write]
  M[LogOperation/Error] --> N[sanitize whitelist]
  N --> O[format]
  O --> P[rotate if needed]
  P --> Q[write]
```
