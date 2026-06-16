package adminplugin

import (
	"net/http"
	"time"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/plugins/admin/access"
)

func (p *Plugin) handleBanUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"ban"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to ban users")
		return
	}
	var req struct {
		UserID       string `json:"userId"`
		BanReason    string `json:"banReason"`
		BanExpiresIn int64  `json:"banExpiresIn"` // seconds; 0 = use default/permanent
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	if req.UserID == c.userID {
		goten.WriteError(w, http.StatusBadRequest, codeCannotBanSelf, "you cannot ban yourself")
		return
	}

	ctx := r.Context()
	if rec, err := p.userRecord(ctx, req.UserID); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	} else if rec == nil {
		goten.WriteError(w, http.StatusNotFound, codeUserNotFound, "user not found")
		return
	}

	reason := req.BanReason
	if reason == "" {
		reason = p.opts.DefaultBanReason
	}
	data := map[string]any{"banned": true, "ban_reason": reason}
	switch {
	case req.BanExpiresIn > 0:
		data["ban_expires"] = time.Now().UTC().Add(time.Duration(req.BanExpiresIn) * time.Second)
	case p.opts.DefaultBanExpiresIn > 0:
		data["ban_expires"] = time.Now().UTC().Add(p.opts.DefaultBanExpiresIn)
	default:
		data["ban_expires"] = nil // permanent
	}

	if _, err := p.auth.InternalAdapter().UpdateUser(ctx, req.UserID, data); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	// Banned users are logged out immediately.
	if err := p.auth.Sessions().RevokeAllForUser(ctx, req.UserID, ""); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.writeUser(w, ctx, req.UserID)
}

func (p *Plugin) handleUnbanUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"ban"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to unban users")
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
	if _, err := p.auth.InternalAdapter().UpdateUser(ctx, req.UserID, map[string]any{
		"banned":      false,
		"ban_reason":  nil,
		"ban_expires": nil,
	}); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.writeUser(w, ctx, req.UserID)
}
