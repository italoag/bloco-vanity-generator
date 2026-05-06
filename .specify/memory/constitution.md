<!--
Sync Impact Report
Version change: 0.0.0 → 1.0.0
Modified principles:
- Template principle 1 placeholder → Idiomatic Go & Cohesive Packages
- Template principle 2 placeholder → Cryptographic Security & Secret Hygiene
- Template principle 3 placeholder → Performance-Driven Concurrency
- Template principle 4 placeholder → Layered Architecture & Extensible Design
- Template principle 5 placeholder → Tested, Observable, and Documented Delivery
Added sections:
- Engineering Standards
- Delivery Workflow & Reviews
Removed sections:
- None
Templates requiring updates:
- ✅ .specify/templates/plan-template.md
- ✅ .specify/templates/spec-template.md
- ✅ .specify/templates/tasks-template.md
Follow-up TODOs:
- None
-->

# Bloco Vanity Generator Constitution

## Core Principles

### I. Idiomatic Go & Cohesive Packages
- All Go code MUST be formatted with `gofmt` and import-sorted with `goimports` before merge; `make fmt` remains the baseline gate.
- Packages under `internal/` MUST expose cohesive APIs with exported identifiers documented via GoDoc comments that cover error and security expectations.
- Functions exceeding roughly 50 lines MUST be decomposed; prefer early-return control flow and explicit error handling to maintain readability.
- Interfaces shared across packages MUST be declared on the consumer side to keep dependencies minimal and enable mocking in tests.

*Rationale: Idiomatic structure and disciplined packages keep contributors productive and reduce regressions in critical wallet tooling.*

### II. Cryptographic Security & Secret Hygiene
- All key material (private keys, mnemonics, keystore passwords) MUST stay confined to secure memory and `keystores/`; logging or printing secrets is prohibited.
- Cryptographic operations MUST route through the vetted modules in `internal/crypto` and `github.com/ethereum/go-ethereum/crypto`; changes require documented security review.
- Keystore derivation MUST default to production-grade KDF presets; weakening parameters demands a mitigation note in docs and explicit CLI safeguards.
- Files that contain secrets MUST be written atomically with `0600` permissions, and tests MUST cover failure paths to verify secrets are not orphaned on disk.

*Rationale: The generator’s reputation depends on uncompromised handling of sensitive material.*

### III. Performance-Driven Concurrency
- Worker pool changes MUST maintain non-blocking statistics updates and honor `context.Context` cancellation propagation down to goroutines.
- Concurrency primitives MUST avoid unsynchronized shared state; `go test -race ./...` MUST pass locally and in CI before merge.
- Any change that affects generation throughput by ±5% MUST ship with benchmarks or profiling evidence using `make bench` or targeted Go benchmarks.
- Long-running operations MUST expose tunable limits (threads, batch sizes) and document their impact on CPU, memory, and IO usage.

*Rationale: Sustained throughput and responsive cancellation make vanity generation usable at scale.*

### IV. Layered Architecture & Extensible Design
- CLI/TUI layers (`internal/cli`, `internal/tui`) MUST depend on domain services via interfaces; they MUST NOT reach directly into worker or crypto internals.
- New capabilities MUST first live in focused services (e.g., `internal/worker`, `internal/config`) and be wired into presentation layers without creating god objects.
- Design patterns (strategy, factory, observer) MUST be applied when they reduce branching or enable plug-in behavior; the chosen pattern MUST be documented in the feature plan rationale.
- Shared types and utilities MUST live in dedicated packages (`internal/` or `pkg/`) to prevent circular dependencies and encourage reuse.

*Rationale: Clear architectural seams enable safe refactors—critical for a security-sensitive codebase evolving under load.*

### V. Tested, Observable, and Documented Delivery
- Test-first delivery is mandatory: write failing contract/integration/unit tests before implementation, and keep them in the repo (`go test ./...`).
- CI and local workflows MUST include `go test -race`, `golangci-lint run`, and `gosec ./...` (when available) prior to merging.
- Structured logging and metrics MUST accompany new behaviors, reusing the secure logger to preserve secrecy while enabling traceability.
- CLI changes MUST update command help, `docs/`, and README examples as part of the same change set.

*Rationale: Observable, well-documented features prove correctness and protect users from silent regressions.*

## Engineering Standards

- **Toolchain**: Target Go 1.24+. Developers MUST run `make fmt vet test-race lint` before opening a pull request; CI MUST cover the same jobs.
- **Static Analysis**: `golangci-lint` findings MUST be resolved or documented with a suppression rationale; disable rules only via consensus in feature plans.
- **Security Scanning**: `gosec ./...` MUST run for changes touching crypto, filesystem, or concurrency primitives; medium-or-higher findings block release.
- **Dependencies**: Prefer the Go standard library; any new third-party library MUST be evaluated for license, maintenance cadence, and security posture before inclusion.
- **Documentation**: Exported APIs, config presets, and CLI flags MUST have GoDoc comments and accompanying updates in `docs/` or README when behavior changes.

## Delivery Workflow & Reviews

- Feature plans MUST document how each principle is satisfied; reviewers MUST reject plans or PRs that omit constitution gates.
- Pull requests MUST link to profiling artifacts when touching performance-critical paths and include before/after metrics when practical.
- Code reviewers MUST confirm that logging remains free of secrets and that new features include automated tests and documentation updates.
- Release candidates MUST pass `make ci security bench` and include a manual validation note confirming keystore artifacts remain secure.

## Governance

- The constitution supersedes conflicting guidance elsewhere in the repo; deviations require explicit approval documented in the relevant spec or plan.
- Amendments require a pull request updating this document plus any impacted templates, with reviewers confirming semantic versioning impact and governance compliance.
- Versioning follows semantic rules: MAJOR for breaking or removed principles, MINOR for new principles/sections, PATCH for clarifications.
- Tech leads (or acting maintainers) MUST run a quarterly compliance review covering tests, security scans, and architecture boundaries; findings belong in `docs/`.

**Version**: 1.0.0 | **Ratified**: 2025-09-29 | **Last Amended**: 2025-09-29