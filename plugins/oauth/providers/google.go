// Package providers contains built-in OAuth providers for the goten oauth plugin.
package providers

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/dnahilman/goten/plugins/oauth"
)

const (
	googleAuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	googleJWKSURL       = "https://www.googleapis.com/oauth2/v3/certs"
)

// GoogleOptions configures the Google provider. ClientID/ClientSecret default to
// the GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET environment variables.
type GoogleOptions struct {
	ClientID     string
	ClientSecret string
	// ClientIDs lists additional client IDs accepted as id_token audiences
	// (cross-platform apps). The primary ClientID is always accepted.
	ClientIDs []string
	// RedirectURI overrides the auto-derived callback URL.
	RedirectURI string
	// Scopes overrides the default ["openid", "email", "profile"].
	Scopes []string
	// AccessType "offline" requests a refresh token (usually with Prompt "consent").
	AccessType string
	// Prompt e.g. "consent", "select_account".
	Prompt string
	// Hd restricts sign-in to a Google Workspace hosted domain.
	Hd string
}

type googleProvider struct {
	opts GoogleOptions
}

// Google returns a Google OAuth provider implementing oauth.Provider.
func Google(opts GoogleOptions) oauth.Provider {
	if opts.ClientID == "" {
		opts.ClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if opts.ClientSecret == "" {
		opts.ClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	if len(opts.Scopes) == 0 {
		opts.Scopes = []string{"openid", "email", "profile"}
	}
	return &googleProvider{opts: opts}
}

func (g *googleProvider) ID() string { return "google" }

func (g *googleProvider) CreateAuthorizationURL(p oauth.AuthURLParams) (string, error) {
	extra := map[string]string{"include_granted_scopes": "true"}
	if g.opts.AccessType != "" {
		extra["access_type"] = g.opts.AccessType
	}
	if g.opts.Prompt != "" {
		extra["prompt"] = g.opts.Prompt
	}
	if g.opts.Hd != "" {
		extra["hd"] = g.opts.Hd
	}
	if p.LoginHint != "" {
		extra["login_hint"] = p.LoginHint
	}
	if p.Display != "" {
		extra["display"] = p.Display
	}
	scopes := p.Scopes
	if len(scopes) == 0 {
		scopes = g.opts.Scopes
	}
	return oauth.CreateAuthorizationURL(oauth.AuthURLInput{
		AuthorizationEndpoint: googleAuthEndpoint,
		ClientID:              g.opts.ClientID,
		State:                 p.State,
		CodeVerifier:          p.CodeVerifier,
		Scopes:                scopes,
		RedirectURI:           g.redirectURI(p.RedirectURI),
		ExtraParams:           extra,
	})
}

func (g *googleProvider) ValidateAuthorizationCode(p oauth.CodeExchangeParams) (*oauth.Tokens, error) {
	return oauth.ValidateAuthorizationCode(context.Background(), oauth.CodeExchangeInput{
		TokenEndpoint:  googleTokenEndpoint,
		ClientID:       g.opts.ClientID,
		ClientSecret:   g.opts.ClientSecret,
		Code:           p.Code,
		CodeVerifier:   p.CodeVerifier,
		RedirectURI:    g.redirectURI(p.RedirectURI),
		Authentication: "post",
	})
}

func (g *googleProvider) GetUserInfo(t *oauth.Tokens) (*oauth.UserInfo, error) {
	if t.IDToken == "" {
		return nil, errors.New("google: id_token missing; include the openid scope")
	}
	claims, err := oauth.DecodeIDTokenClaims(t.IDToken)
	if err != nil {
		return nil, err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("google: id_token missing sub claim")
	}
	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	return &oauth.UserInfo{
		ID:            sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Image:         picture,
	}, nil
}

// RefreshAccessToken implements oauth.TokenRefresherProvider.
func (g *googleProvider) RefreshAccessToken(refreshToken string) (*oauth.Tokens, error) {
	return oauth.RefreshAccessToken(context.Background(), oauth.RefreshInput{
		TokenEndpoint:  googleTokenEndpoint,
		ClientID:       g.opts.ClientID,
		ClientSecret:   g.opts.ClientSecret,
		RefreshToken:   refreshToken,
		Authentication: "post",
	})
}

// VerifyIDToken implements oauth.IDTokenVerifierProvider (native / one-tap).
func (g *googleProvider) VerifyIDToken(token, nonce string) (bool, error) {
	audiences := append([]string{g.opts.ClientID}, g.opts.ClientIDs...)
	if _, err := oauth.VerifyRS256JWT(context.Background(), token, googleJWKSURL, oauth.VerifyJWTOptions{
		Issuers:   []string{"https://accounts.google.com", "accounts.google.com"},
		Audiences: audiences,
		Nonce:     nonce,
		MaxAge:    time.Hour,
	}); err != nil {
		return false, nil
	}
	return true, nil
}

func (g *googleProvider) redirectURI(fallback string) string {
	if g.opts.RedirectURI != "" {
		return g.opts.RedirectURI
	}
	return fallback
}

// Compile-time interface checks.
var (
	_ oauth.Provider                = (*googleProvider)(nil)
	_ oauth.TokenRefresherProvider  = (*googleProvider)(nil)
	_ oauth.IDTokenVerifierProvider = (*googleProvider)(nil)
)
