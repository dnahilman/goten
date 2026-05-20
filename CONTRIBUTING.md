# Contributing to Goten

## Development Setup

```bash
git clone https://github.com/dnahilman/goten
cd goten
make tidy
make build
make test
```

## Workflow

1. Fork → branch → PR
2. All changes require tests
3. `make lint` must be clean before opening a PR
4. Update `CHANGELOG.md` under the `## [Unreleased]` section

## Multi-Module Tips

- Any file change → run `make tidy` (workspace-aware, tidies all modules)
- New plugin → create `plugins/<name>/go.mod` + register in `go.work` + add to `make tidy`
- Run cross-module tests → `make test`
- Build CLI binary → `make cli`

## Code Style

- Follow standard Go formatting (`gofmt`)
- No exported symbol without a doc comment
- Package-level doc goes in `doc.go` or as the first `package` comment
- No `internal/` imports from plugin or adapter modules — use re-exports from root `goten` package

## Reporting Bugs

Open a GitHub issue with:
- Go version (`go version`)
- OS
- Minimal reproduction case

## Security

See [SECURITY.md](SECURITY.md) — please do **not** open public issues for security vulnerabilities.
