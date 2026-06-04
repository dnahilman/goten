package oauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dnahilman/goten/crypto"
)

type linkData struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
}

// stateData is persisted (as JSON) in the verification table and mirrored by a
// signed nonce cookie. Mirrors better-auth's StateData.
type stateData struct {
	CallbackURL   string    `json:"callbackURL"`
	CodeVerifier  string    `json:"codeVerifier"`
	ErrorURL      string    `json:"errorURL,omitempty"`
	NewUserURL    string    `json:"newUserURL,omitempty"`
	Link          *linkData `json:"link,omitempty"`
	RequestSignUp bool      `json:"requestSignUp,omitempty"`
	ExpiresAt     int64     `json:"expiresAt"`
	OAuthState    string    `json:"oauthState"`
}

// generateState fills in codeVerifier/state/expiry on sd, persists it to the
// verification table, and sets the signed state cookie. Returns the state token
// (for the authorization URL) and the code verifier (for PKCE).
func (p *Plugin) generateState(w http.ResponseWriter, r *http.Request, sd stateData) (state, codeVerifier string, err error) {
	codeVerifier = GenerateCodeVerifier()
	state = generateStateToken()
	sd.CodeVerifier = codeVerifier
	sd.OAuthState = state
	expiresAt := time.Now().UTC().Add(p.opts.StateTTL)
	sd.ExpiresAt = expiresAt.Unix()

	payload, err := json.Marshal(sd)
	if err != nil {
		return "", "", err
	}
	if _, err := p.auth.InternalAdapter().CreateVerificationValue(r.Context(), state, string(payload), expiresAt); err != nil {
		return "", "", err
	}
	p.setStateCookie(w, crypto.Sign(state, p.secret()))
	return state, codeVerifier, nil
}

// parseState validates the callback's state against the verification record and
// the signed cookie, then consumes (deletes) the record. Returns the state data.
func (p *Plugin) parseState(w http.ResponseWriter, r *http.Request) (*stateData, error) {
	state := callbackParam(r, "state")
	if state == "" {
		return nil, ErrStateInvalid
	}

	ctx := r.Context()
	rec, err := p.auth.InternalAdapter().FindVerificationValue(ctx, state)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrStateInvalid
	}

	var sd stateData
	if err := json.Unmarshal([]byte(rec.Value), &sd); err != nil {
		return nil, ErrStateInvalid
	}

	// Cross-check the signed cookie nonce against the state param.
	cookie, err := r.Cookie(p.opts.CookieName)
	if err != nil {
		return nil, ErrStateInvalid
	}
	signedValue, ok := crypto.Verify(cookie.Value, p.secret())
	if !ok || signedValue != state || sd.OAuthState != state {
		return nil, ErrStateInvalid
	}

	if time.Now().UTC().After(time.Unix(sd.ExpiresAt, 0)) {
		_ = p.auth.InternalAdapter().DeleteVerificationByIdentifier(ctx, state)
		p.clearStateCookie(w)
		return nil, ErrStateInvalid
	}

	// One-time use: consume the record and cookie.
	_ = p.auth.InternalAdapter().DeleteVerificationByIdentifier(ctx, state)
	p.clearStateCookie(w)
	return &sd, nil
}

func (p *Plugin) secret() string { return p.auth.Config().Secret }

func (p *Plugin) setStateCookie(w http.ResponseWriter, value string) {
	cfg := p.auth.Config().Cookie
	secure := false
	if cfg.Secure != nil {
		secure = *cfg.Secure
	}
	path := cfg.Path
	if path == "" {
		path = "/"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     p.opts.CookieName,
		Value:    value,
		Path:     path,
		Domain:   cfg.Domain,
		MaxAge:   int(p.opts.StateTTL.Seconds()),
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (p *Plugin) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     p.opts.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// callbackParam reads a parameter from the callback request (query first, then
// form body for response_mode=form_post providers).
func callbackParam(r *http.Request, key string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		return r.PostFormValue(key)
	}
	return ""
}
