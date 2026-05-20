package goten

import (
	"net/http"

	"github.com/dnahilman/goten/internal/httputil"
	"github.com/dnahilman/goten/session"
)

func (a *Auth) handleGetSession(w http.ResponseWriter, r *http.Request) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	ctx := r.Context()
	sess, err := a.sessions.Validate(ctx, token)
	if err != nil {
		session.ClearCookie(w, a.cookieConfig())
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	user, err := a.ia.FindUserByID(ctx, sess.UserID)
	if err != nil || user == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found")
		return
	}
	// Refresh cookie if session was extended by sliding refresh.
	session.SetCookie(w, a.cookieConfig(), sess.Token, sess.ExpiresAt)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "session": sess})
}

func (a *Auth) handleListSessions(w http.ResponseWriter, r *http.Request) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	ctx := r.Context()
	sess, err := a.sessions.Validate(ctx, token)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	sessions, err := a.sessions.ListByUserID(ctx, sess.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (a *Auth) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	ctx := r.Context()
	current, err := a.sessions.Validate(ctx, token)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil || req.SessionID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "sessionId required")
		return
	}
	// Fetch target to verify ownership.
	targets, err := a.sessions.ListByUserID(ctx, current.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	var found bool
	for _, s := range targets {
		if s.ID == req.SessionID {
			found = true
			break
		}
	}
	if !found {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session not found")
		return
	}
	if err := a.sessions.RevokeByID(ctx, req.SessionID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (a *Auth) handleRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	ctx := r.Context()
	current, err := a.sessions.Validate(ctx, token)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}
	if err := a.sessions.RevokeAllForUser(ctx, current.UserID, token); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}
