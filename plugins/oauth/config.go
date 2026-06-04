package oauth

import "time"

// Options configures the OAuth plugin.
type Options struct {
	// Providers maps a provider id (e.g. "google") to its implementation,
	// mirroring better-auth's socialProviders map.
	Providers map[string]Provider

	// EncryptOAuthTokens encrypts stored provider tokens at rest (AES-256-GCM,
	// keyed from Config.Secret). Default false — matching better-auth.
	EncryptOAuthTokens bool

	// TrustedProviders lists provider ids whose own email-verified claim is
	// trusted for auto-linking even when the local account is unverified.
	TrustedProviders []string

	// AccountLinking controls how OAuth accounts link to existing users.
	AccountLinking AccountLinkingOptions

	// CookieName overrides the OAuth state cookie name (default "goten_oauth_state").
	CookieName string

	// StateTTL overrides the OAuth state lifetime (default 10 minutes).
	StateTTL time.Duration
}

// AccountLinkingOptions mirrors better-auth's account.accountLinking config.
type AccountLinkingOptions struct {
	// Enabled toggles linking a provider account to an existing user found by
	// email. Default true. When false, sign-in for an existing email is rejected
	// unless the account is already linked.
	Enabled *bool

	// RequireLocalEmailVerified requires the EXISTING local user's email to be
	// verified before an unlinked provider account may auto-link to it. Default
	// true — guards against account takeover.
	RequireLocalEmailVerified *bool

	// AllowDifferentEmails permits the manual /link-social flow to link a
	// provider account whose email differs from the signed-in user's. Default false.
	AllowDifferentEmails bool
}

func (o *Options) applyDefaults() {
	if o.CookieName == "" {
		o.CookieName = "goten_oauth_state"
	}
	if o.StateTTL == 0 {
		o.StateTTL = 10 * time.Minute
	}
	if o.AccountLinking.Enabled == nil {
		t := true
		o.AccountLinking.Enabled = &t
	}
	if o.AccountLinking.RequireLocalEmailVerified == nil {
		t := true
		o.AccountLinking.RequireLocalEmailVerified = &t
	}
}

func (o *Options) linkingEnabled() bool { return o.AccountLinking.Enabled == nil || *o.AccountLinking.Enabled }

func (o *Options) requireLocalEmailVerified() bool {
	return o.AccountLinking.RequireLocalEmailVerified == nil || *o.AccountLinking.RequireLocalEmailVerified
}

func (o *Options) isTrustedProvider(id string) bool {
	for _, p := range o.TrustedProviders {
		if p == id {
			return true
		}
	}
	return false
}
