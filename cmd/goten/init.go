package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

// cmdInit is the urfave/cli adapter; runInit does the actual work.
func cmdInit(_ context.Context, c *cli.Command) error {
	return runInit(
		c.Root().String("config"),
		c.Root().String("env-file"),
		c.Bool("force"),
		c.Bool("no-scan"),
		os.Stdout,
	)
}

// runInit bootstraps a project by copying embedded migration SQL files for
// core + any plugins declared in goten.config.yaml into a single flat
// directory (cfg.Migrations.CoreDir). Plugin attribution is encoded in
// each file's name. Idempotent: identical files are skipped, divergent
// files require force.
//
// Unless noScan is set, runInit also walks the current directory for Go
// import statements and warns when those drift from migrations.plugins.
func runInit(configPath, envFile string, force, noScan bool, out io.Writer) error {
	cfg, err := loadConfigBestEffort(configPath, envFile)
	if err != nil {
		return err
	}

	dest := cfg.Migrations.CoreDir

	coreStats, err := copyEmbedDir(coreSource, "migrations", dest, force)
	if err != nil {
		return fmt.Errorf("core: %w", err)
	}
	fmt.Fprintf(out, "%-12s %s\n", "core:", coreStats)

	for _, name := range cfg.Migrations.Plugins {
		src, ok := pluginSource[name]
		if !ok {
			return fmt.Errorf("unknown plugin %q in migrations.plugins (available: %s)",
				name, strings.Join(availablePluginNames(), ", "))
		}
		stats, err := copyEmbedDir(src, "migrations", dest, force)
		if err != nil {
			return fmt.Errorf("plugin %s: %w", name, err)
		}
		fmt.Fprintf(out, "%-12s %s\n", name+":", stats)
	}

	if !noScan {
		printImportScanWarnings(out, cfg.Migrations.Plugins)
	}
	return nil
}

// loadConfigBestEffort returns the parsed config, or a synthetic default
// when no config file exists yet (so `goten init` can bootstrap a project
// before any config is written). All other errors are surfaced.
func loadConfigBestEffort(path, envFile string) (*Config, error) {
	cfg, err := loadConfig(path, envFile)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		c := &Config{}
		c.Database.Driver = "postgres"
		c.Migrations.CoreDir = "./migrations"
		c.Migrations.Table = "goten_migrations"
		c.GenerateDir = c.Migrations.CoreDir
		return c, nil
	}
	return nil, err
}

type writeStats struct {
	written, skipped, overwrote int
}

func (s writeStats) String() string {
	return fmt.Sprintf("%d written, %d skipped, %d overwrote",
		s.written, s.skipped, s.overwrote)
}

// copyEmbedDir copies every regular file under `srcRoot` in `src` to `dstDir`
// on disk, honoring the idempotent / --force semantics described on cmdInit.
func copyEmbedDir(src embed.FS, srcRoot, dstDir string, force bool) (writeStats, error) {
	var stats writeStats
	entries, err := src.ReadDir(srcRoot)
	if err != nil {
		return stats, err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return stats, fmt.Errorf("create dir %q: %w", dstDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcContent, err := src.ReadFile(srcRoot + "/" + e.Name())
		if err != nil {
			return stats, err
		}
		dstPath := filepath.Join(dstDir, e.Name())
		existing, err := os.ReadFile(dstPath)
		switch {
		case err == nil && bytes.Equal(existing, srcContent):
			stats.skipped++
		case err == nil:
			if !force {
				return stats, fmt.Errorf(
					"%s exists with different content; re-run with --force to overwrite",
					dstPath,
				)
			}
			if err := os.WriteFile(dstPath, srcContent, 0o644); err != nil {
				return stats, err
			}
			stats.overwrote++
		case errors.Is(err, fs.ErrNotExist):
			if err := os.WriteFile(dstPath, srcContent, 0o644); err != nil {
				return stats, err
			}
			stats.written++
		default:
			return stats, fmt.Errorf("read %q: %w", dstPath, err)
		}
	}
	return stats, nil
}
