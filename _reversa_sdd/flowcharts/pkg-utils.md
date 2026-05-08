# Flowchart — pkg/utils

```mermaid
flowchart TD
  A[CalculateDifficulty] --> B[pattern = prefix + suffix]
  B --> C[base = 16^len(pattern)]
  C --> D{checksum?}
  D -- não --> E[base]
  D -- sim --> F[count letters a-f/A-F]
  F --> G[base * 2^letterCount]
  H[CalculateProbability] --> I[1 - (1 - 1/difficulty)^attempts]
  J[CalculateProbability50] --> K[log(0.5)/log(1 - 1/difficulty)]
```
