// Package goten provides composable, self-hosted authentication for Go.
package goten

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dnahilman/goten/models"
	"github.com/dnahilman/goten/session"
)

// Auth is the composition root. Create with New() and mount Handler() on your server.
type Auth struct {
	cfg      Config
	ia       *InternalAdapter
	sessions *session.Manager
	handler  http.Handler

	plugins            []Plugin
	userCreateHooks    []UserCreateHookFn
	sessionCreateHooks []SessionCreateHookFn
}

// New creates an Auth instance. Returns error for invalid config — never panics.
func New(cfg Config) (*Auth, error) {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("goten: %w", err)
	}
	a := &Auth{
		cfg:     cfg,
		plugins: cfg.Plugins,
		ia:      NewInternalAdapter(cfg.Adapter),
	}
	a.sessions = session.NewManager(cfg.Adapter, session.Config{
		ExpiresIn: cfg.Session.ExpiresIn,
		UpdateAge: cfg.Session.UpdateAge,
	})

	// Plugin lifecycle: SetAuth → collect hooks → Init → buildRouter
	for _, p := range a.plugins {
		if aware, ok := p.(AuthAware); ok {
			aware.SetAuth(a)
		}
	}
	for _, p := range a.plugins {
		if uh, ok := p.(UserCreateHookProvider); ok {
			a.userCreateHooks = append(a.userCreateHooks, uh.UserCreateHooks()...)
		}
		if sh, ok := p.(SessionCreateHookProvider); ok {
			a.sessionCreateHooks = append(a.sessionCreateHooks, sh.SessionCreateHooks()...)
		}
	}
	for _, p := range a.plugins {
		if init, ok := p.(Initializer); ok {
			if err := init.Init(); err != nil {
				return nil, fmt.Errorf("goten: plugin %q init failed: %w", p.ID(), err)
			}
		}
	}

	a.handler = a.buildRouter()
	return a, nil
}

// RunUserCreateHooks applies all user-create hooks to data in registration order.
func (a *Auth) RunUserCreateHooks(data map[string]any) map[string]any {
	for _, h := range a.userCreateHooks {
		data = h(data)
	}
	return data
}

// RunSessionCreateHooks runs all session-create veto hooks.
// Returns ErrHookHandled if a hook already wrote the response (caller must not write again).
func (a *Auth) RunSessionCreateHooks(w http.ResponseWriter, r *http.Request, userID string) error {
	for _, h := range a.sessionCreateHooks {
		if err := h(SessionCreateContext{UserID: userID, Request: r, Writer: w}); err != nil {
			return err
		}
	}
	return nil
}

func (a *Auth) Handler() http.Handler             { return a.handler }
func (a *Auth) Config() Config                    { return a.cfg }
func (a *Auth) Adapter() Adapter                  { return a.cfg.Adapter }
func (a *Auth) InternalAdapter() *InternalAdapter { return a.ia }
func (a *Auth) Sessions() *session.Manager        { return a.sessions }
func (a *Auth) Plugins() []Plugin                 { return a.plugins }

// CurrentSession resolves the active session and user from the request
// (session cookie or Bearer token), applying sliding refresh. It is a
// plugin-friendly helper for endpoints that need the caller's identity without
// going through the RequireAuth middleware. Returns an error when there is no
// valid session.
func (a *Auth) CurrentSession(r *http.Request) (*models.Session, *models.User, error) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token == "" {
		return nil, nil, ErrNoSession
	}
	ctx := r.Context()
	sess, err := a.sessions.Validate(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	user, err := a.ia.FindUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrUserNotFound
	}
	return sess, user, nil
}

// IsTrustedOrigin reports whether origin matches the configured BaseURL or one
// of the TrustedOrigins. Exposed so plugins can validate redirect/callback URLs
// (e.g. to prevent open-redirects in the OAuth flow).
func (a *Auth) IsTrustedOrigin(origin string) bool {
	return a.isTrustedOrigin(origin)
}

// SetSessionCookie is a plugin-friendly helper that sets the session cookie
// using Auth's configured cookie settings. Plugins call this instead of
// accessing the internal cookieConfig directly.
func (a *Auth) SetSessionCookie(w http.ResponseWriter, sess *models.Session) {
	session.SetCookie(w, a.cookieConfig(), sess.Token, sess.ExpiresAt)
}

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

// isHookHandled is a helper so callers don't need to import errors themselves.
func isHookHandled(err error) bool { return errors.Is(err, ErrHookHandled) }
