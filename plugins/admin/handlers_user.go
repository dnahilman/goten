package adminplugin

import (
	"context"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/crypto"
	"github.com/dnahilman/goten/plugins/admin/access"
)

func (p *Plugin) handleSetRole(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"set-role"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to set roles")
		return
	}
	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" || req.Role == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId and role are required")
		return
	}
	for _, name := range splitAndTrim(req.Role) {
		if _, ok := p.opts.Roles[name]; !ok {
			goten.WriteError(w, http.StatusBadRequest, codeInvalidRole, "unknown role: "+name)
			return
		}
	}
	ctx := r.Context()
	if _, err := p.auth.InternalAdapter().UpdateUser(ctx, req.UserID, map[string]any{"role": req.Role}); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.writeUser(w, ctx, req.UserID)
}

func (p *Plugin) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"create"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to create users")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "invalid request body")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if _, err := mail.ParseAddress(req.Email); err != nil {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidInput, "valid email is required")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidInput, "password must be 8-72 characters")
		return
	}
	role := req.Role
	if role == "" {
		role = p.opts.DefaultRole
	}
	for _, name := range splitAndTrim(role) {
		if _, ok := p.opts.Roles[name]; !ok {
			goten.WriteError(w, http.StatusBadRequest, codeInvalidRole, "unknown role: "+name)
			return
		}
	}

	ctx := r.Context()
	ia := p.auth.InternalAdapter()
	if existing, err := ia.FindUserByEmail(ctx, req.Email); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	} else if existing != nil {
		goten.WriteError(w, http.StatusConflict, codeUserExists, "a user with this email already exists")
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}

	var userID string
	err = ia.WithTransaction(ctx, func(txCtx context.Context) error {
		u, err := ia.CreateUserWithExtra(txCtx, req.Email, req.Name, false, map[string]any{"role": role})
		if err != nil {
			return err
		}
		if _, err := ia.CreateAccount(txCtx, u.ID, u.ID, "credential", map[string]any{"password": hash}); err != nil {
			return err
		}
		userID = u.ID
		return nil
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.writeUser(w, ctx, userID)
}

func (p *Plugin) handleGetUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"get"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to read users")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	p.writeUser(w, r.Context(), req.UserID)
}

func (p *Plugin) handleListUsers(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"list"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to list users")
		return
	}
	ctx := r.Context()
	query := r.URL.Query()
	q := goten.Query{}
	if v, err := strconv.Atoi(query.Get("limit")); err == nil && v > 0 {
		q.Limit = v
	}
	if v, err := strconv.Atoi(query.Get("offset")); err == nil && v > 0 {
		q.Offset = v
	}
	if v := query.Get("sortBy"); v != "" {
		q.SortBy = v
		q.SortDir = query.Get("sortDir")
	}
	if v := strings.TrimSpace(query.Get("search")); v != "" {
		q.Where = append(q.Where, goten.Where{Field: "email", Operator: "like", Value: "%" + v + "%"})
	}
	rows, err := p.auth.Adapter().FindMany(ctx, "users", q)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	total, err := p.auth.Adapter().Count(ctx, "users", goten.Query{Where: q.Where})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{
		"users":  rows,
		"total":  total,
		"limit":  q.Limit,
		"offset": q.Offset,
	})
}

func (p *Plugin) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"update"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to update users")
		return
	}
	var req struct {
		UserID string         `json:"userId"`
		Data   map[string]any `json:"data"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	// Whitelist non-sensitive fields. Role goes through set-role; ban fields go
	// through ban/unban; password through set-user-password.
	allowed := map[string]bool{"name": true, "image": true, "email": true}
	upd := map[string]any{}
	for k, v := range req.Data {
		if allowed[k] {
			upd[k] = v
		}
	}
	if len(upd) == 0 {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidInput, "no updatable fields (allowed: name, image, email)")
		return
	}
	ctx := r.Context()
	if _, err := p.auth.InternalAdapter().UpdateUser(ctx, req.UserID, upd); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	p.writeUser(w, ctx, req.UserID)
}

func (p *Plugin) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"set-password"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to set passwords")
		return
	}
	var req struct {
		UserID      string `json:"userId"`
		NewPassword string `json:"newPassword"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidInput, "password must be 8-72 characters")
		return
	}
	hash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if err := p.auth.InternalAdapter().UpdatePassword(r.Context(), req.UserID, hash); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (p *Plugin) handleRemoveUser(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	if !p.hasPermission(c.userID, c.role, access.Statements{"user": {"delete"}}) {
		goten.WriteError(w, http.StatusForbidden, codeForbidden, "not allowed to delete users")
		return
	}
	var req struct {
		UserID string `json:"userId"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.UserID == "" {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "userId is required")
		return
	}
	if req.UserID == c.userID {
		goten.WriteError(w, http.StatusBadRequest, codeCannotDelSelf, "you cannot delete yourself")
		return
	}
	ctx := r.Context()
	if err := p.auth.Sessions().RevokeAllForUser(ctx, req.UserID, ""); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if err := p.auth.InternalAdapter().DeleteUser(ctx, req.UserID); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

// writeUser fetches a user's raw row and writes it, or 404 when missing.
func (p *Plugin) writeUser(w http.ResponseWriter, ctx context.Context, userID string) {
	rec, err := p.userRecord(ctx, userID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, codeInternal, "internal error")
		return
	}
	if rec == nil {
		goten.WriteError(w, http.StatusNotFound, codeUserNotFound, "user not found")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"user": rec})
}
