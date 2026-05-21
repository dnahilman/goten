// Package main — Goten basic example with Postgres, GORM adapter, and username plugin.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	goten "github.com/dnahilman/goten"
	gormadapter "github.com/dnahilman/goten/adapters/gorm"
	usernameplugin "github.com/dnahilman/goten/plugins/username"
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

	auth, err := goten.New(goten.Config{
		AppName:  "Goten Example",
		BaseURL:  envOr("BASE_URL", "http://localhost:8080"),
		BasePath: "/api/auth",
		Secret:   envOr("GOTEN_SECRET", "dev-secret-32-bytes-min-please!!"),
		Adapter:  gormadapter.New(db),
		Plugins: []goten.Plugin{
			usernameplugin.New(usernameplugin.Options{}),
		},
	})
	if err != nil {
		log.Fatalf("goten init: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	mux.Handle("/api/me", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := goten.UserFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
	})))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"app": "goten-example",
			"endpoints": []string{
				"POST /api/auth/sign-up/email",
				"POST /api/auth/sign-in/email",
				"POST /api/auth/sign-out",
				"GET  /api/auth/get-session",
				"POST /api/auth/sign-up/username",
				"POST /api/auth/sign-in/username",
				"GET  /api/me (protected)",
				"GET  /health",
			},
		})
	})

	addr := ":" + envOr("PORT", "8080")
	log.Printf("goten example listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
