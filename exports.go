package goten

// Re-exports for plugin authors who need access to internal helpers
// without importing the internal/ package directly.

import (
	"net/http"

	"github.com/dnahilman/goten/internal"
	"github.com/dnahilman/goten/internal/httputil"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	httputil.WriteJSON(w, status, v)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	httputil.WriteError(w, status, code, message)
}

func GetClientIP(r *http.Request, ipHeader string) string {
	return internal.GetClientIP(r, ipHeader)
}
