# Goten

> Composable authentication for Go — inspired by [better-auth](https://better-auth.com), [Limen](https://limenauth.dev), and [Go-better-auth](https://github.com/iambpn/go-better-auth).

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
| ✅ Anti-enumeration on sign-in | 🔜 OAuth (Google, GitHub, …) |
| ✅ Session list/revoke | 🔜 Magic link, 2FA, JWT plugin |

## Quick Start

```bash
go get github.com/dnahilman/goten
go get github.com/dnahilman/goten/adapters/gorm
```

```go
package main

import (
    "log"
    "net/http"

    goten "github.com/dnahilman/goten"
    gormadapter "github.com/dnahilman/goten/adapters/gorm"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    db, _ := gorm.Open(postgres.Open("postgres://..."), &gorm.Config{})

    auth, err := goten.New(goten.Config{
        BaseURL: "http://localhost:8080",
        Secret:  "your-32-byte-secret-key-here!!!!",
        Adapter: gormadapter.New(db),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Mount auth endpoints at /api/auth/*
    http.Handle("/api/auth/", auth.Handler())

    // Protect your own endpoints
    http.Handle("/api/me", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, _ := goten.UserFromContext(r.Context())
        // user.ID, user.Email, user.Name ...
    })))

    http.ListenAndServe(":8080", nil)
}
```

See [`examples/basic/`](examples/basic/) for a full runnable example with Docker Compose.

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

## CLI

```bash
go install github.com/dnahilman/goten/cmd/goten@latest

# Apply migrations (core + plugins)
goten migrate up

# Status
goten migrate status

# Roll back last migration
goten migrate down

# Generate new migration file
goten migrate generate add_phone_number
```

Config file `goten.config.yaml`:

```yaml
database:
  url: postgres://user:pass@localhost:5432/mydb?sslmode=disable

migrations:
  core_dir: ./migrations
  plugins:
    - ./plugins/username/migrations
  table: goten_migrations
```

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

Optional interfaces: `Initializer`, `EndpointProvider`, `SchemaProvider`, `MigrationProvider`, `UserCreateHookProvider`, `SessionCreateHookProvider`.

## Architecture

```
github.com/dnahilman/goten              ← core (Auth, session, crypto, plugin system)
github.com/dnahilman/goten/adapters/gorm  ← GORM adapter (separate module)
github.com/dnahilman/goten/plugins/username ← username plugin (separate module)
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
| OAuth providers | 🔜 | ✅ | ✅ | ✅ |
| Language | Go | TypeScript | Go | Go |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: [SECURITY.md](SECURITY.md).

## License

MIT — see [LICENSE](LICENSE).
