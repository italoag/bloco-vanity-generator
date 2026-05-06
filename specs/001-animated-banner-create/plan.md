
# Implementation Plan: Animated ASCII Banner Experience

**Branch**: `001-animated-banner-create` | **Date**: 2025-09-29 | **Spec**: [/Users/t798157/Projects/lab/bloco-wallet-generator/specs/001-animated-banner-create/spec.md](spec.md)
**Input**: Feature specification from `/specs/001-animated-banner-create/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section using the current constitution (Idiomatic Go, Security, Performance, Architecture, Tests/Docs gates).
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code or `AGENTS.md` for opencode).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Introduce a polished animated banner for the bloco vanity generator TUI by animating the existing ASCII logo with Charmbracelet's Harmonica library while preserving accessibility controls. The banner must auto-detect motion-reduction signals, fall back to the static logo when animation is disabled, and avoid impacting wallet generation throughput or security posture.

## Technical Context
**Language/Version**: Go 1.24 (module already targets latest)  
**Primary Dependencies**: Charmbracelet Harmonica (animation), existing `internal/tui` toolkit, secure logger  
**Storage**: N/A (in-memory rendering only)  
**Testing**: `go test ./...`, `go test -race`, Harmonica animation smoke tests, snapshot assertions for banner output  
**Target Platform**: Terminal-based CLI on macOS/Linux/Windows (interactive TTY)  
**Project Type**: Single CLI/service repository  
**Performance Goals**: Animation CPU overhead ≤5% versus current static banner; no regression in wallet throughput  
**Constraints**: Respect motion-reduction env vars, maintain context cancellation responsiveness, avoid logging secrets  
**Scale/Scope**: Single-user CLI session; limited concurrent goroutines for animation ticker

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Idiomatic Go & Cohesive Packages**: Plan confines changes to `internal/tui` helpers with gofmt/goimports enforced and keeps Harmonica integration behind consumer-owned interfaces.
- **Security & Secret Hygiene**: Animation code avoids touching key material and reuses existing secure logging; no new crypto primitives introduced.
- **Performance-Driven Concurrency**: Animation loop runs on dedicated goroutine with context cancellation and benchmark comparison to confirm ≤5% CPU overhead.
- **Layered Architecture & Extensible Design**: CLI/TUI layers call banner service functions; worker/crypto packages remain untouched, ensuring separation.
- **Tested, Observable, Documented Delivery**: Plan adds failing TUI snapshot tests first, executes `go test -race`, `golangci-lint`, `gosec`, and updates README/docs plus CLI help.

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->
```
cmd/
├── bloco-eth/
│   └── main.go

internal/
├── cli/
├── config/
├── crypto/
├── progress/
├── tui/
├── validation/
└── worker/

pkg/
└── ... (shared utilities)

docs/
└── *.md (architecture, KDF guidance)

keystores/
└── *.json / *.pwd (generated artifacts)

tests live alongside packages; integration assets belong under `internal/` modules.
```

**Structure Decision**: Continue using the existing single-project layout; animation logic lives in `internal/tui/` with possible helpers under `pkg/` if reuse emerges. CLI wiring changes stay in `internal/cli/commands.go` while respecting service boundaries.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - Confirm Harmonica best practices for animating fixed ASCII art.
   - Decide precedence rules between CLI flags and environment variables for motion reduction.
   - Determine acceptable CPU/FPS thresholds and cancellation guarantees.

2. **Generate and dispatch research agents**:
   - Survey Harmonica documentation and examples for frame interpolation.
   - Review accessibility guidance on reduced-motion environment variables for terminal apps.
   - Investigate Go profiling strategies to measure animation CPU overhead alongside worker pools.

3. **Consolidate findings** in `research.md` using format:
   - Decision, rationale, and alternatives for each unknown (recorded in `/specs/001-animated-banner-create/research.md`).

**Output**: `/specs/001-animated-banner-create/research.md` with Harmonica integration notes, motion-reduction detection strategy, CPU overhead mitigation, and cancellation handling guidance.

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Capture `AnimatedBannerSequence`, `BannerPreferences`, and `AnimationContext` structures with validation constraints.

2. **Generate API contracts** from functional requirements:
   - Not applicable; record rationale that banner is internal-only (no REST/IPC surface).

3. **Generate contract tests** from contracts:
   - Not applicable (no contracts).

4. **Extract test scenarios** from user stories:
   - Define failing TUI tests asserting animated vs static output behavior (`internal/tui/banner_test.go`).
   - Document manual verification steps in `/specs/001-animated-banner-create/quickstart.md` aligned with motion toggle scenarios.

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh copilot`
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: `/specs/001-animated-banner-create/data-model.md`, `/specs/001-animated-banner-create/quickstart.md`, agent-specific update (no contracts directory needed).

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base.
- Derive tests from quickstart scenarios (animated default, env disable, override via flag).
- Map entities to implementation tasks: banner renderer, preference resolver, animation context wiring.
- Emphasize Go-specific quality gates (`go test -race`, `golangci-lint`, `gosec`) before implementation polish.

**Ordering Strategy**:
- TDD order: Tests before implementation 
- Dependency order: Models before services before UI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: ~18 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
