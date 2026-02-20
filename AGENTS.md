# AGENTS.md — Rules for AI Coding Agents

This file defines mandatory rules for any AI agent (Copilot, Cursor, Aider,
Cline, etc.) operating on this codebase. These rules are non-negotiable.

---

## 1. Tests Are Sacred — Never Modify Test Files

**Files matching `*_test.go` must NEVER be modified, deleted, or renamed.**

Tests are the specification. Production code must conform to the tests, not the
other way around. If a test fails after a code change, the code change is wrong.

### Specifically:
- Do **not** change test expectations to make failing tests pass.
- Do **not** delete or skip tests.
- Do **not** rename test functions.
- Do **not** weaken assertions (e.g., changing `assert.Equal` to `assert.Contains`).
- Do **not** add `t.Skip()` to bypass failures.
- You **may** add *new* test files or *new* test functions in existing test files,
  but only to increase coverage — never to replace existing tests.

### Why:
Tests encode the verified behaviour of the system. Modifying them silently
changes the contract, which defeats the purpose of testing.

---

## 2. Coverage Must Not Decrease

Before committing, run:

```bash
task test:coverage
```

If coverage on any testable package drops below its current level, the change
must be revised.

**Testable packages (must stay at 100%):**
- `internal/config`
- `internal/logger`
- `internal/orchestrator`

**Infrastructure packages (tested via mocks at the orchestrator level):**
- `internal/service/vmware.go` — VMware methods require a real ESXi host
- `internal/service/calendar.go` — Google Calendar methods require real credentials
- `cmd/server/main.go` — `run()` wires real services; `getEnvOrDefault()` is tested

---

## 3. Interfaces Live in `interfaces.go`

All service interfaces are defined in `internal/service/interfaces.go`.
When adding a new external dependency, define its interface there first, then
implement it. This keeps the codebase mockable and testable.

---

## 4. Business Logic Lives in the Orchestrator

`internal/orchestrator/orchestrator.go` contains all business logic.
`cmd/server/main.go` is a thin wiring file — it constructs real services and
passes them to the orchestrator. Do not put business logic in `main.go`.

---

## 5. No `os.Exit` Outside `main()`

Only `main()` in `cmd/server/main.go` may call `os.Exit()`. All other code
must return errors to the caller.

---

## 6. Hand-Written Mocks

This project uses hand-written mocks with function fields (not generated mocks).
Each mock struct has fields like `ListVMSnapshotsFn func(...)` that tests set
to control behavior. Do not introduce mock generation tools (mockgen, moq, etc.).

---

## 7. Running Tests

```bash
# Run all tests
task test

# Run tests with coverage report
task test:coverage

# Run tests verbosely
task test:verbose
```
