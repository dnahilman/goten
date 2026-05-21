package goten

import (
	"net/http"
	"strings"

	"github.com/dnahilman/goten/crypto"
	"github.com/dnahilman/goten/internal"
	"github.com/dnahilman/goten/internal/httputil"
	"github.com/dnahilman/goten/internal/validate"
	"github.com/dnahilman/goten/session"
)

// dummyHash is a pre-computed bcrypt hash used for anti-enumeration timing.
var dummyHash string

func init() {
	h, _ := crypto.HashPassword("goten-anti-enum-dummy-please-ignore-2026")
	dummyHash = h
}

func (a *Auth) handleSignUpEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !validate.IsValidEmail(req.Email) {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_EMAIL", "invalid email")
		return
	}
	ep := a.cfg.EmailPassword
	if err := validate.Password(req.Password, ep.MinPasswordLength, ep.MaxPasswordLength); err != nil {
		if len(req.Password) < ep.MinPasswordLength {
			httputil.WriteError(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", err.Error())
		} else {
			httputil.WriteError(w, http.StatusBadRequest, "PASSWORD_TOO_LONG", err.Error())
		}
		return
	}

	ctx := r.Context()
	existing, err := a.ia.FindUserByEmail(ctx, req.Email)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if existing != nil {
		httputil.WriteError(w, http.StatusConflict, "EMAIL_EXISTS", "email already exists")
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	extra := a.RunUserCreateHooks(map[string]any{})
	user, err := a.ia.CreateUserWithExtra(ctx, req.Email, req.Name, false, extra)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if _, err = a.ia.CreateAccount(ctx, user.ID, user.ID, "credential", map[string]any{
		"password": hash,
	}); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	if ep.AutoSignIn {
		if err := a.RunSessionCreateHooks(w, r, user.ID); err != nil {
			if !isHookHandled(err) {
				httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			}
			return
		}
		sess, err := a.sessions.Create(ctx, user.ID,
			internal.GetClientIP(r, ""),
			r.UserAgent(),
		)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
		session.SetCookie(w, a.cookieConfig(), sess.Token, sess.ExpiresAt)
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "session": sess})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (a *Auth) handleSignInEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	ctx := r.Context()
	user, err := a.ia.FindUserByEmail(ctx, req.Email)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	// Anti-enumeration: always do a bcrypt operation even when user not found.
	if user == nil {
		_, _ = crypto.VerifyPassword(dummyHash, req.Password)
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	accounts, err := a.ia.FindAccountsByUserID(ctx, user.ID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
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
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	ok, err := crypto.VerifyPassword(hash, req.Password)
	if err != nil || !ok {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	if err := a.RunSessionCreateHooks(w, r, user.ID); err != nil {
		if !isHookHandled(err) {
			httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		}
		return
	}

	sess, err := a.sessions.Create(ctx, user.ID,
		internal.GetClientIP(r, ""),
		r.UserAgent(),
	)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	session.SetCookie(w, a.cookieConfig(), sess.Token, sess.ExpiresAt)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "session": sess})
}

func (a *Auth) handleSignOut(w http.ResponseWriter, r *http.Request) {
	token := session.GetSessionToken(r, a.cfg.Cookie.Name)
	if token != "" {
		_ = a.sessions.Revoke(r.Context(), token)
	}
	session.ClearCookie(w, a.cookieConfig())
	httputil.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}
