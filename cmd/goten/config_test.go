package main

import (
	"os"
	"path/filepath"
	"testing"

	goten "github.com/dnahilman/goten"
)

func TestLoadConfig_ParseYAML(t *testing.T) {
	cfg, err := loadConfig("testdata/sample.config.yaml")
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(cfg.Plugins))
	}
	if cfg.Generate.OutputDir != "./internal/auth" {
		t.Errorf("unexpected output_dir: %s", cfg.Generate.OutputDir)
	}
	if cfg.Generate.Package != "authmodels" {
		t.Errorf("unexpected package: %s", cfg.Generate.Package)
	}
	if cfg.Generate.ORM != "gorm" {
		t.Errorf("unexpected orm: %s", cfg.Generate.ORM)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte("plugins: [username]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(p)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Generate.OutputDir != "./internal/auth" {
		t.Errorf("expected default output_dir, got %s", cfg.Generate.OutputDir)
	}
	if cfg.Generate.Package != "authmodels" {
		t.Errorf("expected default package, got %s", cfg.Generate.Package)
	}
	if cfg.Generate.ORM != "gorm" {
		t.Errorf("expected default orm gorm, got %s", cfg.Generate.ORM)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := loadConfig("testdata/nonexistent.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMergeSchema_UnknownPlugin(t *testing.T) {
	if _, err := mergeSchema([]string{"does-not-exist"}); err == nil {
		t.Fatal("expected error for unknown plugin")
	}
}

func TestMergeSchema_CoreOnly(t *testing.T) {
	schema, err := mergeSchema(nil)
	if err != nil {
		t.Fatalf("mergeSchema: %v", err)
	}
	for _, table := range []string{"users", "sessions", "accounts", "verification"} {
		if _, ok := schema[table]; !ok {
			t.Errorf("core table %q missing", table)
		}
	}
	// Without the oauth plugin, accounts must NOT carry token columns.
	for _, f := range schema["accounts"].Fields {
		if f.Name == "access_token" {
			t.Errorf("access_token should be absent without the oauth plugin")
		}
	}
}

func TestMergeSchema_WithPlugins(t *testing.T) {
	schema, err := mergeSchema([]string{"username", "oauth"})
	if err != nil {
		t.Fatalf("mergeSchema: %v", err)
	}
	if !hasField(schema["users"].Fields, "username") {
		t.Errorf("username column missing from users")
	}
	if !hasField(schema["accounts"].Fields, "access_token") {
		t.Errorf("access_token column missing from accounts")
	}
}

func hasField(fields []goten.FieldDef, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}
