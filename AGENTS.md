# Flicker

Minimal durable workflow framework for Go integration middleware.

## What this is

Flicker provides durable execution for multi-step integration workflows without the
operational complexity of Temporal/Cadence. Your code stays your code — no framework DSL,
no magic replay, no special contexts.

## Core concepts

- **WorkflowStore** — DB table tracking workflow instances (ID, type, version, status)
- **Job Runner** — polling loop that picks up incomplete workflows and calls `Execute()`
- **Durable Wrappers** — GoWrap-generated decorators that add checkpointing to existing interfaces
- **Named Steps** — each step has a key, not a position. Supports parallel execution via errgroup.

## Design principles

- Recovery points are internal to the job, not exposed to the runner
- Workflow ID on struct, not context — explicit structural dependency
- Cached vs fresh reads depend on workflow state (guard mode vs replay mode)
- Compensation is the workflow's problem — framework surfaces failures generically
- Optimistic locking on all state transitions
- Status and signals are separate concerns

## Make targets

| Target | Description |
|---|---|
| `make build` | Build all targets (includes `flicker` binary) |
| `make flicker` | Build the flicker binary |
| `make install` | Install the flicker binary |
| `make test` | Run tests |
| `make lint` | Lint code (auto-fixes via golangci-lint) |
| `make format` | Format code |
| `make tidy-go` | Tidy Go modules |
| `make generate` | Run go generate |
| `make verify` | Run all checks and record verified tree SHA for commit |
| `make pr-ready` | Run comprehensive pre-commit checks (tidy, generate, format, build, lint, test, git-dirty) |
| `make clean` | Clean build artifacts |

## Commit workflow

A pre-commit hook enforces that staged changes have been verified before committing:

1. Stage your changes (`git add ...`)
2. Run `make verify` — runs all checks and records the verified tree SHA
3. Commit (`git commit ...`) — pre-commit hook confirms staged tree matches verified one

If you modify staged files after running `make verify`, you must run it again.

## See also

Design notes: `../notes/hardening-external-integrations.md` (in the va workspace)
