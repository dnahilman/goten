// Package main — Goten "Sign in with Google" example (Postgres, GORM adapter, oauth plugin).
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	goten "github.com/dnahilman/goten"
	gormadapter "github.com/dnahilman/goten/adapters/gorm"
	authmodels "github.com/dnahilman/goten/examples/oauth-google/internal/auth"
	oauthplugin "github.com/dnahilman/goten/plugins/oauth"
	"github.com/dnahilman/goten/plugins/oauth/providers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := envOr("DATABASE_URL", "postgres://goten:goten@localhost:5432/goten?sslmode=disable")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	// Create/upgrade goten's tables from the generated models (run `goten generate`).
	if err := db.AutoMigrate(authmodels.AllModels()...); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	baseURL := envOr("BASE_URL", "http://localhost:8080")
	auth, err := goten.New(goten.Config{
		AppName:  "Goten OAuth (Google) Example",
		BaseURL:  baseURL,
		BasePath: "/api/auth",
		Secret:   envOr("GOTEN_SECRET", "dev-secret-32-bytes-min-please!!"),
		Adapter:  gormadapter.New(db),
		// Frontend origin(s) allowed as callbackURL targets.
		TrustedOrigins: []string{envOr("FRONTEND_URL", "http://localhost:3000")},
		Plugins: []goten.Plugin{
			oauthplugin.New(oauthplugin.Options{
				Providers: map[string]oauthplugin.Provider{
					"google": providers.Google(providers.GoogleOptions{
						ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
						ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
						// Request a refresh token so /refresh-token works.
						AccessType: "offline",
						Prompt:     "consent",
					}),
				},
				// Encrypt provider tokens at rest (opt-in; default false like better-auth).
				EncryptOAuthTokens: true,
			}),
		},
	})
	if err != nil {
		log.Fatalf("goten.New: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	mux.Handle("/api/me", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := goten.UserFromContext(r.Context())
		writeJSON(w, map[string]any{"user": user})
	})))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"app": "goten-oauth-google-example",
			"howTo": []string{
				"1. POST /api/auth/sign-in/social {\"provider\":\"google\",\"callbackURL\":\"" + envOr("FRONTEND_URL", "http://localhost:3000") + "\"}",
				"2. Open the returned url, complete Google consent",
				"3. Google redirects to /api/auth/callback/google, a session cookie is set, then you are sent to callbackURL",
			},
			"endpoints": []string{
				"POST /api/auth/sign-in/social",
				"GET  /api/auth/callback/google",
				"GET  /api/auth/list-accounts (protected)",
				"POST /api/auth/link-social (protected)",
				"POST /api/auth/unlink-account (protected)",
				"POST /api/auth/get-access-token (protected)",
				"POST /api/auth/refresh-token (protected)",
				"GET  /api/me (protected)",
			},
		})
	})

	addr := ":" + envOr("PORT", "8080")
	log.Printf("goten oauth-google example listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
