package main

import (
	"database/sql"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDB(cfg *Config) (*gorm.DB, error) {
	switch cfg.Database.Driver {
	case "postgres":
		return gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	default:
		return nil, fmt.Errorf("unsupported driver %q (only \"postgres\" is supported in MVP)", cfg.Database.Driver)
	}
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func rawDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}
