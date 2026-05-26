package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanImportedPlugins_DetectsUsername(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "main.go", `package main

import (
    usernameplugin "github.com/dnahilman/goten/plugins/username"
    _ "github.com/somebody/unrelated"
)

func _unused() { _ = usernameplugin.Options{} }
`)
	got, err := scanImportedPlugins(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "username" {
		t.Errorf("expected [username], got %v", got)
	}
}

func TestScanImportedPlugins_HonorsBlankImport(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "main.go", `package main

import (
    _ "github.com/dnahilman/goten/plugins/oauth"
)
`)
	got, _ := scanImportedPlugins(dir)
	if len(got) != 1 || got[0] != "oauth" {
		t.Errorf("expected blank import to be detected, got %v", got)
	}
}

func TestScanImportedPlugins_SkipsVendorAndClaude(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "main.go", `package main
import _ "github.com/dnahilman/goten/plugins/username"
`)
	writeGoFile(t, dir, "vendor/x/x.go", `package x
import _ "github.com/dnahilman/goten/plugins/should-not-be-detected"
`)
	writeGoFile(t, dir, ".claude/scratch.go", `package scratch
import _ "github.com/dnahilman/goten/plugins/also-not-detected"
`)
	got, _ := scanImportedPlugins(dir)
	if len(got) != 1 || got[0] != "username" {
		t.Errorf("expected only [username] (vendor/.claude skipped), got %v", got)
	}
}

func TestScanImportedPlugins_DedupAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "a.go", `package x
import _ "github.com/dnahilman/goten/plugins/username"
`)
	writeGoFile(t, dir, "b.go", `package x
import _ "github.com/dnahilman/goten/plugins/username"
`)
	got, _ := scanImportedPlugins(dir)
	if len(got) != 1 {
		t.Errorf("expected dedup, got %v", got)
	}
}

func TestPrintImportScanWarnings_DriftBothWays(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeGoFile(t, dir, "main.go", `package main
import _ "github.com/dnahilman/goten/plugins/username"
`)
	// YAML lists "oauth" but not "username" — drift in both directions.
	var buf bytes.Buffer
	printImportScanWarnings(&buf, []string{"oauth"})

	out := buf.String()
	if !strings.Contains(out, `"username"`) || !strings.Contains(out, "imported in your code but not") {
		t.Errorf("expected warning about username being imported but absent from YAML, got: %s", out)
	}
	if !strings.Contains(out, `"oauth"`) || !strings.Contains(out, "but not imported anywhere") {
		t.Errorf("expected warning about oauth being in YAML but not imported, got: %s", out)
	}
}

func TestPrintImportScanWarnings_NoDriftSilent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeGoFile(t, dir, "main.go", `package main
import _ "github.com/dnahilman/goten/plugins/username"
`)
	var buf bytes.Buffer
	printImportScanWarnings(&buf, []string{"username"})
	if buf.Len() != 0 {
		t.Errorf("expected no output when YAML and imports agree, got: %s", buf.String())
	}
}

func writeGoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
