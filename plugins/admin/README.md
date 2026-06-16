# goten admin plugin

Administrative user management for [goten](https://github.com/dnahilman/goten):
role management, banning, admin-side user CRUD, session control, impersonation,
and a small reusable **RBAC** model. Inspired by better-auth's admin plugin.

```bash
go get github.com/dnahilman/goten/plugins/admin
```

## Setup

```go
import (
    goten "github.com/dnahilman/goten"
    adminplugin "github.com/dnahilman/goten/plugins/admin"
)

auth, _ := goten.New(goten.Config{
    // ...
    Plugins: []goten.Plugin{
        adminplugin.New(adminplugin.Options{
            // AdminUserIDs bootstraps the first admin without a role in the DB.
            AdminUserIDs: []string{"g10_...your-user-id..."},
        }),
    },
})
```

Then add `admin` to `goten.config.yaml`, run `goten generate`, and apply the
generated models. The plugin adds these columns:

| Table | Columns |
|-------|---------|
| `users` | `role` (default `'user'`), `banned` (default `false`), `ban_reason`, `ban_expires` |
| `sessions` | `impersonated_by` |

```yaml
plugins:
  - admin
generate:
  output_dir: ./internal/auth
  package: authmodels
  orm: gorm
```

## Endpoints

All routes are mounted under `<BasePath>/admin/*` (default `/api/auth/admin/*`)
and require the caller to be authenticated **and** hold the listed permission.

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| POST | `/admin/set-role` | `user:set-role` | Set a user's role |
| POST | `/admin/create-user` | `user:create` | Create a user (email + password + role) |
| POST | `/admin/get-user` | `user:get` | Fetch a user (incl. role/ban fields) |
| GET | `/admin/list-users` | `user:list` | List users (`?search=&limit=&offset=&sortBy=&sortDir=`) |
| POST | `/admin/update-user` | `user:update` | Update `name`/`image`/`email` |
| POST | `/admin/set-user-password` | `user:set-password` | Set a user's password |
| POST | `/admin/remove-user` | `user:delete` | Delete a user + revoke sessions |
| POST | `/admin/ban-user` | `user:ban` | Ban a user + revoke their sessions |
| POST | `/admin/unban-user` | `user:ban` | Lift a ban |
| POST | `/admin/impersonate-user` | `user:impersonate` (+ `user:impersonate-admins` for admins) | Sign in as another user |
| POST | `/admin/stop-impersonating` | — | Return to the original admin |
| POST | `/admin/list-user-sessions` | `session:list` | List a user's sessions |
| POST | `/admin/revoke-user-session` | `session:revoke` | Revoke one session |
| POST | `/admin/revoke-user-sessions` | `session:revoke` | Revoke all of a user's sessions |
| POST | `/admin/has-permission` | — | Check whether the caller (or a given role) has permissions |

Errors use goten's shape: `{ "code": "...", "message": "..." }` with the
matching HTTP status.

## Access control (RBAC)

Permissions are modeled as **statements** (resource → actions) and **roles**
(a subset of those actions). The defaults:

```go
// access.DefaultStatements
user:    create, list, set-role, ban, impersonate, impersonate-admins, delete, set-password, get, update
session: list, revoke, delete

// access.DefaultRoles
admin → everything except impersonate-admins
user  → nothing
```

Define custom roles with the `access` subpackage:

```go
import "github.com/dnahilman/goten/plugins/admin/access"

moderator := access.DefaultAC.NewRole(access.Statements{
    "user":    {"list", "get", "ban"},
    "session": {"list", "revoke"},
})

adminplugin.New(adminplugin.Options{
    Roles: map[string]*access.Role{
        "admin":     access.AdminRole,
        "moderator": moderator,
        "user":      access.UserRole,
    },
    AdminRoles: []string{"admin"}, // roles treated as "admin" for the impersonate-admins guard
})
```

A user's role is read from the `users.role` column (comma-separated for multiple
roles). `AdminUserIDs` always pass every check — an escape hatch for the first
admin.

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `DefaultRole` | `"user"` | Role assigned to new users |
| `AdminRoles` | `["admin"]` | Roles treated as admin (must exist in `Roles`) |
| `AdminUserIDs` | `nil` | User IDs that bypass all permission checks |
| `Roles` | `access.DefaultRoles` | Role name → permissions |
| `ImpersonationSessionDuration` | `1h` | Lifetime of impersonation sessions |
| `DefaultBanReason` | `""` | Reason used when `ban-user` omits one |
| `DefaultBanExpiresIn` | `0` (permanent) | Default ban duration |
| `BannedUserMessage` | (built-in) | Message returned to a banned user at sign-in |

## How banning is enforced

- `ban-user` sets the ban fields **and revokes the user's existing sessions**, so
  they are logged out immediately.
- A `SessionCreateHook` rejects new sign-ins for banned users (and auto-unbans
  once `ban_expires` has passed). There is no per-request check — goten has no
  session-validate hook — but the revoke-on-ban closes the main gap.

## Impersonation note

goten exposes no signed-cookie helper to plugins, so this plugin differs from
better-auth's `admin_session` approach:

- `impersonate-user` mints a new session for the target (with `impersonated_by`
  set to the admin's id) and swaps the session cookie. The admin's original
  session is left intact.
- `stop-impersonating` revokes the impersonation session and mints a **fresh**
  session for the original admin.

(A future enhancement could add `SetSignedCookie`/`GetSignedCookie` to goten core
to restore the admin's exact original session.)

## Quick manual test (curl)

```bash
BASE=http://localhost:8080
# admin must be signed in; here we use a bearer token for brevity
TOKEN=g10_...admin-session-token...

# promote a user
curl -s -X POST $BASE/api/auth/admin/set-role \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"userId":"g10_...","role":"admin"}'

# ban a user (1 hour)
curl -s -X POST $BASE/api/auth/admin/ban-user \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"userId":"g10_...","banReason":"spam","banExpiresIn":3600}'

# list users
curl -s "$BASE/api/auth/admin/list-users?limit=20" -H "Authorization: Bearer $TOKEN"
```
