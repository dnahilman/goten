# Changelog

All notable changes are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **CLI** — `${VAR}` environment-variable interpolation in `goten.config.yaml`. Use e.g. `url: ${GOTEN_DATABASE_URL}` to keep credentials out of the committed config. Bare `$VAR` (no braces) is left untouched so passwords containing literal `$` remain safe.
- **CLI** — automatic `.env` loading from the current working directory, plus `--env-file <path>` flag (Docker-style) for explicit paths. Real environment variables are not overridden — `.env` only fills in missing values.
- **CLI** — `env_file:` field in `goten.config.yaml` to declare the `.env` path inline (e.g. `env_file: ./config/.env.staging`), removing the need for a CLI flag on every invocation. Precedence: `--env-file` flag > `env_file` YAML field > default `.env` in CWD.

## [0.1.0] - 2026-05-20

### Added

- **Core** (`github.com/dnahilman/goten`)
  - Email/password sign-up and sign-in with bcrypt (cost 12)
  - Opaque session management (cookie + Bearer token)
  - Session sliding refresh, list, revoke, revoke-others
  - `g10_` prefix on all IDs (UUID v7) and session tokens (base64url-32-bytes)
  - Plugin system: `Plugin`, `AuthAware`, `Initializer`, `EndpointProvider`, `SchemaProvider`, `MigrationProvider`, `UserCreateHookProvider`, `SessionCreateHookProvider`
  - `ErrHookHandled` sentinel for response-handling hooks
  - `RequireAuth` middleware — injects user + session into context
  - CSRF origin check middleware (permissive when `TrustedOrigins` empty, strict when set)
  - Anti-enumeration: dummy bcrypt verify when user not found on sign-in
  - `New()` returns error — never panics

- **GORM Adapter** (`github.com/dnahilman/goten/adapters/gorm`)
  - Map-based GORM adapter for Postgres (extensible to MySQL/SQLite)
  - Operator whitelist + `quoteIdent` for SQL injection defense
  - `Select(keys)` on Update to handle zero-value fields correctly

- **Username Plugin** (`github.com/dnahilman/goten/plugins/username`)
  - Sign-up and sign-in via username (3–32 chars, alphanumeric + underscore)
  - Partial unique index `WHERE username IS NOT NULL` (coexists with email users)
  - Synthetic email `<username>@username.local.invalid` (RFC 6761 reserved TLD)
  - `//go:embed` migration files via `MigrationProvider`

- **CLI Tool** (`github.com/dnahilman/goten/cmd/goten`)
  - `goten migrate up` — apply pending migrations with per-migration transactions
  - `goten migrate down` — roll back the last applied migration
  - `goten migrate status` — tabular view of applied/pending migrations
  - `goten migrate generate <name>` — create up/down SQL template files
  - `goten.config.yaml` config with `GOTEN_DATABASE_URL` env override
  - Multi-dir discovery: core + plugin migration directories

- **Example** (`github.com/dnahilman/goten/examples/basic`)
  - Runnable Postgres + GORM + username plugin example
  - Docker Compose for local Postgres
