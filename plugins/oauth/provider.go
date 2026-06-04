package oauth

// Provider is the contract every social provider implements. It mirrors
// better-auth's OAuthProvider object: a required core plus optional capabilities.
type Provider interface {
	// ID is the provider identifier (e.g. "google"); matches the registration key.
	ID() string
	// CreateAuthorizationURL builds the provider's authorization URL.
	CreateAuthorizationURL(p AuthURLParams) (string, error)
	// ValidateAuthorizationCode exchanges an authorization code for tokens.
	ValidateAuthorizationCode(p CodeExchangeParams) (*Tokens, error)
	// GetUserInfo fetches the normalized user profile from the tokens.
	GetUserInfo(t *Tokens) (*UserInfo, error)
}

// TokenRefresherProvider is optional. If implemented, it is used to refresh an
// access token; otherwise refresh is unavailable for that provider.
type TokenRefresherProvider interface {
	RefreshAccessToken(refreshToken string) (*Tokens, error)
}

// IDTokenVerifierProvider is optional. If implemented, the provider supports the
// "sign in with id_token" branch of /sign-in/social (native/one-tap), verifying
// the token signature (e.g. via the provider's JWKS).
type IDTokenVerifierProvider interface {
	VerifyIDToken(token, nonce string) (bool, error)
}

// SignUpControlProvider is optional. It lets a provider disable implicit sign-up
// (requiring requestSignUp) or sign-up entirely.
type SignUpControlProvider interface {
	DisableImplicitSignUp() bool
	DisableSignUp() bool
}
