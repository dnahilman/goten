package main

import (
	"embed"
	"path/filepath"
	"sort"
	"strings"

	goten "github.com/dnahilman/goten"
	usernameplugin "github.com/dnahilman/goten/plugins/username"
)

// coreSource holds the core migrations the CLI ships with.
var coreSource embed.FS = goten.CoreMigrationsFS

// pluginSource maps a plugin shorthand name to its embedded migrations FS.
// Every official plugin must register here to be reachable from `goten init`.
// Third-party plugins are not supported by the official CLI — users with
// custom plugins must copy migration files manually or build a custom CLI.
var pluginSource = map[string]embed.FS{
	"username": usernameplugin.MigrationsFS,
}

// availablePluginNames returns the registered plugin shorthand names, sorted,
// for use in error messages.
func availablePluginNames() []string {
	names := make([]string, 0, len(pluginSource))
	for n := range pluginSource {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// resolvePluginEntry interprets a `migrations.plugins[]` entry as either:
//   - a shorthand plugin name (e.g. "username"), expanding to
//     "./plugins/<name>/migrations"
//   - an explicit filesystem path containing "/" or "\"
//
// It returns the canonical (name, dir) pair. The name is used to look up the
// embedded source in pluginSource; the dir is used as the read/write location
// on disk by both `goten init` and the migration discovery walker.
func resolvePluginEntry(entry string) (name, dir string) {
	if strings.ContainsAny(entry, `/\`) {
		return filepath.Base(filepath.Dir(entry)), entry
	}
	return entry, filepath.Join(".", "plugins", entry, "migrations")
}
