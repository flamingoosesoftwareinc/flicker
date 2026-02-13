# Agents

## Make targets

| Target | Description |
|---|---|
| `make build` | Build all targets (includes `flicker` binary) |
| `make flicker` | Build the flicker CLI binary |
| `make install` | Install the flicker binary |
| `make test` | Run tests |
| `make lint` | Lint code (auto-fixes via golangci-lint) |
| `make format` | Format code |
| `make tidy-go` | Tidy Go modules |
| `make generate` | Run go generate |
| `make update-golden-go` | Update golden test files |
| `make verify` | Run all checks and record verified tree SHA for commit |
| `make pr-ready` | Run comprehensive pre-commit checks (tidy, generate, format, build, lint, test, git-dirty) |
| `make clean` | Clean build artifacts |

## Commit workflow

A pre-commit hook enforces that staged changes have been verified before committing. The process is:

1. Stage your changes (`git add ...`)
2. Run `make verify` — this runs all checks against the staged tree and records the verified tree SHA
3. Commit (`git commit ...`) — the pre-commit hook confirms the staged tree matches the verified one

If you modify staged files after running `make verify`, you must run it again before committing.
