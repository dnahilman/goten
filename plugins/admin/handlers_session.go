package adminplugin

import (
	"net/http"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/plugins/admin/access"
)

func (p *Plugin) handleListUserSessions(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"session": {"list"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to list sessions")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	sessions, err := p.auth.Sessions().ListByUserID(r.Context(), req.UserID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (p *Plugin) handleRevokeUserSession(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"session": {"revoke"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to revoke sessions")
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.SessionID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "sessionId is required")
		return
	}
	if err := p.auth.Sessions().RevokeByID(r.Context(), req.SessionID); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (p *Plugin) handleRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"session": {"revoke"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to revoke sessions")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	if err := p.auth.Sessions().RevokeAllForUser(r.Context(), req.UserID, ""); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}
