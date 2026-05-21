package goten

import (
	"net/http"
	"net/url"
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

// middlewareOriginCheck protects non-safe methods from cross-site requests.
//
// Rules:
//   - GET/HEAD/OPTIONS: always pass through
//   - Bearer present: pass through (mobile / server-to-server clients)
//   - No Origin header and TrustedOrigins empty: pass through (dev / curl)
//   - No Origin header and TrustedOrigins set: reject 403 (strict mode)
//   - Origin present and trusted: pass through
//   - Origin present and not trusted: reject 403
func (a *Auth) middlewareOriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// Bearer token bypasses origin check (API / mobile clients).
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = extractOriginFromReferer(r.Header.Get("Referer"))
		}

		if origin == "" {
			// Strict mode: require Origin when TrustedOrigins is configured.
			if len(a.cfg.TrustedOrigins) > 0 {
				httputil.WriteError(w, http.StatusForbidden, "ORIGIN_REQUIRED",
					"Origin header is required for this request")
				return
			}
			// Permissive mode (dev): no origin and no trusted list → allow.
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

// extractOriginFromReferer parses the origin (scheme + host) out of a Referer URL.
// Returns empty string if the Referer is missing or invalid.
func extractOriginFromReferer(referer string) string {
	if referer == "" {
		return ""
	}
	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
