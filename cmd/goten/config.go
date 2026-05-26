package main

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		URL    string `yaml:"url"`
		Driver string `yaml:"driver"`
	} `yaml:"database"`
	Migrations struct {
		CoreDir string   `yaml:"core_dir"`
		Plugins []string `yaml:"plugins"`
		Table   string   `yaml:"table"`
	} `yaml:"migrations"`
	GenerateDir string `yaml:"generate_dir"`
}

// envVarPattern matches ${VAR} but not bare $VAR, so literal $ in values
// (e.g. passwords containing $) are left untouched.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		return os.Getenv(match[2 : len(match)-1])
	})
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w (create a goten.config.yaml or use --config)", path, err)
	}
	data = []byte(expandEnv(string(data)))
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Defaults
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Migrations.CoreDir == "" {
		cfg.Migrations.CoreDir = "./migrations"
	}
	if cfg.Migrations.Table == "" {
		cfg.Migrations.Table = "goten_migrations"
	}
	if cfg.GenerateDir == "" {
		cfg.GenerateDir = cfg.Migrations.CoreDir
	}
	// Env override
	if envURL := os.Getenv("GOTEN_DATABASE_URL"); envURL != "" {
		cfg.Database.URL = envURL
	}
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("database.url required — set in config file or GOTEN_DATABASE_URL env var")
	}
	return &cfg, nil
}
