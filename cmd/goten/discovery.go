package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Migration represents a single versioned migration (up + optional down).
type Migration struct {
	ID       string // 14-digit timestamp, e.g. "20260520120000"
	Name     string // human name, e.g. "initial"
	FullName string // "<ts>_<plugin>_<name>"
	Plugin   string // "core" or plugin shorthand (extracted from filename)
	UpPath   string
	DownPath string // may be empty if .down.sql is missing
}

// migrationFilenamePattern enforces the flat-layout naming convention:
//
//	<timestamp>_<plugin>_<name>.up.sql
//	<timestamp>_<plugin>_<name>.down.sql
//
// where <plugin> is "core" for core migrations or a plugin shorthand name.
var migrationFilenamePattern = regexp.MustCompile(`^(\d{14})_([a-z][a-z0-9]*)_(.+)\.up\.sql$`)

// discoverMigrations walks a single flat directory (cfg.Migrations.CoreDir)
// and returns all migrations sorted ascending by ID (timestamp).
// Plugin attribution is encoded in each filename, not in directory structure.
func discoverMigrations(cfg *Config) ([]*Migration, error) {
	migs, err := walkDir(cfg.Migrations.CoreDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations dir %q: %w", cfg.Migrations.CoreDir, err)
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].ID < migs[j].ID })
	return migs, nil
}

// walkDir reads *.up.sql files from a directory and builds Migration entries.
// Filenames that don't match the <ts>_<plugin>_<name>.up.sql pattern are skipped.
func walkDir(dir string) ([]*Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var migs []*Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		m := migrationFilenamePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		ts, plugin, name := m[1], m[2], m[3]
		base := strings.TrimSuffix(e.Name(), ".up.sql")
		downName := base + ".down.sql"
		downPath := filepath.Join(dir, downName)
		if _, err := os.Stat(downPath); os.IsNotExist(err) {
			downPath = "" // missing down file — allowed, error only if `down` is called
		}
		migs = append(migs, &Migration{
			ID:       ts,
			Name:     name,
			FullName: base,
			Plugin:   plugin,
			UpPath:   filepath.Join(dir, e.Name()),
			DownPath: downPath,
		})
	}
	return migs, nil
}
