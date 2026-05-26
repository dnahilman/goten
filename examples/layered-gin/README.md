# layered-gin example

Goten + Gin + GORM in a **layered architecture**: `handler → service → repository → model`.

The point of this example is **not** to be production-ready — it's the smallest readable demonstration of where Goten plugs into a typical Go web app with three layers.

## Layout

```
examples/layered-gin/
├── cmd/server/main.go              ← composition root (wire everything)
├── internal/
│   ├── model/user.go               ← GORM struct: UserProfile
│   ├── repository/user_repo.go     ← DB access (CRUD)
│   ├── service/user_service.go     ← business rules (validate phone, etc.)
│   └── handler/
│       ├── middleware.go           ← Gin adapter for goten RequireAuth
│       └── user_handler.go         ← Gin handlers (parse body, call service)
├── docker-compose.yml              ← local Postgres on :5433
├── goten.config.yaml               ← CLI config (uses ${DATABASE_URL})
├── .env.example                    ← copy to .env (gitignored)
└── go.mod                          ← own module, replace-points at the repo
```

Import direction is one-way:

```
handler  ──▶  service  ──▶  repository  ──▶  model
```

Layers below do not know about layers above. Only `main.go` knows about all of them.

## What Goten provides here

- `auth.Handler()` — mounted at `/api/auth/*action` via `gin.WrapH`.
- `auth.RequireAuth(...)` — wrapped as Gin middleware in `internal/handler/middleware.go`.
- `goten.UserFromContext(...)` — handlers read the authenticated user via the small `AuthUserID(c)` helper.

The domain model (`UserProfile`) is **separate** from Goten's `goten_users`. It joins on `UserID` (FK to `goten_users.id`) and stores app-specific fields like `Phone` and `Role`.

## Run it

Prereqs: Go 1.23+, Docker, the `goten` CLI installed (`go install github.com/dnahilman/goten/cmd/goten@latest`).

> **Conflict warning:** uses port `5433` and DB name `goten_layered` to coexist with `examples/basic` (which uses `5432` + `goten`). Stop the other example's Postgres if you have port conflicts.

```bash
cd examples/layered-gin

# 1. Start Postgres
docker compose up -d

# 2. Configure .env
cp .env.example .env   # already points at :5433/goten_layered

# 3. Scaffold Goten migrations (core + username plugin)
goten init

# 4. Apply migrations
goten migrate up

# 5. Run the server (also runs db.AutoMigrate for UserProfile on startup)
go run ./cmd/server
```

Server is on `http://localhost:8080`.

## Try the full flow

```bash
# Sign up (Goten endpoint)
curl -i -X POST http://localhost:8080/api/auth/sign-up/email \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"correct-horse-battery-staple","name":"Alice"}'

# Save the token from the response body
TOKEN="g10_..."

# Create the profile (app endpoint, protected)
curl -i -X POST http://localhost:8080/api/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Alice Wonderland"}'

# Read it back
curl -i http://localhost:8080/api/me \
  -H "Authorization: Bearer $TOKEN"

# Update phone (with validation)
curl -i -X PATCH http://localhost:8080/api/me/phone \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"phone":"+628123456789"}'

# Invalid phone → 400 from the service layer
curl -i -X PATCH http://localhost:8080/api/me/phone \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"phone":"abc"}'
```

## What this example does NOT do (deliberately)

- **No tests.** Adding them is straightforward (repository tests with a real DB, service tests with a fake repo, handler tests with `httptest`); kept out for readability.
- **No structured logging, no metrics, no graceful shutdown.** Bare Gin defaults.
- **`AutoMigrate` for the domain table.** Production should use versioned SQL files (Atlas-generated or hand-written) committed under `./migrations/` so `goten migrate up` applies them alongside core + plugin migrations.
- **No DDD purity.** GORM structs flow through every layer. Upgrade to DDD (separate domain entities + repository interfaces in the domain) once business rules grow complex enough to justify it.

## Compared to `examples/basic`

| | examples/basic | examples/layered-gin |
|---|---|---|
| Routing | stdlib `net/http` | Gin |
| Code layout | single `main.go` | layered (`handler` / `service` / `repository`) |
| Domain table | none | `app_user_profiles` |
| Auth endpoints | from Goten | same |
| Postgres port | 5432 | 5433 |
| DB name | `goten` | `goten_layered` |
