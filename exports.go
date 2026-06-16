package goten

// Re-exports for plugin authors who need access to internal helpers
// without importing the internal/ package directly.

import (
	"net/http"

	"github.com/dnahilman/goten/internal"
	"github.com/dnahilman/goten/internal/httputil"
	"github.com/dnahilman/goten/models"
)

// Type aliases so plugin authors only need to import the root package.
type User = models.User
type Session = models.Session
type Account = models.Account

func DecodeJSON(r *http.Request, dst any) error {
	return httputil.DecodeJSON(r, dst)
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	httputil.WriteJSON(w, status, v)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	httputil.WriteError(w, status, code, message)
}

// GetClientIP resolves the client IP, preferring ipHeader (a header you control
// at the edge) and otherwise falling back to X-Forwarded-For / X-Real-IP /
// CF-Connecting-IP (left-most) then RemoteAddr. The value is used only to record
// a session's ip_address; configure your reverse proxy to OVERWRITE
// X-Forwarded-For so the left-most entry cannot be spoofed. See the package docs
// and SECURITY.md ("Reverse proxy & client IP") for details.
func GetClientIP(r *http.Request, ipHeader string) string {
	return internal.GetClientIP(r, ipHeader)
}
