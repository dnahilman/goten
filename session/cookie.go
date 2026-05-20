package session

import (
	"net/http"
	"strings"
	"time"
)

// CookieConfig holds cookie settings. HTTPOnly defaults to true when using DefaultCookieConfig.
type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

// DefaultCookieConfig returns a CookieConfig with secure defaults.
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     "goten_session",
		Path:     "/",
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func SetCookie(w http.ResponseWriter, cfg CookieConfig, token string, expiresAt time.Time) {
	name := cfg.Name
	if name == "" {
		name = "goten_session"
	}
	path := cfg.Path
	if path == "" {
		path = "/"
	}
	sameSite := cfg.SameSite
	if sameSite == 0 {
		sameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     path,
		Domain:   cfg.Domain,
		Expires:  expiresAt,
		Secure:   cfg.Secure,
		HttpOnly: cfg.HTTPOnly,
		SameSite: sameSite,
	})
}

func ClearCookie(w http.ResponseWriter, cfg CookieConfig) {
	name := cfg.Name
	if name == "" {
		name = "goten_session"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// GetSessionToken extracts the session token from the request.
// Priority: cookie → Authorization: Bearer <token>.
func GetSessionToken(r *http.Request, cookieName string) string {
	if cookieName == "" {
		cookieName = "goten_session"
	}
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimPrefix(authz, "Bearer ")
	}
	return ""
}
