package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

func cmdMigrateGenerate(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() < 1 {
		return fmt.Errorf("usage: goten migrate generate <name>")
	}
	name := sanitizeName(c.Args().First())
	if name == "" {
		return fmt.Errorf("migration name must contain at least one alphanumeric character")
	}

	cfg, err := loadConfig(c.Root().String("config"))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.GenerateDir, 0755); err != nil {
		return fmt.Errorf("create generate_dir %q: %w", cfg.GenerateDir, err)
	}

	ts := time.Now().UTC().Format("20060102150405")
	upPath := filepath.Join(cfg.GenerateDir, fmt.Sprintf("%s_%s.up.sql", ts, name))
	downPath := filepath.Join(cfg.GenerateDir, fmt.Sprintf("%s_%s.down.sql", ts, name))
	now := time.Now().Format(time.RFC3339)

	upContent := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n-- Write your UP migration SQL here\n", name, now)
	downContent := fmt.Sprintf("-- Rollback for: %s\n-- Created: %s\n\n-- Write your DOWN migration SQL here\n", name, now)

	if err := os.WriteFile(upPath, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("write %s: %w", upPath, err)
	}
	if err := os.WriteFile(downPath, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("write %s: %w", downPath, err)
	}

	fmt.Printf("✓ Created %s\n", upPath)
	fmt.Printf("✓ Created %s\n", downPath)
	return nil
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
