// Package goten provides composable, self-hosted authentication for Go.
package goten

import (
	"fmt"
	"net/http"

	"github.com/dnahilman/goten/session"
)

// Auth is the composition root. Create with New() and mount Handler() on your server.
type Auth struct {
	cfg      Config
	ia       *InternalAdapter
	sessions *session.Manager
	handler  http.Handler
}

// New creates an Auth instance. Returns error for invalid config — never panics.
func New(cfg Config) (*Auth, error) {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("goten: %w", err)
	}
	a := &Auth{
		cfg: cfg,
		ia:  NewInternalAdapter(cfg.Adapter),
	}
	a.sessions = session.NewManager(cfg.Adapter, session.Config{
		ExpiresIn: cfg.Session.ExpiresIn,
		UpdateAge: cfg.Session.UpdateAge,
	})
	a.handler = a.buildRouter()
	return a, nil
}

func (a *Auth) Handler() http.Handler          { return a.handler }
func (a *Auth) Config() Config                 { return a.cfg }
func (a *Auth) Adapter() Adapter               { return a.cfg.Adapter }
func (a *Auth) InternalAdapter() *InternalAdapter { return a.ia }
func (a *Auth) Sessions() *session.Manager     { return a.sessions }

func (a *Auth) cookieConfig() session.CookieConfig {
	return session.CookieConfig{
		Name:     a.cfg.Cookie.Name,
		Domain:   a.cfg.Cookie.Domain,
		Path:     a.cfg.Cookie.Path,
		Secure:   *a.cfg.Cookie.Secure,
		HTTPOnly: *a.cfg.Cookie.HTTPOnly,
		SameSite: a.cfg.Cookie.SameSite,
	}
}
