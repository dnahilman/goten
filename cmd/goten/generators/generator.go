// Package generators turns goten's merged schema metadata into ORM model source
// code. Each supported ORM has its own generator (currently GORM); the structure
// mirrors better-auth's per-ORM generators so new targets can be added later.
package generators

import (
	"sort"

	goten "github.com/dnahilman/goten"
)

// Options configures a generation run.
type Options struct {
	// Package is the Go package name for the emitted file.
	Package string
	// TableOrder lists table names in emission order (referenced tables first).
	// Tables present in the schema but absent here are appended in name order.
	TableOrder []string
}

// Result is the output of a generation run.
type Result struct {
	Code string
}

// Generator emits ORM models from a merged table schema.
type Generator interface {
	Generate(schema map[string]goten.TableSchema, opts Options) (Result, error)
}

var registry = map[string]Generator{
	"gorm": gormGenerator{},
}

// Get returns the generator registered for an ORM id.
func Get(orm string) (Generator, bool) {
	g, ok := registry[orm]
	return g, ok
}

// Available returns the registered ORM ids, sorted.
func Available() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// orderedTables returns the schema's table names in TableOrder first, then any
// remaining tables sorted by name.
func orderedTables(schema map[string]goten.TableSchema, order []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range order {
		if _, ok := schema[t]; ok && !seen[t] {
			out = append(out, t)
			seen[t] = true
		}
	}
	var rest []string
	for t := range schema {
		if !seen[t] {
			rest = append(rest, t)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}
