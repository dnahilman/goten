package usernameplugin

import (
	"context"
	"errors"
	"net/http"
	"strings"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/crypto"
)

// dummyHash is pre-computed for anti-enumeration timing on sign-in.
var dummyHash string

func init() {
	h, _ := crypto.HashPassword("dummy-anti-enum-username-plugin-2026")
	dummyHash = h
}

func (p *Plugin) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	if err := p.validate.Var(req.Username, "required,username"); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_USERNAME",
			"username must match "+p.opts.Regex.String())
		return
	}
	if err := p.validate.Var(req.Password, "required,min=8,max=72"); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_PASSWORD", "password must be 8-72 characters")
		return
	}

	ctx := r.Context()
	ia := p.auth.InternalAdapter()

	existing, err := ia.Adapter().FindOne(ctx, "users", goten.Query{
		Where: []goten.Where{goten.EQ("username", req.Username)},
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if existing != nil {
		goten.WriteError(w, http.StatusConflict, "USERNAME_EXISTS", "username already taken")
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	// Email is synthetic: username auth does not require an email address.
	// We use a reserved TLD (.invalid per RFC 6761) to avoid conflicts.
	syntheticEmail := req.Username + "@username.local.invalid"

	var user *goten.User
	err = ia.WithTransaction(ctx, func(txCtx context.Context) error {
		u, err := ia.CreateUserWithExtra(txCtx, syntheticEmail, req.Name, false, map[string]any{
			"username": req.Username,
		})
		if err != nil {
			return err
		}
		if _, err := ia.CreateAccount(txCtx, u.ID, u.ID, "credential", map[string]any{
			"password": hash,
		}); err != nil {
			return err
		}
		user = u
		return nil
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	if err := p.auth.RunSessionCreateHooks(w, r, user.ID); err != nil {
		if !errors.Is(err, goten.ErrHookHandled) {
			goten.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		}
		return
	}

	sess, err := p.auth.Sessions().Create(ctx, user.ID,
		goten.GetClientIP(r, ""), r.UserAgent())
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	p.auth.SetSessionCookie(w, sess)
	goten.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "session": sess})
}

func (p *Plugin) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	ctx := r.Context()
	ia := p.auth.InternalAdapter()

	rec, err := ia.Adapter().FindOne(ctx, "users", goten.Query{
		Where: []goten.Where{goten.EQ("username", req.Username)},
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	// Anti-enumeration: always run bcrypt even when user not found.
	if rec == nil {
		_, _ = crypto.VerifyPassword(dummyHash, req.Password)
		goten.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	userID, _ := rec["id"].(string)
	user, err := ia.FindUserByID(ctx, userID)
	if err != nil || user == nil {
		_, _ = crypto.VerifyPassword(dummyHash, req.Password)
		goten.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	accounts, err := ia.FindAccountsByUserID(ctx, user.ID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	var hash string
	for _, acc := range accounts {
		if acc.ProviderID == "credential" && acc.Password != nil {
			hash = *acc.Password
			break
		}
	}
	if hash == "" {
		_, _ = crypto.VerifyPassword(dummyHash, req.Password)
		goten.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	ok, err := crypto.VerifyPassword(hash, req.Password)
	if err != nil || !ok {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	if err := p.auth.RunSessionCreateHooks(w, r, user.ID); err != nil {
		if !errors.Is(err, goten.ErrHookHandled) {
			goten.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		}
		return
	}

	sess, err := p.auth.Sessions().Create(ctx, user.ID,
		goten.GetClientIP(r, ""), r.UserAgent())
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	p.auth.SetSessionCookie(w, sess)
	goten.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "session": sess})
}
