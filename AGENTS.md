# Flicker

Minimal durable workflow framework for Go integration middleware.

## What this is

Flicker provides durable execution for multi-step integration workflows without the
operational complexity of Temporal/Cadence. Your code stays your code — no framework DSL,
no magic replay, no special contexts.

## Package structure

```
flicker.go          # Core types: Workflow[R], Status, Signal, RetryPolicy
context.go          # WorkflowContext: Stop(), Log(), Time, ID
run.go              # Run[T]() — durable step cache (read-through/write-through)
providers.go        # TimeProvider, IDProvider — deterministic on replay
engine.go           # Engine: scheduler (polling) + worker pool + execution
store.go            # WorkflowStore interface + WorkflowRecord + StepResult
registry.go         # Define[R](), WorkflowDef, Factory, Instance
sqlite/sqlite.go    # SQLite implementation (modernc.org/sqlite, pure Go, no CGO)
workflows/          # Example workflow implementations
test/               # Golden tests
cmd/flicker/        # CLI binary (placeholder)
```

## Core concepts

- **Workflow[R]** — generic interface on input type R. `Execute(ctx, request) error`
- **Engine** — scheduler + worker pool. Polls store, dispatches to workers
- **WorkflowStore** — interface for workflow persistence. SQLite impl for POC
- **Run[T]()** — read-through/write-through step cache. Named steps, not positions
- **Three-way outcome** — `return nil` (complete), `return error` (retry), `Stop(WithError(err))` (permanent fail)
- **Status vs Signal** — status = where workflow IS, signal = what you WANT it to do

## Design principles

- Recovery points are internal to the workflow, not exposed to the runner
- Workflow ID on struct (via WorkflowContext), not on context.Context
- WorkflowContext exposes only Stop(), Log(), Time, ID — store is hidden
- Cached vs fresh reads depend on workflow state (guard mode vs replay mode)
- Compensation is the workflow's problem — framework surfaces failures generically
- Optimistic concurrency control on all state transitions (`WHERE occ_version = $expected`)
- Status and signals are separate concerns

## Writing a workflow

Workflows embed `*flicker.WorkflowContext` and implement `flicker.Workflow[R]`:

```go
type MyWorkflow struct {
    *flicker.WorkflowContext
}

var Definition = flicker.Define[MyRequest]("my-workflow", "v1",
    func(wc *flicker.WorkflowContext) flicker.Workflow[MyRequest] {
        return &MyWorkflow{WorkflowContext: wc}
    },
)

func (w *MyWorkflow) Execute(ctx context.Context, req MyRequest) error {
    var result string
    if err := flicker.Run(ctx, w.WorkflowContext, "step_name", &result, func(ctx context.Context) (string, error) {
        // This only runs on first execution. On replay, cached result is returned.
        return callExternalAPI(ctx, req.ID)
    }); err != nil {
        return err
    }
    // use result...
    return nil
}
```

Register and submit:

```go
eng := flicker.NewEngine(store)
factory := Definition.Register(eng)
instance, err := factory.Submit(ctx, MyRequest{ID: "123"})
eng.Start(ctx) // blocks, polls for work
```

## Make targets

| Target | Description |
|---|---|
| `make build` | Build all targets (includes `flicker` binary) |
| `make test` | Run tests |
| `make lint` | Lint code (auto-fixes via golangci-lint) |
| `make format` | Format code |
| `make tidy-go` | Tidy Go modules |
| `make verify` | Run all checks and record verified tree SHA for commit |
| `make pr-ready` | Run comprehensive pre-commit checks |

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
