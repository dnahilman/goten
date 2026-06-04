// Command server runs the layered-gin example app.
//
// Layers:
//
//	handler  → service  → repository  → model (GORM struct)
//
// Aturan import: layer atas boleh import layer bawah; tidak sebaliknya.
// Goten itu infrastructure concern — di-wire di main, dipasang sebagai
// http.Handler dan middleware. Domain (model/repo/service) tidak import goten.
package main

import (
	"log"
	"net/http"
	"os"

	goten "github.com/dnahilman/goten"
	gormadapter "github.com/dnahilman/goten/adapters/gorm"
	authmodels "github.com/dnahilman/goten/examples/layered-gin/internal/auth"
	usernameplugin "github.com/dnahilman/goten/plugins/username"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/dnahilman/goten/examples/layered-gin/internal/handler"
	"github.com/dnahilman/goten/examples/layered-gin/internal/model"
	"github.com/dnahilman/goten/examples/layered-gin/internal/repository"
	"github.com/dnahilman/goten/examples/layered-gin/internal/service"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	dsn := envOr(
		"DATABASE_URL",
		"postgres://goten:goten@localhost:5433/goten_layered?sslmode=disable",
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("connect db: ", err)
	}

	// Goten's auth tables come from generated models (`goten generate`); the
	// domain table (UserProfile) is app-owned. AutoMigrate both during dev.
	models := append(authmodels.AllModels(), &model.UserProfile{})
	if err := db.AutoMigrate(models...); err != nil {
		log.Fatal("automigrate: ", err)
	}

	// Goten auth
	auth, err := goten.New(goten.Config{
		AppName: "Goten Layered-Gin Example",
		BaseURL: envOr("BASE_URL", "http://localhost:8080"),
		Secret:  envOr("GOTEN_SECRET", "dev-secret-32-bytes-min-please!!"),
		Adapter: gormadapter.New(db),
		Plugins: []goten.Plugin{
			usernameplugin.New(usernameplugin.Options{}),
		},
	})
	if err != nil {
		log.Fatal("goten.New: ", err)
	}

	// Wire layers
	userRepo := repository.NewUserRepo(db)
	userSvc := service.NewUserService(userRepo)
	userHnd := handler.NewUserHandler(userSvc)

	// HTTP — Gin
	r := gin.Default()

	// Public
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Goten's auth endpoints under /api/auth/*
	r.Any("/api/auth/*action", gin.WrapH(auth.Handler()))

	// Protected app routes
	api := r.Group("/api", handler.RequireAuth(auth))
	userHnd.Register(api)

	port := envOr("PORT", "8080")
	log.Printf("listening on :%s", port)
	log.Fatal(r.Run(":" + port))
}
