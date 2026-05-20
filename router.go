package goten

import (
	"net/http"
	"strings"
)

func (a *Auth) buildRouter() http.Handler {
	mux := http.NewServeMux()
	bp := strings.TrimRight(a.cfg.BasePath, "/")

	mux.HandleFunc("POST "+bp+"/sign-up/email", a.handleSignUpEmail)
	mux.HandleFunc("POST "+bp+"/sign-in/email", a.handleSignInEmail)
	mux.HandleFunc("POST "+bp+"/sign-out", a.handleSignOut)

	mux.HandleFunc("GET "+bp+"/get-session", a.handleGetSession)
	mux.HandleFunc("POST "+bp+"/get-session", a.handleGetSession)
	mux.HandleFunc("GET "+bp+"/list-sessions", a.handleListSessions)
	mux.HandleFunc("POST "+bp+"/revoke-session", a.handleRevokeSession)
	mux.HandleFunc("POST "+bp+"/revoke-other-sessions", a.handleRevokeOtherSessions)

	return a.middlewareOriginCheck(mux)
}
