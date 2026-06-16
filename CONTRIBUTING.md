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

## Releasing (lockstep versioning)

Goten uses **lockstep versioning**: every module shares one version number, so
"goten vX.Y.Z" means the whole project (core, adapters, plugins, CLI) at that
version — like better-auth's Changesets `fixed` group, done with git tags.

To cut a release:

1. Land all changes on `main`; move `CHANGELOG.md` `## [Unreleased]` → `## [X.Y.Z]`.
2. From a clean, pushed `main`:
   ```bash
   make release VERSION=v0.4.0   # release-check (build+test+lint) then tags ALL modules
   ```
   `make release` runs `tag-all`, which tags and pushes every module
   (`vX.Y.Z`, `adapters/gorm/vX.Y.Z`, `plugins/{username,oauth,admin}/vX.Y.Z`,
   `cmd/goten/vX.Y.Z`) in one push so the inter-module `require`s resolve.
3. `gh release create vX.Y.Z --generate-notes` (or with notes).

Notes:
- **New module** → add a `tag-<name>` target and a line in `tag-all`.
- A module's `require github.com/dnahilman/goten*` may stay at an older
  compatible version. Only when a module starts using a **new** feature of a
  sibling, bump it before tagging:
  `cd <module> && go mod edit -require=github.com/dnahilman/goten@vX.Y.Z && cd .. && go work sync`,
  commit, then `make release`.

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
