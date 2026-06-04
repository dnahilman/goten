package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/cmd/goten/generators"
	"github.com/urfave/cli/v3"
)

func cmdGenerate(_ context.Context, c *cli.Command) error {
	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return err
	}
	if pkg := c.String("package"); pkg != "" {
		cfg.Generate.Package = pkg
	}

	schema, err := mergeSchema(cfg.Plugins)
	if err != nil {
		return err
	}

	gen, ok := generators.Get(cfg.Generate.ORM)
	if !ok {
		return fmt.Errorf("unknown orm %q (available: %s)", cfg.Generate.ORM, strings.Join(generators.Available(), ", "))
	}
	res, err := gen.Generate(schema, generators.Options{
		Package:    cfg.Generate.Package,
		TableOrder: goten.CoreTableOrder,
	})
	if err != nil {
		return err
	}

	out := c.String("output")
	if out == "" {
		out = filepath.Join(cfg.Generate.OutputDir, "auth_models.go")
	}

	if existing, err := os.ReadFile(out); err == nil {
		if string(existing) == res.Code {
			fmt.Println("Schema already up to date:", out)
			return nil
		}
		if !c.Bool("yes") && !confirm(fmt.Sprintf("%s already exists. Overwrite?", out)) {
			return fmt.Errorf("generation aborted")
		}
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(out, []byte(res.Code), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("🚀 Generated %d models → %s\n", len(schema), out)
	return nil
}

// mergeSchema combines the core schema with each active plugin's schema. Plugin
// fields are appended to the table they extend; an unknown plugin is an error.
func mergeSchema(plugins []string) (map[string]goten.TableSchema, error) {
	merged := map[string]goten.TableSchema{}
	for table, ts := range goten.CoreSchema() {
		merged[table] = goten.TableSchema{
			Fields:         append([]goten.FieldDef(nil), ts.Fields...),
			UniqueTogether: append([][]string(nil), ts.UniqueTogether...),
		}
	}
	for _, name := range plugins {
		sp, ok := pluginRegistry[name]
		if !ok {
			return nil, fmt.Errorf("unknown plugin %q (available: %s)", name, strings.Join(availablePluginNames(), ", "))
		}
		for table, ts := range sp.Schema() {
			cur := merged[table]
			cur.Fields = append(cur.Fields, ts.Fields...)
			cur.UniqueTogether = append(cur.UniqueTogether, ts.UniqueTogether...)
			merged[table] = cur
		}
	}
	return merged, nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
