package goten

import (
	"net/http"
	"strings"

	"github.com/dnahilman/goten/internal/httputil"
	"github.com/dnahilman/goten/session"
)

// RequireAuth middleware validates the session and injects user+session into context.
// Use this to protect your own application endpoints.
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := session.GetSessionToken(r, a.cfg.Cookie.Name)
		if token == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
			return
		}
		sess, err := a.sessions.Validate(r.Context(), token)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			return
		}
		user, err := a.ia.FindUserByID(r.Context(), sess.UserID)
		if err != nil || user == nil {
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found")
			return
		}
		ctx := WithSession(r.Context(), sess, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) middlewareOriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods and Bearer requests bypass origin check.
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}
		// No origin = direct request (curl, server-side) — allow.
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = r.Header.Get("Referer")
		}
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !a.isTrustedOrigin(origin) {
			httputil.WriteError(w, http.StatusForbidden, "ORIGIN_DENIED", "origin not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) isTrustedOrigin(origin string) bool {
	if origin == a.cfg.BaseURL {
		return true
	}
	for _, o := range a.cfg.TrustedOrigins {
		if o == origin {
			return true
		}
	}
	return false
}
