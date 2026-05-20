package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// StartPostgres starts a Postgres container (or uses GOTEN_TEST_DSN env var) and
// returns a connected *gorm.DB plus a cleanup func. Calls t.Fatal on error.
func StartPostgres(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	if dsn := os.Getenv("GOTEN_TEST_DSN"); dsn != "" {
		db, err := gorm.Open(pgdriver.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("connect to GOTEN_TEST_DSN: %v", err)
		}
		return db, func() {}
	}

	ctx := context.Background()
	pgc, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("goten"),
		postgres.WithUsername("goten"),
		postgres.WithPassword("goten"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := pgc.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	port, err := pgc.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("get container port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://goten:goten@%s:%s/goten?sslmode=disable", host, port.Port())

	db, err := gorm.Open(pgdriver.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect to test postgres: %v", err)
	}

	cleanup := func() {
		_ = pgc.Terminate(ctx)
	}
	return db, cleanup
}
