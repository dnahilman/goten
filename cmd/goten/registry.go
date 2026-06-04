package main

import (
	"sort"

	goten "github.com/dnahilman/goten"
	oauthplugin "github.com/dnahilman/goten/plugins/oauth"
	usernameplugin "github.com/dnahilman/goten/plugins/username"
)

// pluginRegistry maps a plugin shorthand name to a SchemaProvider, used by
// `goten generate` to collect the columns each plugin contributes. Plugins are
// instantiated with zero-value options because Schema() is static (it does not
// depend on option values).
//
// Every official plugin must register here to be reachable from `goten generate`.
var pluginRegistry = map[string]goten.SchemaProvider{
	"username": usernameplugin.New(usernameplugin.Options{}),
	"oauth":    oauthplugin.New(oauthplugin.Options{}),
}

// availablePluginNames returns the registered plugin shorthand names, sorted.
func availablePluginNames() []string {
	names := make([]string, 0, len(pluginRegistry))
	for n := range pluginRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
