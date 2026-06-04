package oauth

import (
	goten "github.com/dnahilman/goten"
)

// Plugin implements goten's OAuth/social sign-in plugin.
type Plugin struct {
	opts Options
	auth *goten.Auth
}

// New creates the OAuth plugin. Register providers via Options.Providers.
func New(opts Options) *Plugin {
	opts.applyDefaults()
	if opts.Providers == nil {
		opts.Providers = map[string]Provider{}
	}
	return &Plugin{opts: opts}
}

func (p *Plugin) ID() string { return "oauth" }

func (p *Plugin) SetAuth(a *goten.Auth) { p.auth = a }

func (p *Plugin) Schema() map[string]goten.TableSchema {
	return map[string]goten.TableSchema{
		"accounts": {
			Fields: []goten.FieldDef{
				{Name: "access_token", Type: "text"},
				{Name: "refresh_token", Type: "text"},
				{Name: "id_token", Type: "text"},
				{Name: "access_token_expires_at", Type: "timestamp"},
				{Name: "refresh_token_expires_at", Type: "timestamp"},
				{Name: "scope", Type: "text"},
			},
		},
	}
}

func (p *Plugin) Endpoints() []goten.Endpoint {
	return []goten.Endpoint{
		{Method: "POST", Path: "/sign-in/social", Handler: p.handleSignInSocial},
		{Method: "GET", Path: "/callback/{provider}", Handler: p.handleCallback},
		{Method: "GET", Path: "/list-accounts", Handler: p.handleListAccounts},
		{Method: "POST", Path: "/link-social", Handler: p.handleLinkSocial},
		{Method: "POST", Path: "/unlink-account", Handler: p.handleUnlinkAccount},
		{Method: "POST", Path: "/refresh-token", Handler: p.handleRefreshToken},
		{Method: "POST", Path: "/get-access-token", Handler: p.handleGetAccessToken},
	}
}

// provider looks up a registered provider by id.
func (p *Plugin) provider(id string) (Provider, bool) {
	pr, ok := p.opts.Providers[id]
	return pr, ok
}

// Compile-time interface checks.
var (
	_ goten.Plugin           = (*Plugin)(nil)
	_ goten.AuthAware        = (*Plugin)(nil)
	_ goten.SchemaProvider   = (*Plugin)(nil)
	_ goten.EndpointProvider = (*Plugin)(nil)
)
