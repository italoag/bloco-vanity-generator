# Tasks: Animated ASCII Banner Experience

**Input**: `/specs/001-animated-banner-create/plan.md`, `/specs/001-animated-banner-create/research.md`, `/specs/001-animated-banner-create/data-model.md`, `/specs/001-animated-banner-create/quickstart.md`
**Prerequisites**: plan.md (complete), research.md, data-model.md, quickstart.md

## Phase 3.1: Setup
- [x] T001 Ensure Harmonica dependency is available: `go get github.com/charmbracelet/harmonica@latest` and run `go mod tidy` (`go.mod`, `go.sum`).
- [x] T002 Run formatting and baseline checks to confirm clean state: `make fmt` followed by `golangci-lint run` (expect existing warnings only) and document any pre-existing lint noise (`internal/tui/`, `internal/cli/`, tooling output).

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
- [x] T003 Create failing animation default test in `internal/tui/banner_test.go` that asserts animated frames stream when no motion-reduction signals are present.
- [x] T004 Extend `internal/tui/banner_test.go` with failing cases for env-var disable (`BLOCO_DISABLE_ANIMATION`, `NO_COLOR`, `CLICOLOR=0`) returning static output.
- [x] T005 Add failing override and cancellation tests in `internal/tui/banner_test.go` verifying `--enable-animation` forces animation and context cancellation stops the ticker immediately.

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [x] T006 Implement `BannerPreferences` resolver in `internal/tui/preferences.go` handling flag/env precedence and max CPU target metadata.
- [x] T007 Implement `AnimatedBannerSequence` builder using Harmonica easing in `internal/tui/banner.go`, generating timed frames from `logo.go` lines.
- [x] T008 Refactor `internal/tui/logo.go` to expose reusable frame data (no animation side-effects) consumed by the sequence builder.
- [x] T009 Integrate animation lifecycle into `internal/tui/manager.go`, wiring context-based start/stop and routing output through the TUI renderer.
- [x] T010 Add CLI flag/env plumbing in `internal/cli/commands.go` (and related config) to surface `--enable-animation` and document defaults.

## Phase 3.4: Integration & Quality Gates
- [x] T011 Add benchmark or profiling helper in `internal/tui/banner_bench_test.go` capturing CPU overhead vs. static banner and store comparison results.
- [x] T012 Run `go test -race ./...` to confirm concurrency safety post-changes and resolve any race findings.
- [x] T013 Run `golangci-lint run` and address new lint issues introduced by animation code.
- [x] T014 Run `gosec ./...` ensuring no new security alerts from animation logging or file writes.

## Phase 3.5: Polish
- [ ] T015 Update user docs (`README.md`, `docs/` entries) with animation behavior, accessibility flags, and performance notes.
- [ ] T016 Refresh CLI help output and usage examples to include animation flags (`internal/cli` help generator and `docs/CLI_USAGE.md` via `make docs`).
- [ ] T017 Execute quickstart validation steps end-to-end (per quickstart.md) and record results in the PR description or notes.

## Dependencies
- Tests (T003–T005) must fail before implementing core logic (T006–T010).
- T006 precedes T007–T010 since preference resolution feeds animation lifecycle.
- T011 depends on core implementation to benchmark real behavior.
- Quality gates (T012–T014) run after all code changes.
- Polish tasks (T015–T017) run last once tests and quality gates pass.

## Parallel Execution Example
```
# After completing core implementation (T006–T010), run these in parallel:
/task run T012
/task run T013
/task run T014
```
- `T012`, `T013`, and `T014` touch different commands and can execute concurrently once the codebase compiles.

```