# Goten

> **Go Language Otentikasi** — composable authentication for Go, inspired by [better-auth](https://better-auth.com), [Limen](https://limenauth.dev), and [Go-better-auth](https://github.com/iambpn/go-better-auth).

[![CI](https://github.com/dnahilman/goten/actions/workflows/ci.yml/badge.svg)](https://github.com/dnahilman/goten/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/dnahilman/goten.svg)](https://pkg.go.dev/github.com/dnahilman/goten)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Goten is a modular authentication library for Go with a **multi-module plugin architecture** — install only what you need, no unused code in your binary.

**Status**: 🚧 v0.1.0 — early release, API may change before v1.0

## Features

| | |
|---|---|
| ✅ Email/password sign-up & sign-in | ✅ Plugin system with capability interfaces |
| ✅ Opaque session tokens (`g10_` prefix) | ✅ Username plugin |
| ✅ Cookie + Bearer auth | ✅ CLI migration tool |
| ✅ GORM adapter (Postgres) | ✅ CSRF origin check |
| ✅ Anti-enumeration on sign-in | ✅ OAuth plugin — Sign in with Google (PKCE + OIDC) |
| ✅ Session list/revoke | 🔜 Magic link, 2FA, JWT plugin |

## Quick Start

```bash
go get github.com/dnahilman/goten
go get github.com/dnahilman/goten/adapters/gorm
```

For a full walkthrough — database setup, migrations, runnable example, and end-to-end testing with `curl` — see the **[Quick Start guide on the Wiki](https://github.com/dnahilman/goten/wiki/Quick-Start)**.

Runnable examples in the repo:
- [`examples/basic/`](examples/basic/) — minimal `net/http` server.
- [`examples/layered-gin/`](examples/layered-gin/) — Gin + GORM + layered architecture (`handler → service → repository`).
- [`examples/oauth-google/`](examples/oauth-google/) — Sign in with Google (OAuth plugin).

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/sign-up/email` | Register with email + password |
| `POST` | `/api/auth/sign-in/email` | Login with email + password |
| `POST` | `/api/auth/sign-out` | Revoke current session |
| `GET`  | `/api/auth/get-session` | Get current session + user |
| `GET`  | `/api/auth/list-sessions` | List all sessions for user |
| `POST` | `/api/auth/revoke-session` | Revoke a specific session |
| `POST` | `/api/auth/revoke-other-sessions` | Revoke all other sessions |

With the **OAuth plugin** enabled:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/sign-in/social` | Start social sign-in → `{redirect, url}` (or sign in with an `idToken`) |
| `GET`  | `/api/auth/callback/{provider}` | OAuth redirect callback |
| `GET`  | `/api/auth/list-accounts` | List the user's linked accounts |
| `POST` | `/api/auth/link-social` | Link a provider to the current user |
| `POST` | `/api/auth/unlink-account` | Unlink a provider |
| `POST` | `/api/auth/get-access-token` | Get a valid provider access token |
| `POST` | `/api/auth/refresh-token` | Refresh the stored provider tokens |

## CLI

The CLI is **generate-only** (like better-auth's `generate`): it emits ORM model
definitions from the core schema plus the active plugins' schema. You apply them with
your ORM — for GORM, `db.AutoMigrate(authmodels.AllModels()...)`.

```bash
go install github.com/dnahilman/goten/cmd/goten@latest

# Generate models into <generate.output_dir>/auth_models.go
goten generate
```

Config file `goten.config.yaml`:

```yaml
plugins:
  - username
  - oauth          # adds the verification table + account token columns to the models

generate:
  output_dir: ./internal/auth
  package: authmodels
  orm: gorm
```

Then wire it into your app:

```go
import authmodels "yourapp/internal/auth"

db.AutoMigrate(authmodels.AllModels()...) // create/upgrade goten's tables
```

> `AutoMigrate` is **additive** (creates tables, adds columns/indexes/constraints). It does
> not drop/rename columns, change types destructively, or roll back. Destructive changes and
> data migrations are handled with your own SQL tooling.

**Editor autocomplete:** add `# yaml-language-server: $schema=https://raw.githubusercontent.com/dnahilman/goten/main/goten.config.schema.json` as the first line of your `goten.config.yaml` for autocomplete + inline validation in VS Code (Red Hat YAML extension), JetBrains, and other editors. See [`examples/basic/goten.config.yaml`](examples/basic/goten.config.yaml).

## Plugins

### Username Plugin

Login via username instead of (or alongside) email:

```bash
go get github.com/dnahilman/goten/plugins/username
```

```go
import usernameplugin "github.com/dnahilman/goten/plugins/username"

auth, _ := goten.New(goten.Config{
    // ...
    Plugins: []goten.Plugin{
        usernameplugin.New(usernameplugin.Options{}),
    },
})
```

Adds endpoints: `POST /api/auth/sign-up/username`, `POST /api/auth/sign-in/username`.

### OAuth Plugin (Sign in with Google)

Social sign-in via OAuth 2.0 Authorization Code + PKCE + OIDC, modeled after better-auth.
Providers are registered in a map (like better-auth's `socialProviders`); Google is built in.

```bash
go get github.com/dnahilman/goten/plugins/oauth
```

```go
import (
    oauthplugin "github.com/dnahilman/goten/plugins/oauth"
    "github.com/dnahilman/goten/plugins/oauth/providers"
)

auth, _ := goten.New(goten.Config{
    BaseURL:        "http://localhost:8080",
    TrustedOrigins: []string{"http://localhost:3000"}, // allowed callbackURL origins
    // ...
    Plugins: []goten.Plugin{
        oauthplugin.New(oauthplugin.Options{
            Providers: map[string]oauthplugin.Provider{
                "google": providers.Google(providers.GoogleOptions{
                    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
                    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
                    AccessType:   "offline", // request a refresh token
                }),
            },
            // EncryptOAuthTokens: true, // opt-in AES-256-GCM token encryption (default off)
        }),
    },
})
```

Then add `oauth` to `plugins` in `goten.config.yaml`, run `goten generate`, and `AutoMigrate`
the result. The generated models gain a core `verification` table (which stores the OAuth
sign-in state) plus token columns on `accounts`.

Security: PKCE (S256), CSRF state (verification row + signed cookie), redirect-origin
validation, and anti account-takeover linking (an existing user is auto-linked only when the
provider's email is verified and, by default, the local account is verified too). Full
walkthrough — including Google Cloud Console setup — in [`examples/oauth-google/`](examples/oauth-google/).

### Building Your Own Plugin

```go
type MyPlugin struct{ auth *goten.Auth }

func (p *MyPlugin) ID() string          { return "my-plugin" }
func (p *MyPlugin) SetAuth(a *goten.Auth) { p.auth = a }
func (p *MyPlugin) Endpoints() []goten.Endpoint {
    return []goten.Endpoint{
        {Method: "GET", Path: "/my-endpoint", Handler: p.handle},
    }
}
```

Optional interfaces: `Initializer`, `EndpointProvider`, `SchemaProvider` (declares the columns the plugin adds, consumed by `goten generate`), `UserCreateHookProvider`, `SessionCreateHookProvider`.

## Architecture

```
github.com/dnahilman/goten              ← core (Auth, session, crypto, plugin system)
github.com/dnahilman/goten/adapters/gorm  ← GORM adapter (separate module)
github.com/dnahilman/goten/plugins/username ← username plugin (separate module)
github.com/dnahilman/goten/plugins/oauth   ← OAuth plugin + Google provider (separate module)
github.com/dnahilman/goten/cmd/goten    ← CLI tool (separate module)
```

Each module is independently versioned — `go get` only what you use.

## ID & Token Format

All IDs and tokens carry a `g10_` prefix for easy identification in logs and secret scanning:

- User/Session ID: `g10_018f4a23-1234-7890-abcd-ef1234567890` (UUID v7, time-sortable)
- Session token: `g10_<base64url-32-bytes>` (256-bit entropy)

## CSRF Protection

Goten applies CSRF origin checking to all non-safe methods (`POST`, `PUT`, `DELETE`, etc.):

- **Bearer token present** → bypass (mobile/API clients)
- **`TrustedOrigins` empty** → allow requests without `Origin` (dev-friendly)
- **`TrustedOrigins` set** → require `Origin` to match; reject others with `403`

```go
auth, _ := goten.New(goten.Config{
    BaseURL:        "https://myapp.com",
    TrustedOrigins: []string{"https://myapp.com", "https://www.myapp.com"},
    // ...
})
```

## Comparison

| | Goten | better-auth (TS) | Limen | Go-better-auth |
|---|---|---|---|---|
| Multi-module plugins | ✅ | ✅ (subpath) | ✅ | ❌ |
| CLI migration tool | ✅ | ✅ | ✅ | ❌ |
| Map-based adapter | ✅ | — | ✅ | ✅ |
| OAuth providers | ✅ (Google) | ✅ | ✅ | ✅ |
| Language | Go | TypeScript | Go | Go |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: [SECURITY.md](SECURITY.md).

## License

MIT — see [LICENSE](LICENSE).
