# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure
2. Load optional design documents:
   → data-model.md: Extract entities → model tasks
   → contracts/: Each file → contract test task
   → research.md: Extract decisions → setup tasks
3. Generate tasks by category:
   → Setup: module hygiene, dependencies, linting
   → Tests: contract/integration/unit tests (must fail first)
   → Core: services, worker changes, CLI wiring
   → Integration: logging, concurrency controls, persistence
   → Quality: race detection, security scans, benchmarks, docs
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests?
   → All entities have models?
   → All endpoints implemented?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- Production code lives in `cmd/` (entry points) and `internal/` packages (`cli`, `worker`, `crypto`, `config`, `tui`, `validation`, `progress`).
- Shared libraries belong in `pkg/` when they need external reuse.
- Tests sit beside their packages as `_test.go` files; integration fixtures may live under `internal/<module>/testdata`.
- Generated keystore artifacts remain under `keystores/` and MUST NOT be committed.

## Phase 3.1: Setup
- [ ] T001 Ensure module tidy state (`go mod tidy`) and capture dependency diffs
- [ ] T002 Configure formatting/lint hooks (`make fmt`, `golangci-lint run`) for touched packages
- [ ] T003 [P] Document plan impact on constitution gates inside spec/plan updates

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [ ] T004 [P] Add contract test in `internal/cli/commands_test.go` covering new flag behavior
- [ ] T005 [P] Add worker integration test in `internal/worker/pool_test.go` validating concurrency boundaries
- [ ] T006 [P] Add crypto security test in `internal/crypto/<feature>_test.go` ensuring no secrets leak
- [ ] T007 [P] Extend benchmark in `internal/worker/bench_test.go` to capture performance target

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [ ] T008 [P] Implement service changes in `internal/worker/<feature>.go` to satisfy tests
- [ ] T009 [P] Wire CLI/TUI handlers in `internal/cli/commands.go` or `internal/tui/manager.go`
- [ ] T010 Harden validation rules in `internal/validation/<feature>.go`
- [ ] T011 Ensure secure logging and docs updates for new behavior (`docs/` + README)
- [ ] T012 Review and adjust configuration bindings in `internal/config/config.go`
- [ ] T013 Extend progress/statistics reporting in `internal/progress` as needed

## Phase 3.4: Integration
- [ ] T014 Validate concurrency safety with `go test -race ./...`
- [ ] T015 Run `golangci-lint run` and resolve findings across touched packages
- [ ] T016 Run `gosec ./...` for security-sensitive changes and address issues
- [ ] T017 Capture updated benchmarks (`go test -bench` or `make bench`) and store results in PR notes

## Phase 3.5: Polish
- [ ] T018 [P] Add focused unit tests for edge cases in affected packages
- [ ] T019 Update CLI help and `docs/` snippets with new examples
- [ ] T020 [P] Verify keystore artifacts and logs exclude secrets (manual validation)
- [ ] T021 Remove duplication and ensure gofmt/goimports clean
- [ ] T022 Capture final `make ci` run and attach output summary to PR

## Dependencies
- Tests (T004-T007) before implementation (T008-T013)
- T008 blocks T009-T013
- Quality gates (T014-T017) block polish (T018-T022)
- Final polish requires all earlier phases complete

## Parallel Example
```
# Launch T004-T007 together:
Task: "Contract test POST /api/users in tests/contract/test_users_post.py"
Task: "Contract test GET /api/users/{id} in tests/contract/test_users_get.py"
Task: "Integration test registration in tests/integration/test_registration.py"
Task: "Integration test auth in tests/integration/test_auth.py"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing
- Commit after each task
- Avoid: vague tasks, same file conflicts

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - Each contract file → contract test task [P]
   - Each endpoint → implementation task
   
2. **From Data Model**:
   - Each entity → model creation task [P]
   - Relationships → service layer tasks
   
3. **From User Stories**:
   - Each story → integration test [P]
   - Quickstart scenarios → validation tasks

4. **Ordering**:
   - Setup → Tests → Models → Services → Endpoints → Polish
   - Dependencies block parallel execution

## Validation Checklist
*GATE: Checked by main() before returning*

- [ ] All contracts have corresponding tests
- [ ] All entities have model tasks
- [ ] All tests come before implementation
- [ ] Parallel tasks truly independent
- [ ] Each task specifies exact file path
- [ ] No task modifies same file as another [P] task