package handler

import (
	"net/http"

	goten "github.com/dnahilman/goten"
	"github.com/gin-gonic/gin"
)

// RequireAuth wraps Goten's RequireAuth as Gin middleware. It propagates the
// user/session that Goten injects into the request context back into c.Request,
// so downstream Gin handlers can read them via AuthUserID / goten.UserFromContext.
func RequireAuth(auth *goten.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		allow := false
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			allow = true
			c.Request = r
		})
		auth.RequireAuth(next).ServeHTTP(c.Writer, c.Request)
		if !allow {
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthUserID returns the authenticated user ID from the request context.
// Returns "" when no session is present (should not happen behind RequireAuth).
func AuthUserID(c *gin.Context) string {
	user, ok := goten.UserFromContext(c.Request.Context())
	if !ok {
		return ""
	}
	return user.ID
}
