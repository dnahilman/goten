package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

func cmdMigrateUp(ctx context.Context, c *cli.Command) error {
	cfg, err := loadConfig(c.Root().String("config"))
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer closeDB(db)

	if err := ensureTrackingTable(db, cfg.Migrations.Table); err != nil {
		return err
	}
	migrations, err := discoverMigrations(cfg)
	if err != nil {
		return err
	}
	applied, err := loadApplied(db, cfg.Migrations.Table)
	if err != nil {
		return err
	}

	var pending []*Migration
	for _, m := range migrations {
		if !applied[m.ID] {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		fmt.Println("✓ No pending migrations")
		return nil
	}

	fmt.Printf("Applying %d migration(s)...\n", len(pending))
	sqlDB, err := rawDB(db)
	if err != nil {
		return err
	}
	for _, m := range pending {
		fmt.Printf("  → %s [%s] ... ", m.FullName, m.Plugin)
		sql, err := os.ReadFile(m.UpPath)
		if err != nil {
			fmt.Println("FAIL")
			return fmt.Errorf("read %s: %w", m.UpPath, err)
		}
		tx, err := sqlDB.BeginTx(ctx, nil)
		if err != nil {
			fmt.Println("FAIL")
			return err
		}
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			_ = tx.Rollback()
			fmt.Println("FAIL")
			return fmt.Errorf("migrate %s: %w", m.FullName, err)
		}
		if err := tx.Commit(); err != nil {
			fmt.Println("FAIL")
			return err
		}
		if err := recordApplied(db, cfg.Migrations.Table, m); err != nil {
			return err
		}
		fmt.Println("OK")
	}
	fmt.Println("✓ Done")
	return nil
}

func cmdMigrateDown(ctx context.Context, c *cli.Command) error {
	cfg, err := loadConfig(c.Root().String("config"))
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer closeDB(db)

	if err := ensureTrackingTable(db, cfg.Migrations.Table); err != nil {
		return err
	}

	last, err := loadLastApplied(db, cfg.Migrations.Table)
	if err != nil {
		return err
	}
	if last == nil {
		fmt.Println("✓ No applied migrations to roll back")
		return nil
	}

	// Find the matching migration file
	migrations, err := discoverMigrations(cfg)
	if err != nil {
		return err
	}
	var target *Migration
	for _, m := range migrations {
		if m.ID == last.ID {
			target = m
			break
		}
	}
	if target == nil {
		return fmt.Errorf("applied migration %q not found in discovery paths — cannot roll back", last.ID)
	}
	if target.DownPath == "" {
		return fmt.Errorf("no .down.sql file for migration %s", target.FullName)
	}

	fmt.Printf("Rolling back %s [%s] ... ", target.FullName, target.Plugin)
	sql, err := os.ReadFile(target.DownPath)
	if err != nil {
		fmt.Println("FAIL")
		return fmt.Errorf("read %s: %w", target.DownPath, err)
	}
	sqlDB, err := rawDB(db)
	if err != nil {
		return err
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		fmt.Println("FAIL")
		return err
	}
	if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
		_ = tx.Rollback()
		fmt.Println("FAIL")
		return fmt.Errorf("rollback %s: %w", target.FullName, err)
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("FAIL")
		return err
	}
	if err := deleteApplied(db, cfg.Migrations.Table, target.ID); err != nil {
		return err
	}
	fmt.Println("OK")
	fmt.Println("✓ Done")
	return nil
}

func cmdMigrateStatus(ctx context.Context, c *cli.Command) error {
	cfg, err := loadConfig(c.Root().String("config"))
	if err != nil {
		return err
	}
	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer closeDB(db)

	if err := ensureTrackingTable(db, cfg.Migrations.Table); err != nil {
		return err
	}
	migrations, err := discoverMigrations(cfg)
	if err != nil {
		return err
	}
	appliedAt, err := loadAppliedAt(db, cfg.Migrations.Table)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPLUGIN\tSTATUS\tAPPLIED AT")
	fmt.Fprintln(w, "--\t----\t------\t------\t----------")
	for _, m := range migrations {
		if at, ok := appliedAt[m.ID]; ok {
			fmt.Fprintf(w, "%s\t%s\t%s\t✓ applied\t%s\n", m.ID, m.Name, m.Plugin, at.Format(time.DateTime))
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t⏳ pending\t-\n", m.ID, m.Name, m.Plugin)
		}
	}
	return w.Flush()
}

// --- Tracking table helpers ---

func ensureTrackingTable(db *gorm.DB, table string) error {
	return db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			plugin     TEXT,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, table)).Error
}

type appliedRow struct {
	ID        string
	AppliedAt time.Time
}

func loadApplied(db *gorm.DB, table string) (map[string]bool, error) {
	var rows []appliedRow
	if err := db.Raw(fmt.Sprintf("SELECT id, applied_at FROM %s", table)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.ID] = true
	}
	return m, nil
}

func loadAppliedAt(db *gorm.DB, table string) (map[string]time.Time, error) {
	var rows []appliedRow
	if err := db.Raw(fmt.Sprintf("SELECT id, applied_at FROM %s", table)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]time.Time, len(rows))
	for _, r := range rows {
		m[r.ID] = r.AppliedAt
	}
	return m, nil
}

func loadLastApplied(db *gorm.DB, table string) (*appliedRow, error) {
	var row appliedRow
	res := db.Raw(fmt.Sprintf("SELECT id, applied_at FROM %s ORDER BY id DESC LIMIT 1", table)).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return &row, nil
}

func recordApplied(db *gorm.DB, table string, m *Migration) error {
	return db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, name, plugin) VALUES (?, ?, ?)", table),
		m.ID, m.Name, m.Plugin,
	).Error
}

func deleteApplied(db *gorm.DB, table, id string) error {
	return db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), id).Error
}
