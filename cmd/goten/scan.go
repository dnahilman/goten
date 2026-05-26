package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// pluginImportPrefix is the module path prefix that identifies a Goten
// plugin import. The shorthand name is whatever follows.
const pluginImportPrefix = "github.com/dnahilman/goten/plugins/"

// scanDirsSkipList holds directory names that scanImportedPlugins never
// descends into. These are paths that commonly contain Go files unrelated
// to the user's own application code.
var scanDirsSkipList = map[string]bool{
	"vendor":       true,
	".git":         true,
	".claude":      true,
	"node_modules": true,
	"testdata":     true,
}

// scanImportedPlugins walks the Go source files under rootDir and returns
// the sorted, deduplicated set of Goten plugin shorthand names found in
// import statements (matching pluginImportPrefix).
//
// Test files (*_test.go) are included; blank imports (`_ "..."`) are
// honored too, since they indicate intentional use of plugin side-effects.
func scanImportedPlugins(rootDir string) ([]string, error) {
	seen := map[string]struct{}{}
	fset := token.NewFileSet()

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if scanDirsSkipList[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			// Don't fail the whole scan on a single bad file — the user may
			// have work-in-progress code. Skip it silently.
			return nil
		}
		for _, imp := range f.Imports {
			// imp.Path.Value is the quoted import string, e.g. `"github.com/..."`.
			p := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(p, pluginImportPrefix) {
				name := strings.SplitN(strings.TrimPrefix(p, pluginImportPrefix), "/", 2)[0]
				if name != "" {
					seen[name] = struct{}{}
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

// printImportScanWarnings compares yamlPlugins (from goten.config.yaml) with
// the plugins detected by scanning Go imports under "." and writes a one-line
// warning per drift case to out. Failures while scanning are themselves printed
// as a single info line, not propagated — the validator must never block init.
func printImportScanWarnings(out io.Writer, yamlPlugins []string) {
	imported, err := scanImportedPlugins(".")
	if err != nil {
		fmt.Fprintf(out, "\nnote: skipped Go import scan: %v\n", err)
		return
	}
	yamlSet := toSet(yamlPlugins)
	importedSet := toSet(imported)

	var importedOnly, yamlOnly []string
	for _, p := range imported {
		if _, ok := yamlSet[p]; !ok {
			importedOnly = append(importedOnly, p)
		}
	}
	for _, p := range yamlPlugins {
		if _, ok := importedSet[p]; !ok {
			yamlOnly = append(yamlOnly, p)
		}
	}
	if len(importedOnly) == 0 && len(yamlOnly) == 0 {
		return
	}
	fmt.Fprintln(out)
	for _, p := range importedOnly {
		fmt.Fprintf(out, "⚠ %q is imported in your code but not in migrations.plugins — "+
			"`goten init` will not scaffold its migrations.\n", p)
	}
	for _, p := range yamlOnly {
		fmt.Fprintf(out, "⚠ %q is in migrations.plugins but not imported anywhere — "+
			"its migrations are applied but the plugin code is not loaded.\n", p)
	}
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}
