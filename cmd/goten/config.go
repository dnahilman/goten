package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the goten.config.yaml schema for the generate-only CLI.
type Config struct {
	// Plugins lists the active plugin shorthand names whose schema is merged
	// into the generated models (e.g. "username", "oauth").
	Plugins []string `yaml:"plugins"`

	Generate struct {
		// OutputDir is the directory the generated model file is written to.
		OutputDir string `yaml:"output_dir"`
		// Package is the Go package name for the generated file.
		Package string `yaml:"package"`
		// ORM selects the target generator (currently only "gorm").
		ORM string `yaml:"orm"`
	} `yaml:"generate"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w (create a goten.config.yaml or use --config)", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Generate.OutputDir == "" {
		c.Generate.OutputDir = "./internal/auth"
	}
	if c.Generate.Package == "" {
		c.Generate.Package = "authmodels"
	}
	if c.Generate.ORM == "" {
		c.Generate.ORM = "gorm"
	}
}
