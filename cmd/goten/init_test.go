package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches CWD to dir for the lifetime of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

// writeFile writes a single file under dir/path, creating parents.
func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunInit_CoreOnly_FreshProject(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml", "database:\n  url: postgres://x/db\n")

	var buf bytes.Buffer
	if err := runInit("goten.config.yaml", "", false, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	for _, name := range []string{
		"20260520120000_initial.up.sql",
		"20260520120000_initial.down.sql",
	} {
		got, err := os.ReadFile(filepath.Join(dir, "migrations", name))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		want, err := coreSource.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from embedded source", name)
		}
	}
	if !strings.Contains(buf.String(), "core:") {
		t.Errorf("expected core summary line, got: %s", buf.String())
	}
}

func TestRunInit_CoreAndUsername_Shorthand(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml",
		"database:\n  url: postgres://x/db\n"+
			"migrations:\n  plugins:\n    - username\n")

	var buf bytes.Buffer
	if err := runInit("goten.config.yaml", "", false, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Plugin migrations should appear at ./plugins/username/migrations/
	entries, err := os.ReadDir(filepath.Join(dir, "plugins", "username", "migrations"))
	if err != nil {
		t.Fatalf("plugin migrations dir missing: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one plugin SQL file")
	}
}

func TestRunInit_CoreAndUsername_FullPath(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml",
		"database:\n  url: postgres://x/db\n"+
			"migrations:\n  plugins:\n    - ./plugins/username/migrations\n")

	var buf bytes.Buffer
	if err := runInit("goten.config.yaml", "", false, &buf); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "username", "migrations")); err != nil {
		t.Fatalf("expected plugin dir from explicit path: %v", err)
	}
}

func TestRunInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml", "database:\n  url: postgres://x/db\n")

	var first, second bytes.Buffer
	if err := runInit("goten.config.yaml", "", false, &first); err != nil {
		t.Fatalf("first runInit: %v", err)
	}
	if err := runInit("goten.config.yaml", "", false, &second); err != nil {
		t.Fatalf("second runInit: %v", err)
	}
	if !strings.Contains(second.String(), "0 written") {
		t.Errorf("second run should report 0 written, got: %s", second.String())
	}
}

func TestRunInit_ConflictWithoutForce(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml", "database:\n  url: postgres://x/db\n")
	writeFile(t, dir, "migrations/20260520120000_initial.up.sql", "-- my custom content\n")

	err := runInit("goten.config.yaml", "", false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when destination has different content")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}

	// Original file must be untouched.
	got, _ := os.ReadFile(filepath.Join(dir, "migrations/20260520120000_initial.up.sql"))
	if string(got) != "-- my custom content\n" {
		t.Errorf("original file was modified: %q", got)
	}
}

func TestRunInit_ConflictWithForce(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml", "database:\n  url: postgres://x/db\n")
	writeFile(t, dir, "migrations/20260520120000_initial.up.sql", "-- my custom content\n")

	if err := runInit("goten.config.yaml", "", true, &bytes.Buffer{}); err != nil {
		t.Fatalf("runInit with --force: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "migrations/20260520120000_initial.up.sql"))
	want, _ := coreSource.ReadFile("migrations/20260520120000_initial.up.sql")
	if !bytes.Equal(got, want) {
		t.Error("--force should overwrite to match embedded source")
	}
}

func TestRunInit_UnknownPlugin(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "goten.config.yaml",
		"database:\n  url: postgres://x/db\n"+
			"migrations:\n  plugins:\n    - magiclink\n")

	err := runInit("goten.config.yaml", "", false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unknown plugin")
	}
	if !strings.Contains(err.Error(), "magiclink") {
		t.Errorf("error should mention the unknown plugin name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "username") {
		t.Errorf("error should list available plugins (username), got: %v", err)
	}
}

func TestRunInit_NoConfigFile_FreshBootstrap(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	// No goten.config.yaml exists — init should use synthetic defaults.

	if err := runInit("goten.config.yaml", "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("runInit without config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "migrations/20260520120000_initial.up.sql")); err != nil {
		t.Fatalf("core file missing after bootstrap-no-config init: %v", err)
	}
}

func TestResolvePluginEntry(t *testing.T) {
	tests := []struct{ entry, wantName, wantDir string }{
		{"username", "username", filepath.Join(".", "plugins", "username", "migrations")},
		{"./plugins/username/migrations", "username", "./plugins/username/migrations"},
		{"./custom/myauth/migrations", "myauth", "./custom/myauth/migrations"},
	}
	for _, tt := range tests {
		gotName, gotDir := resolvePluginEntry(tt.entry)
		if gotName != tt.wantName || gotDir != tt.wantDir {
			t.Errorf("resolvePluginEntry(%q) = (%q, %q), want (%q, %q)",
				tt.entry, gotName, gotDir, tt.wantName, tt.wantDir)
		}
	}
}
