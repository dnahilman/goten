package main

import (
	"os"
	"testing"
)

func TestLoadConfig_ParseYAML(t *testing.T) {
	cfg, err := loadConfig("testdata/sample.config.yaml")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://goten:goten@localhost:5432/goten?sslmode=disable" {
		t.Errorf("unexpected URL: %s", cfg.Database.URL)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("unexpected driver: %s", cfg.Database.Driver)
	}
	if cfg.Migrations.CoreDir != "./migrations" {
		t.Errorf("unexpected core_dir: %s", cfg.Migrations.CoreDir)
	}
	if cfg.Migrations.Table != "goten_migrations" {
		t.Errorf("unexpected table: %s", cfg.Migrations.Table)
	}
	if len(cfg.Migrations.Plugins) != 1 {
		t.Errorf("unexpected plugins count: %d", len(cfg.Migrations.Plugins))
	}
	if cfg.GenerateDir != "./migrations" {
		t.Errorf("unexpected generate_dir: %s", cfg.GenerateDir)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	t.Setenv("GOTEN_DATABASE_URL", "postgres://env@host/db")
	cfg, err := loadConfig("testdata/sample.config.yaml")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://env@host/db" {
		t.Errorf("env override not applied, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_MissingURL(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("database:\n  driver: postgres\n")
	f.Close()

	_, err = loadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error when database.url is missing")
	}
}

func TestLoadConfig_EnvInterpolation(t *testing.T) {
	t.Setenv("TEST_DB_URL", "postgres://interp@host:5432/db")
	t.Setenv("TEST_TABLE", "custom_migrations")

	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("database:\n  url: ${TEST_DB_URL}\nmigrations:\n  table: ${TEST_TABLE}\n")
	f.Close()

	cfg, err := loadConfig(f.Name())
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://interp@host:5432/db" {
		t.Errorf("env not interpolated, got: %s", cfg.Database.URL)
	}
	if cfg.Migrations.Table != "custom_migrations" {
		t.Errorf("env not interpolated, got: %s", cfg.Migrations.Table)
	}
}

func TestLoadConfig_EnvInterpolation_BareDollarUntouched(t *testing.T) {
	// Bare $VAR (no braces) must NOT be expanded, so passwords with literal $ are safe.
	t.Setenv("PASS", "should-not-appear")

	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("database:\n  url: \"postgres://u:p$PASS@host/db\"\n")
	f.Close()

	cfg, err := loadConfig(f.Name())
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://u:p$PASS@host/db" {
		t.Errorf("bare $VAR should not be expanded, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_EnvInterpolation_MissingVarExpandsEmpty(t *testing.T) {
	// ${UNSET_VAR} expands to empty string; if it was the only source for
	// database.url, the existing validation catches it.
	os.Unsetenv("DEFINITELY_UNSET_VAR_XYZ")

	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("database:\n  url: ${DEFINITELY_UNSET_VAR_XYZ}\n")
	f.Close()

	_, err = loadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error when interpolated database.url is empty")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("database:\n  url: postgres://x@host/db\n")
	f.Close()

	cfg, err := loadConfig(f.Name())
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.Driver != "postgres" {
		t.Errorf("expected default driver postgres, got %s", cfg.Database.Driver)
	}
	if cfg.Migrations.CoreDir != "./migrations" {
		t.Errorf("expected default core_dir, got %s", cfg.Migrations.CoreDir)
	}
	if cfg.Migrations.Table != "goten_migrations" {
		t.Errorf("expected default table, got %s", cfg.Migrations.Table)
	}
}
