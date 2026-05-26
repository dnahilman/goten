# Changelog

All notable changes are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed (breaking, pre-v1)

- **Migrations layout is now flat.** All SQL files (core + plugin) live in a single directory (`cfg.Migrations.CoreDir`, default `./migrations`). Plugin attribution is encoded in the filename: `<timestamp>_<plugin>_<name>.{up,down}.sql`. The legacy nested layout (`./plugins/<name>/migrations/`) is no longer supported by the discovery walker.
- **Source migration files renamed** to the new convention:
  - `migrations/20260520120000_initial.{up,down}.sql` → `20260520120000_core_initial.{up,down}.sql`
  - `plugins/username/migrations/20260520130000_add_username.{up,down}.sql` → `20260520130000_username_add_username.{up,down}.sql`
- **`migrations.plugins[]` semantics**: now strictly a list of plugin **shorthand names** to scaffold via `goten init` (e.g. `- username`). Explicit-path entries (`- ./plugins/username/migrations`) are no longer accepted. The field is no longer read by `goten migrate up/down/status` — those commands walk only `core_dir`.

### Added

- **CLI** — `goten init` now writes everything to `cfg.Migrations.CoreDir` (flat). Per-plugin destination subdirectories are no longer created.
- **CLI** — Import-scan validator: after `goten init` runs, it walks your project's `*.go` files and warns when the set of imported `github.com/dnahilman/goten/plugins/*` packages drifts from `migrations.plugins` in YAML (in either direction). Skip with `--no-scan`.
- **Example** — `examples/layered-gin/`: minimal Goten + Gin app organized as `handler → service → repository → model`. Demonstrates the Gin middleware wrapper for `RequireAuth`, a separate `app_user_profiles` table joined to `goten_users`, and `db.AutoMigrate` for the domain table. Postgres on port 5433 to coexist with `examples/basic`.

### Added

- **CLI** — `${VAR}` environment-variable interpolation in `goten.config.yaml`. Use e.g. `url: ${GOTEN_DATABASE_URL}` to keep credentials out of the committed config. Bare `$VAR` (no braces) is left untouched so passwords containing literal `$` remain safe.
- **CLI** — automatic `.env` loading from the current working directory, plus `--env-file <path>` flag (Docker-style) for explicit paths. Real environment variables are not overridden — `.env` only fills in missing values.
- **CLI** — `env_file:` field in `goten.config.yaml` to declare the `.env` path inline (e.g. `env_file: ./config/.env.staging`), removing the need for a CLI flag on every invocation. Precedence: `--env-file` flag > `env_file` YAML field > default `.env` in CWD.
- **CLI** — new top-level `goten init` command scaffolds embedded core + plugin migration SQL files into your project. Reads `goten.config.yaml`'s `migrations.plugins:` list, looks up each plugin in the CLI's internal registry, and copies its SQL files to the project. Idempotent: identical files are skipped, divergent files require `--force`. Unknown plugin names error with the available list. Bootstraps cleanly when no config file exists yet.
- **CLI** — `migrations.plugins:` now accepts plugin **shorthand names** (e.g. `- username`) in addition to explicit paths. Shorthand expands to `./plugins/<name>/migrations` for both reading (migrate up/down) and writing (init).
- **Core** — `goten.CoreMigrationsFS` (`embed.FS`) exposes the core schema migrations for consumption by the CLI's `init` command.
- **Username plugin** — `usernameplugin.MigrationsFS` (renamed from unexported `migrationsFS`) exposes the plugin's migrations to the CLI registry.
- **Docs** — published JSON Schema at [`goten.config.schema.json`](goten.config.schema.json) for editor autocomplete and inline validation of `goten.config.yaml`. Add the `# yaml-language-server: $schema=…` magic comment to your config to enable it in VS Code (Red Hat YAML extension), JetBrains, and other LSP-aware editors.

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
