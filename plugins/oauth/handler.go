package oauth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	goten "github.com/dnahilman/goten"
)

// handleSignInSocial starts a social sign-in. With an idToken it verifies and
// signs in immediately ({redirect:false,...}); otherwise it returns an
// authorization URL for the browser to follow ({redirect:true,url}).
func (p *Plugin) handleSignInSocial(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider           string   `json:"provider"`
		CallbackURL        string   `json:"callbackURL"`
		ErrorCallbackURL   string   `json:"errorCallbackURL"`
		NewUserCallbackURL string   `json:"newUserCallbackURL"`
		RequestSignUp      bool     `json:"requestSignUp"`
		Scopes             []string `json:"scopes"`
		LoginHint          string   `json:"loginHint"`
		IDToken            *struct {
			Token        string `json:"token"`
			Nonce        string `json:"nonce"`
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"idToken"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	provider, ok := p.provider(req.Provider)
	if !ok {
		goten.WriteError(w, http.StatusNotFound, "PROVIDER_NOT_FOUND", "provider not found")
		return
	}

	// id_token branch (native / one-tap): verify and sign in directly.
	if req.IDToken != nil {
		verifier, ok := provider.(IDTokenVerifierProvider)
		if !ok {
			goten.WriteError(w, http.StatusBadRequest, "ID_TOKEN_NOT_SUPPORTED", "provider does not support id_token sign-in")
			return
		}
		valid, err := verifier.VerifyIDToken(req.IDToken.Token, req.IDToken.Nonce)
		if err != nil || !valid {
			goten.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid id_token")
			return
		}
		tokens := &Tokens{IDToken: req.IDToken.Token, AccessToken: req.IDToken.AccessToken, RefreshToken: req.IDToken.RefreshToken}
		info, err := provider.GetUserInfo(tokens)
		if err != nil || info == nil || info.ID == "" {
			goten.WriteError(w, http.StatusUnauthorized, "FAILED_TO_GET_USER_INFO", "unable to get user info")
			return
		}
		if info.Email == "" {
			goten.WriteError(w, http.StatusUnauthorized, "EMAIL_NOT_FOUND", "provider did not return an email")
			return
		}
		result, err := p.handleOAuthUserInfo(r.Context(), provider.ID(), info, tokens, providerDisablesSignUp(provider, req.RequestSignUp))
		if err != nil {
			goten.WriteError(w, http.StatusUnauthorized, "OAUTH_LINK_ERROR", err.Error())
			return
		}
		if err := p.auth.RunSessionCreateHooks(w, r, result.User.ID); err != nil {
			if !errors.Is(err, goten.ErrHookHandled) {
				goten.WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			}
			return
		}
		sess, err := p.auth.Sessions().Create(r.Context(), result.User.ID, goten.GetClientIP(r, ""), r.UserAgent())
		if err != nil {
			goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
		p.auth.SetSessionCookie(w, sess)
		goten.WriteJSON(w, http.StatusOK, map[string]any{"redirect": false, "token": sess.Token, "user": result.User})
		return
	}

	// redirect branch: build authorization URL.
	callbackURL, err := p.resolveCallback(req.CallbackURL)
	if err != nil {
		goten.WriteError(w, http.StatusForbidden, "UNTRUSTED_ORIGIN", "callbackURL is not a trusted origin")
		return
	}
	if req.ErrorCallbackURL != "" {
		if _, err := p.resolveCallback(req.ErrorCallbackURL); err != nil {
			goten.WriteError(w, http.StatusForbidden, "UNTRUSTED_ORIGIN", "errorCallbackURL is not a trusted origin")
			return
		}
	}

	sd := stateData{
		CallbackURL:   callbackURL,
		ErrorURL:      req.ErrorCallbackURL,
		NewUserURL:    req.NewUserCallbackURL,
		RequestSignUp: req.RequestSignUp,
	}
	state, codeVerifier, err := p.generateState(w, r, sd)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	authURL, err := provider.CreateAuthorizationURL(AuthURLParams{
		State:        state,
		CodeVerifier: codeVerifier,
		Scopes:       req.Scopes,
		RedirectURI:  p.callbackURI(provider.ID()),
		LoginHint:    req.LoginHint,
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"redirect": true, "url": authURL})
}

// handleCallback completes the OAuth flow: exchanges the code, links/creates the
// user, issues a session, and redirects back to the caller's callbackURL.
func (p *Plugin) handleCallback(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("provider")
	provider, ok := p.provider(providerID)
	if !ok {
		p.redirectError(w, r, p.defaultErrorURL(), "provider_not_found")
		return
	}

	sd, err := p.parseState(w, r)
	if err != nil {
		p.redirectError(w, r, p.defaultErrorURL(), "invalid_state")
		return
	}
	errorURL := sd.ErrorURL
	if errorURL == "" {
		errorURL = p.defaultErrorURL()
	}

	if e := callbackParam(r, "error"); e != "" {
		p.redirectError(w, r, errorURL, e)
		return
	}
	code := callbackParam(r, "code")
	if code == "" {
		p.redirectError(w, r, errorURL, "no_code")
		return
	}

	tokens, err := provider.ValidateAuthorizationCode(CodeExchangeParams{
		Code:         code,
		CodeVerifier: sd.CodeVerifier,
		RedirectURI:  p.callbackURI(providerID),
	})
	if err != nil {
		p.redirectError(w, r, errorURL, "invalid_code")
		return
	}
	info, err := provider.GetUserInfo(tokens)
	if err != nil || info == nil || info.ID == "" {
		p.redirectError(w, r, errorURL, "unable_to_get_user_info")
		return
	}
	if info.Email == "" {
		p.redirectError(w, r, errorURL, "email_not_found")
		return
	}

	// Link-to-current-user flow.
	if sd.Link != nil {
		user, err := p.auth.InternalAdapter().FindUserByID(r.Context(), sd.Link.UserID)
		if err != nil || user == nil {
			p.redirectError(w, r, errorURL, "unable_to_link_account")
			return
		}
		if err := p.linkToCurrentUser(r.Context(), user, providerID, info, tokens); err != nil {
			p.redirectError(w, r, errorURL, errCode(err))
			return
		}
		p.redirect(w, r, sd.CallbackURL)
		return
	}

	// Sign-in / sign-up flow.
	result, err := p.handleOAuthUserInfo(r.Context(), providerID, info, tokens, providerDisablesSignUp(provider, sd.RequestSignUp))
	if err != nil {
		p.redirectError(w, r, errorURL, errCode(err))
		return
	}
	if err := p.auth.RunSessionCreateHooks(w, r, result.User.ID); err != nil {
		if !errors.Is(err, goten.ErrHookHandled) {
			p.redirectError(w, r, errorURL, "forbidden")
		}
		return
	}
	sess, err := p.auth.Sessions().Create(r.Context(), result.User.ID, goten.GetClientIP(r, ""), r.UserAgent())
	if err != nil {
		p.redirectError(w, r, errorURL, "internal_error")
		return
	}
	p.auth.SetSessionCookie(w, sess)

	target := sd.CallbackURL
	if result.IsRegister && sd.NewUserURL != "" {
		target = sd.NewUserURL
	}
	p.redirect(w, r, target)
}

// handleListAccounts lists the linked accounts for the current user.
func (p *Plugin) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	_, user, err := p.auth.CurrentSession(r)
	if err != nil {
		goten.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	accounts, err := p.auth.InternalAdapter().FindAccountsByUserID(r.Context(), user.ID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	for _, a := range accounts {
		a.Password = nil // never expose credential hashes
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

// handleLinkSocial starts linking a provider account to the current user.
func (p *Plugin) handleLinkSocial(w http.ResponseWriter, r *http.Request) {
	_, user, err := p.auth.CurrentSession(r)
	if err != nil {
		goten.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	var req struct {
		Provider         string   `json:"provider"`
		CallbackURL      string   `json:"callbackURL"`
		ErrorCallbackURL string   `json:"errorCallbackURL"`
		Scopes           []string `json:"scopes"`
		LoginHint        string   `json:"loginHint"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	provider, ok := p.provider(req.Provider)
	if !ok {
		goten.WriteError(w, http.StatusNotFound, "PROVIDER_NOT_FOUND", "provider not found")
		return
	}
	callbackURL, err := p.resolveCallback(req.CallbackURL)
	if err != nil {
		goten.WriteError(w, http.StatusForbidden, "UNTRUSTED_ORIGIN", "callbackURL is not a trusted origin")
		return
	}

	sd := stateData{
		CallbackURL: callbackURL,
		ErrorURL:    req.ErrorCallbackURL,
		Link:        &linkData{UserID: user.ID, Email: user.Email},
	}
	state, codeVerifier, err := p.generateState(w, r, sd)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	authURL, err := provider.CreateAuthorizationURL(AuthURLParams{
		State:        state,
		CodeVerifier: codeVerifier,
		Scopes:       req.Scopes,
		RedirectURI:  p.callbackURI(provider.ID()),
		LoginHint:    req.LoginHint,
	})
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]any{"url": authURL})
}

// handleUnlinkAccount removes a linked provider account from the current user.
func (p *Plugin) handleUnlinkAccount(w http.ResponseWriter, r *http.Request) {
	_, user, err := p.auth.CurrentSession(r)
	if err != nil {
		goten.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return
	}
	var req struct {
		Provider string `json:"provider"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.Provider == "" {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "provider required")
		return
	}
	accounts, err := p.auth.InternalAdapter().FindAccountsByUserID(r.Context(), user.ID)
	if err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	var found bool
	for _, a := range accounts {
		if a.ProviderID == req.Provider {
			found = true
			break
		}
	}
	if !found {
		goten.WriteError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "account not found")
		return
	}
	if len(accounts) <= 1 {
		goten.WriteError(w, http.StatusBadRequest, "CANNOT_UNLINK_LAST", "cannot unlink the only login method")
		return
	}
	if err := p.auth.Adapter().Delete(r.Context(), "accounts", goten.Query{Where: []goten.Where{
		goten.EQ("user_id", user.ID),
		goten.EQ("provider_id", req.Provider),
	}}); err != nil {
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	goten.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleRefreshToken refreshes and persists the provider tokens for the user.
func (p *Plugin) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	user, provider, ok := p.authedProvider(w, r)
	if !ok {
		return
	}
	tokens, err := p.refreshAccessToken(r.Context(), provider, user.ID)
	if err != nil {
		p.writeTokenError(w, err)
		return
	}
	goten.WriteJSON(w, http.StatusOK, tokenJSON(tokens))
}

// handleGetAccessToken returns a valid access token, refreshing it if expired.
func (p *Plugin) handleGetAccessToken(w http.ResponseWriter, r *http.Request) {
	user, provider, ok := p.authedProvider(w, r)
	if !ok {
		return
	}
	tokens, err := p.getAccessToken(r.Context(), provider, user.ID)
	if err != nil {
		p.writeTokenError(w, err)
		return
	}
	goten.WriteJSON(w, http.StatusOK, tokenJSON(tokens))
}

// --- helpers ---

// authedProvider resolves the current user and the requested provider for the
// protected token endpoints, writing the error response itself on failure.
func (p *Plugin) authedProvider(w http.ResponseWriter, r *http.Request) (*goten.User, Provider, bool) {
	_, user, err := p.auth.CurrentSession(r)
	if err != nil {
		goten.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "no session")
		return nil, nil, false
	}
	var req struct {
		Provider string `json:"provider"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || req.Provider == "" {
		goten.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "provider required")
		return nil, nil, false
	}
	provider, ok := p.provider(req.Provider)
	if !ok {
		goten.WriteError(w, http.StatusNotFound, "PROVIDER_NOT_FOUND", "provider not found")
		return nil, nil, false
	}
	return user, provider, true
}

func (p *Plugin) writeTokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		goten.WriteError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "account not found")
	case errors.Is(err, ErrNoRefreshToken):
		goten.WriteError(w, http.StatusBadRequest, "NO_REFRESH_TOKEN", "no refresh token stored")
	case errors.Is(err, ErrRefreshNotSupported):
		goten.WriteError(w, http.StatusBadRequest, "REFRESH_NOT_SUPPORTED", "provider does not support refresh")
	default:
		goten.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

func tokenJSON(t *Tokens) map[string]any {
	out := map[string]any{
		"accessToken": t.AccessToken,
		"scopes":      t.Scopes,
	}
	if t.AccessTokenExpiresAt != nil {
		out["accessTokenExpiresAt"] = t.AccessTokenExpiresAt
	}
	if t.IDToken != "" {
		out["idToken"] = t.IDToken
	}
	return out
}

// callbackURI returns the absolute redirect_uri for a provider's callback.
func (p *Plugin) callbackURI(providerID string) string {
	cfg := p.auth.Config()
	base := strings.TrimRight(cfg.BaseURL, "/")
	bp := strings.TrimRight(cfg.BasePath, "/")
	if bp == "" {
		bp = "/api/auth"
	}
	return base + bp + "/callback/" + providerID
}

func (p *Plugin) defaultErrorURL() string { return p.auth.Config().BaseURL }

// resolveCallback validates a caller-supplied callback URL. Empty → BaseURL;
// relative ("/...") → allowed (same origin); absolute → must be a trusted origin.
func (p *Plugin) resolveCallback(raw string) (string, error) {
	if raw == "" {
		return p.auth.Config().BaseURL, nil
	}
	if strings.HasPrefix(raw, "/") {
		return raw, nil
	}
	if !p.auth.IsTrustedOrigin(originOf(raw)) {
		return "", ErrUntrustedRedirect
	}
	return raw, nil
}

func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Scheme + "://" + u.Host
}

func (p *Plugin) redirect(w http.ResponseWriter, r *http.Request, target string) {
	if target == "" {
		target = p.auth.Config().BaseURL
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (p *Plugin) redirectError(w http.ResponseWriter, r *http.Request, target, code string) {
	u, err := url.Parse(target)
	if err != nil {
		http.Redirect(w, r, p.auth.Config().BaseURL, http.StatusFound)
		return
	}
	q := u.Query()
	q.Set("error", code)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func providerDisablesSignUp(pr Provider, requestSignUp bool) bool {
	sc, ok := pr.(SignUpControlProvider)
	if !ok {
		return false
	}
	if sc.DisableSignUp() {
		return true
	}
	return sc.DisableImplicitSignUp() && !requestSignUp
}

func errCode(err error) string {
	switch {
	case errors.Is(err, ErrAccountNotLinked):
		return "account_not_linked"
	case errors.Is(err, ErrSignUpDisabled):
		return "signup_disabled"
	case errors.Is(err, ErrAccountAlreadyLinked):
		return "account_already_linked"
	case errors.Is(err, ErrEmailMismatch):
		return "email_mismatch"
	case errors.Is(err, ErrEmailNotFound):
		return "email_not_found"
	default:
		return "oauth_error"
	}
}
