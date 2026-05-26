package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// (resolvePluginEntry lives in registry.go and is reused here.)

// Migration represents a single versioned migration (up + optional down).
type Migration struct {
	ID       string // timestamp, e.g. "20260520120000"
	Name     string // human name, e.g. "initial"
	FullName string // "<ts>_<name>"
	Plugin   string // "core" or plugin folder name
	UpPath   string
	DownPath string // may be empty if .down.sql is missing
}

// discoverMigrations walks the core dir and any plugin dirs listed in config.
// Returns migrations sorted ascending by ID (timestamp).
func discoverMigrations(cfg *Config) ([]*Migration, error) {
	var all []*Migration

	coreMigs, err := walkDir(cfg.Migrations.CoreDir, "core")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("core migrations dir %q: %w", cfg.Migrations.CoreDir, err)
	}
	all = append(all, coreMigs...)

	for _, entry := range cfg.Migrations.Plugins {
		pluginName, pluginDir := resolvePluginEntry(entry)
		pluginMigs, err := walkDir(pluginDir, pluginName)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin migrations dir %q: %w", pluginDir, err)
		}
		all = append(all, pluginMigs...)
	}

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, nil
}

// walkDir reads *.up.sql files from a directory and builds Migration entries.
func walkDir(dir, plugin string) ([]*Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var migs []*Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			continue
		}
		ts, name := parts[0], parts[1]
		downName := fmt.Sprintf("%s_%s.down.sql", ts, name)
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
