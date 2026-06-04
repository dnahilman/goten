// Package oauth provides social/OAuth2 sign-in for goten, modeled after
// better-auth's social-provider design: a small Provider contract plus a manual
// OAuth2 engine (no external OAuth library). Providers (e.g. Google) are
// registered via Options.Providers, keyed by provider id.
package oauth

import "time"

// Tokens holds an OAuth2 token-exchange result (mirror of better-auth OAuth2Tokens).
type Tokens struct {
	TokenType             string
	AccessToken           string
	RefreshToken          string
	IDToken               string
	AccessTokenExpiresAt  *time.Time
	RefreshTokenExpiresAt *time.Time
	Scopes                []string
	// Raw preserves the provider's raw token response for provider-specific fields.
	Raw map[string]any
}

// UserInfo is the normalized profile returned by a provider (mirror of OAuth2UserInfo).
type UserInfo struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	Image         string
}

// AuthURLParams are the inputs the engine passes to Provider.CreateAuthorizationURL.
type AuthURLParams struct {
	State        string
	CodeVerifier string
	Scopes       []string
	RedirectURI  string
	LoginHint    string
	Display      string
}

// CodeExchangeParams are the inputs the engine passes to Provider.ValidateAuthorizationCode.
type CodeExchangeParams struct {
	Code         string
	CodeVerifier string
	RedirectURI  string
}
