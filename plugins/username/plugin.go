package usernameplugin

import (
	"embed"
	"io/fs"
	"regexp"

	goten "github.com/dnahilman/goten"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var defaultUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// Options configures the username plugin.
type Options struct {
	// Regex overrides the default username validation pattern.
	Regex *regexp.Regexp
	// MinLength overrides the minimum username length (default 3).
	MinLength int
	// MaxLength overrides the maximum username length (default 32).
	MaxLength int
}

// Plugin implements the username authentication plugin.
type Plugin struct {
	opts Options
	auth *goten.Auth
}

// New creates a new username plugin with the given options.
func New(opts Options) *Plugin {
	if opts.Regex == nil {
		opts.Regex = defaultUsernameRegex
	}
	if opts.MinLength == 0 {
		opts.MinLength = 3
	}
	if opts.MaxLength == 0 {
		opts.MaxLength = 32
	}
	return &Plugin{opts: opts}
}

func (p *Plugin) ID() string { return "username" }

func (p *Plugin) SetAuth(a *goten.Auth) { p.auth = a }

func (p *Plugin) Schema() map[string]goten.TableSchema {
	return map[string]goten.TableSchema{
		"users": {
			Fields: []goten.FieldDef{
				{Name: "username", Type: "text", Required: false, Unique: true},
			},
		},
	}
}

func (p *Plugin) Migrations() fs.FS {
	sub, _ := fs.Sub(migrationsFS, "migrations")
	return sub
}

func (p *Plugin) Endpoints() []goten.Endpoint {
	return []goten.Endpoint{
		{Method: "POST", Path: "/sign-up/username", Handler: p.handleSignUp},
		{Method: "POST", Path: "/sign-in/username", Handler: p.handleSignIn},
	}
}

// Compile-time interface checks.
var (
	_ goten.Plugin            = (*Plugin)(nil)
	_ goten.AuthAware         = (*Plugin)(nil)
	_ goten.SchemaProvider    = (*Plugin)(nil)
	_ goten.MigrationProvider = (*Plugin)(nil)
	_ goten.EndpointProvider  = (*Plugin)(nil)
)
