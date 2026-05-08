# ADR 0007 — Usar CI com testes curtos e scans de segurança

## Status

Aceito retroativamente.

## Contexto

O projeto possui workflows para CI, Go, Docker, Release, Semgrep e version bump. O commit `4a593e0` configura CI para pular testes longos, e `c1856bf` adiciona permissões para packages/security-events.

## Decisão

Executar no CI testes curtos com race detector, lint, gofmt, go vet, gosec, govulncheck e uploads SARIF. Releases por tag geram binários multi-plataforma, checksums e Docker image.

## Evidências

- `.github/workflows/ci.yaml`: `go test -short ./... -race -timeout=90s`, gosec, govulncheck.
- `.github/workflows/semgrep.yml`: Semgrep programado e por PR/push.
- `.github/workflows/release.yaml`: releases, checksums e Docker.
- Git: `4a593e0`, `c1856bf`, `757c580`.

## Consequências

- Pipeline fica mais rápido e viável para PRs.
- Testes longos podem ficar fora do sinal principal de CI.
- Security tab recebe SARIF de scanners.

## Confiança

🟢 CONFIRMADO.
