package adminplugin

import (
	"net/http"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/plugins/admin/access"
)

// handleImpersonate signs the caller in as another user. The impersonation
// session records the admin's id in impersonated_by.
//
// NOTE: unlike better-auth (which stashes the admin's original session in a
// signed admin_session cookie), goten exposes no signed-cookie helper to
// plugins, so the admin's original session is left intact and
// stop-impersonating mints a fresh admin session. See the plugin README.
func (p *Plugin) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"impersonate"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to impersonate users")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	ctx := r.Context()
	target, err := p.userRecord(ctx, req.UserID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if target == nil {
		goten.WriteError(w, http.StatusNotFound, codeUserNotFound, "user not found")
		return
	}
	targetRole, _ := target["role"].(string)
	if p.isAdmin(req.UserID, targetRole) {
		if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"impersonate-admins"}}) {
			goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to impersonate admins")
			return
		}
	}

	sess, err := p.auth.Sessions().Create(ctx, req.UserID, goten.GetClientIP(r, ""), r.UserAgent())
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if _, err := p.auth.Adapter().Update(ctx, "sessions",
		goten.Query{Where: []goten.Where{goten.EQ("id", sess.ID)}},
		map[string]any{"impersonated_by": c.userID},
	); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.auth.SetSessionCookie(w, sess)
	goten.WriteJSON(w, http.StatusOK, map[string]any{"session": sess, "user": target})
}

// handleStopImpersonating ends an impersonation session and mints a fresh
// session for the original admin (read from impersonated_by).
func (p *Plugin) handleStopImpersonating(w http.ResponseWriter, r *http.Request) {
	sess, _, err := p.auth.CurrentSession(r)
	if err != nil || sess == nil {
		goten.WriteError(w, http.StatusUnauthorized, codeUnauthorized, "authentication required")
		return
	}
	ctx := r.Context()
	rec, err := p.auth.Adapter().FindOne(ctx, "sessions",
		goten.Query{Where: []goten.Where{goten.EQ("id", sess.ID)}})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	adminID := ""
	if rec != nil {
		adminID, _ = rec["impersonated_by"].(string)
	}
	if adminID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeNotImpersonate, "not impersonating")
		return
	}
	// Revoke the impersonation session and mint a fresh admin session.
	if err := p.auth.Sessions().RevokeByID(ctx, sess.ID); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	adminSess, err := p.auth.Sessions().Create(ctx, adminID, goten.GetClientIP(r, ""), r.UserAgent())
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.auth.SetSessionCookie(w, adminSess)
	admin, _ := p.userRecord(ctx, adminID)
	goten.WriteJSON(w, http.StatusOK, map[string]any{"session": adminSess, "user": admin})
}
