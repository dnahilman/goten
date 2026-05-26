package main

import (
	"os"
	"testing"
)

func TestLoadConfig_ParseYAML(t *testing.T) {
	cfg, err := loadConfig("testdata/sample.config.yaml", "")
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
	cfg, err := loadConfig("testdata/sample.config.yaml", "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://env@host/db" {
		t.Errorf("env override not applied, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := loadConfig("testdata/nonexistent.yaml", "")
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

	_, err = loadConfig(f.Name(), "")
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

	cfg, err := loadConfig(f.Name(), "")
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

	cfg, err := loadConfig(f.Name(), "")
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

	_, err = loadConfig(f.Name(), "")
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

	cfg, err := loadConfig(f.Name(), "")
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

func TestLoadConfig_DotenvExplicit(t *testing.T) {
	envPath := t.TempDir() + "/custom.env"
	if err := os.WriteFile(envPath, []byte("DOTENV_TEST_URL=postgres://from-dotenv@host/db\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := t.TempDir() + "/c.yaml"
	if err := os.WriteFile(cfgPath, []byte("database:\n  url: ${DOTENV_TEST_URL}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("DOTENV_TEST_URL")
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_URL") })

	cfg, err := loadConfig(cfgPath, envPath)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://from-dotenv@host/db" {
		t.Errorf("expected interpolated URL from dotenv, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_DotenvExplicitMissing(t *testing.T) {
	cfgPath := t.TempDir() + "/c.yaml"
	_ = os.WriteFile(cfgPath, []byte("database:\n  url: postgres://x/db\n"), 0644)

	_, err := loadConfig(cfgPath, t.TempDir()+"/nonexistent.env")
	if err == nil {
		t.Fatal("expected error when explicit --env-file is missing")
	}
}

func TestLoadConfig_DotenvCWDAutoLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/.env", []byte("AUTO_LOADED_URL=postgres://auto@host/db\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/c.yaml", []byte("database:\n  url: ${AUTO_LOADED_URL}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("AUTO_LOADED_URL")
	t.Cleanup(func() { os.Unsetenv("AUTO_LOADED_URL") })

	oldWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	cfg, err := loadConfig("c.yaml", "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://auto@host/db" {
		t.Errorf("expected .env auto-loaded, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_DotenvCWDMissingNoError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/c.yaml", []byte("database:\n  url: postgres://no-dotenv/db\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	if _, err := loadConfig("c.yaml", ""); err != nil {
		t.Fatalf("missing .env should not be fatal, got: %v", err)
	}
}

func TestLoadConfig_EnvFileFromYAML(t *testing.T) {
	dir := t.TempDir()
	envPath := dir + "/staging.env"
	if err := os.WriteFile(envPath, []byte("YAML_ENV_URL=postgres://from-yaml-env-file@host/db\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := dir + "/c.yaml"
	cfgBody := "env_file: " + envPath + "\ndatabase:\n  url: ${YAML_ENV_URL}\n"
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0644); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("YAML_ENV_URL")
	t.Cleanup(func() { os.Unsetenv("YAML_ENV_URL") })

	cfg, err := loadConfig(cfgPath, "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://from-yaml-env-file@host/db" {
		t.Errorf("expected env_file from YAML to be loaded, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_EnvFileFlagOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	yamlEnv := dir + "/yaml.env"
	flagEnv := dir + "/flag.env"
	_ = os.WriteFile(yamlEnv, []byte("PRECEDENCE_URL=from-yaml\n"), 0644)
	_ = os.WriteFile(flagEnv, []byte("PRECEDENCE_URL=from-flag\n"), 0644)

	cfgPath := dir + "/c.yaml"
	cfgBody := "env_file: " + yamlEnv + "\ndatabase:\n  url: postgres://x/${PRECEDENCE_URL}\n"
	_ = os.WriteFile(cfgPath, []byte(cfgBody), 0644)

	os.Unsetenv("PRECEDENCE_URL")
	t.Cleanup(func() { os.Unsetenv("PRECEDENCE_URL") })

	cfg, err := loadConfig(cfgPath, flagEnv)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://x/from-flag" {
		t.Errorf("--env-file flag must win over YAML env_file, got: %s", cfg.Database.URL)
	}
}

func TestLoadConfig_EnvFileFromYAMLMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/c.yaml"
	cfgBody := "env_file: " + dir + "/missing.env\ndatabase:\n  url: postgres://x/db\n"
	_ = os.WriteFile(cfgPath, []byte(cfgBody), 0644)

	_, err := loadConfig(cfgPath, "")
	if err == nil {
		t.Fatal("expected error when env_file points to a missing file")
	}
}

func TestLoadConfig_DotenvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(dir+"/.env", []byte("EXISTING_VAR=from-file\n"), 0644)
	_ = os.WriteFile(dir+"/c.yaml", []byte("database:\n  url: postgres://x/${EXISTING_VAR}\n"), 0644)

	t.Setenv("EXISTING_VAR", "from-real-env")

	oldWD, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	cfg, err := loadConfig("c.yaml", "")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Database.URL != "postgres://x/from-real-env" {
		t.Errorf("real env must win over .env file, got: %s", cfg.Database.URL)
	}
}
