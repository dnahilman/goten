package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// EnvFile points to a .env file loaded before YAML interpolation.
	// May not itself reference ${VAR}, since env hasn't been loaded yet when it's read.
	// CLI flag --env-file overrides this; both override the default lookup of ".env" in CWD.
	EnvFile string `yaml:"env_file"`

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

// loadDotenv populates os.Environ from a .env file before config parsing.
// If explicit is non-empty, that path is required and must exist.
// If explicit is empty, ".env" in the current directory is loaded when present
// (missing is fine; any other read/parse error is fatal).
// Existing environment variables are NOT overridden — explicit env wins over file.
func loadDotenv(explicit string) error {
	path := explicit
	if path == "" {
		if _, err := os.Stat(".env"); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("stat .env: %w", err)
		}
		path = ".env"
	}
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("load env file %q: %w", path, err)
	}
	return nil
}

func loadConfig(path, envFileFlag string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w (create a goten.config.yaml or use --config)", path, err)
	}

	// Pre-pass: extract env_file from the raw YAML so we can populate the
	// environment before ${VAR} interpolation runs on the full document.
	// The pre-pass intentionally has no expansion — env_file itself cannot use ${VAR}.
	var pre struct {
		EnvFile string `yaml:"env_file"`
	}
	if err := yaml.Unmarshal(data, &pre); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Precedence: --env-file flag > env_file in YAML > default ".env" in CWD.
	envFile := envFileFlag
	if envFile == "" {
		envFile = pre.EnvFile
	}
	if err := loadDotenv(envFile); err != nil {
		return nil, err
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
