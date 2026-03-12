# Flicker

Minimal durable workflow framework for Go integration middleware.

## What this is

Flicker provides durable execution for multi-step integration workflows without the
operational complexity of Temporal/Cadence. Your code stays your code — no framework DSL,
no magic replay, no special contexts.

## Package structure

```
flicker.go        # Core types: Workflow[R], Status, Signal, RetryPolicy, WorkflowContext
engine.go         # Engine: scheduler (polling) + worker pool + registry + execution
store.go          # WorkflowStore interface + record types
sqlite.go         # SQLite implementation (modernc.org/sqlite, no CGO)
durable/durable.go  # Step cache: read-through/write-through durable step wrapper
example_test.go   # Golden tests demonstrating workflow execution + retry
cmd/flicker/      # Placeholder CLI binary
testdata/         # Golden files (.golden, binary in .gitattributes)
```

## Core concepts

- **Workflow[R]** — generic interface on input type R. `Execute(ctx, request) error`
- **Engine** — scheduler + worker pool combined. Polls store, dispatches to workers
- **WorkflowStore** — interface for workflow persistence. SQLite impl for POC
- **durable.Step[T]** — read-through/write-through step cache. Independently useful
- **Named Steps** — each step has a string key, not a position
- **Three-way outcome** — `return nil` (complete), `return error` (retry), `Stop(WithError(err))` (permanent fail)
- **Status vs Signal** — status = where workflow IS, signal = what you WANT it to do

## Design principles

- Recovery points are internal to the workflow, not exposed to the runner
- Workflow ID injected via `SetWorkflowID()` — explicit structural dependency
- Cached vs fresh reads depend on workflow state (guard mode vs replay mode)
- Compensation is the workflow's problem — framework surfaces failures generically
- Optimistic concurrency control on all state transitions (`WHERE occ_version = $expected`)
- Status and signals are separate concerns

## Key interfaces

Workflows implement `Workflow[R]` + `ExecuteJSON` (for engine dispatch) + embed `WorkflowContext`:

```go
type MyWorkflow struct {
    flicker.WorkflowContext
    store      durable.StepStore
    workflowID string
}

func (w *MyWorkflow) Execute(ctx context.Context, req MyRequest) error { ... }
func (w *MyWorkflow) ExecuteJSON(ctx context.Context, payload []byte) error { ... }
func (w *MyWorkflow) SetWorkflowID(id string) { w.workflowID = id }
func (w *MyWorkflow) GetWorkflowContext() *flicker.WorkflowContext { return &w.WorkflowContext }
```

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

## Testing

Golden tests with `goldie/v2`. Update golden files: `UPDATE_GOLDEN=1 go test ./...` or `make update-golden-go`.

Assertions use `testify/require` (not `testify/assert`).

## Lint rules

- No direct `slog` usage — use instance logger (`w.Log()` or `e.logger`)
- `gofumpt` formatting enforced
- All linters enabled by default (see `.golangci.yml` for disabled list)

## See also

Design notes: `../notes/hardening-external-integrations.md` (in the va workspace)
