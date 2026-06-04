# Goten — Sign in with Google example

A minimal `net/http` server wiring goten's `oauth` plugin with the built-in Google
provider (OAuth 2.0 Authorization Code + PKCE + OIDC), modeled after better-auth.

## 1. Google Cloud Console setup

1. Create a project → **APIs & Services → Credentials → Create credentials → OAuth client ID**.
2. Application type: **Web application**.
3. **Authorized redirect URIs** — add exactly:
   ```
   http://localhost:8080/api/auth/callback/google
   ```
   (This is `BASE_URL` + `/api/auth/callback/{provider}`. It must match precisely.)
4. Copy the **Client ID** and **Client secret** into `.env`.

## 2. Configure

```bash
cp .env.example .env
# edit .env: set GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET (and DATABASE_URL if needed)
```

## 3. Generate models

From this directory (the `goten` CLI reads `goten.config.yaml`):

```bash
go run github.com/dnahilman/goten/cmd/goten generate
# → writes ./internal/auth/auth_models.go (User/Session/Account/Verification + oauth token columns)
```

The generated package (`internal/auth`, package name `authmodels`) is committed in this
example; re-run `generate` whenever you change the active plugins.

## 4. Start the server

```bash
go run ./cmd/server
# startup runs db.AutoMigrate(authmodels.AllModels()...) to create/upgrade the tables
# listening on :8080
```

## 5. Try the flow

```bash
# 1. Begin sign-in — returns an authorization URL
curl -s -X POST http://localhost:8080/api/auth/sign-in/social \
  -H 'content-type: application/json' -H 'origin: http://localhost:3000' \
  -d '{"provider":"google","callbackURL":"http://localhost:3000"}'
# → {"redirect":true,"url":"https://accounts.google.com/o/oauth2/v2/auth?..."}
```

Open the `url` in a browser, complete consent. Google redirects to
`/api/auth/callback/google`, which creates/links the user, sets the `goten_session`
cookie, and redirects to your `callbackURL`.

Protected endpoints (send the session cookie):

```
GET  /api/me
GET  /api/auth/list-accounts
POST /api/auth/link-social        {"provider":"google","callbackURL":"..."}
POST /api/auth/unlink-account     {"provider":"google"}
POST /api/auth/get-access-token   {"provider":"google"}
POST /api/auth/refresh-token      {"provider":"google"}
```

## Notes

- **State / CSRF**: the sign-in state is stored in the core `verification` table and
  cross-checked against a signed `goten_oauth_state` cookie (better-auth's default
  "database" strategy). PKCE (S256) is always used.
- **Tokens**: this example sets `EncryptOAuthTokens: true`, so access/refresh/id
  tokens are AES-256-GCM encrypted at rest (default is plaintext, matching better-auth).
- **Account linking**: an existing user is auto-linked only when the Google email is
  verified and the local account's email is verified too (anti account-takeover).
