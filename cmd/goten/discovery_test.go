package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverMigrations_SortedByTimestamp(t *testing.T) {
	cfg := &Config{}
	cfg.Migrations.CoreDir = "testdata/migrations"
	cfg.Migrations.Table = "goten_migrations"

	migs, err := discoverMigrations(cfg)
	if err != nil {
		t.Fatalf("discoverMigrations: %v", err)
	}
	if len(migs) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migs))
	}
	if migs[0].ID != "20260101000000" {
		t.Errorf("expected first ID 20260101000000, got %s", migs[0].ID)
	}
	if migs[1].ID != "20260202000000" {
		t.Errorf("expected second ID 20260202000000, got %s", migs[1].ID)
	}
	if migs[0].Plugin != "core" {
		t.Errorf("expected plugin=core, got %s", migs[0].Plugin)
	}
}

func TestDiscoverMigrations_DownFileOptional(t *testing.T) {
	cfg := &Config{}
	cfg.Migrations.CoreDir = "testdata/migrations"
	cfg.Migrations.Table = "goten_migrations"

	migs, err := discoverMigrations(cfg)
	if err != nil {
		t.Fatalf("discoverMigrations: %v", err)
	}
	// create_bar has no .down.sql
	var bar *Migration
	for _, m := range migs {
		if m.Name == "create_bar" {
			bar = m
		}
	}
	if bar == nil {
		t.Fatal("expected create_bar migration")
	}
	if bar.DownPath != "" {
		t.Errorf("expected empty DownPath for create_bar, got %s", bar.DownPath)
	}
}

func TestDiscoverMigrations_MissingCoreDir(t *testing.T) {
	cfg := &Config{}
	cfg.Migrations.CoreDir = "testdata/nonexistent"
	cfg.Migrations.Table = "goten_migrations"

	// Missing dir returns empty list, not error
	migs, err := discoverMigrations(cfg)
	if err != nil {
		t.Fatalf("unexpected error for missing dir: %v", err)
	}
	if len(migs) != 0 {
		t.Errorf("expected 0 migrations for missing dir, got %d", len(migs))
	}
}

func TestSanitizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"add phone number", "add_phone_number"},
		{"Add-Phone-Number", "add_phone_number"},
		{"hello123", "hello123"},
		{"with spaces", "with_spaces"},
		{"UPPER_CASE", "upper_case"},
		{"special!@#chars", "specialchars"},
	}
	for _, tc := range cases {
		got := sanitizeName(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWalkDir_FiltersNonUpSQL(t *testing.T) {
	migs, err := walkDir("testdata/migrations")
	if err != nil {
		t.Fatalf("walkDir: %v", err)
	}
	for _, m := range migs {
		if m.UpPath == "" {
			t.Error("migration has empty UpPath")
		}
	}
}

func TestWalkDir_SkipsUnprefixedFilenames(t *testing.T) {
	// Files that don't match <ts>_<plugin>_<name>.up.sql must be ignored
	// (no fallback to the legacy <ts>_<name>.up.sql shape).
	dir := t.TempDir()
	for _, name := range []string{
		"20260101000000_legacy.up.sql",
		"20260101000000_core_ok.up.sql",
		"not_a_migration.up.sql",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- noop\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	migs, err := walkDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(migs) != 1 || migs[0].Name != "ok" {
		t.Fatalf("expected only the well-formed file to be picked up, got %+v", migs)
	}
	if migs[0].Plugin != "core" {
		t.Errorf("expected plugin=core, got %q", migs[0].Plugin)
	}
}
