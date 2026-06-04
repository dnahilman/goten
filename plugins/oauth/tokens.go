package oauth

import (
	"context"
	"strings"
	"time"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/crypto"
)

// accountTokenFields builds the account columns for the provider tokens,
// encrypting them when EncryptOAuthTokens is enabled.
func (p *Plugin) accountTokenFields(t *Tokens) map[string]any {
	fields := map[string]any{
		"access_token":  p.maybeEncrypt(t.AccessToken),
		"refresh_token": p.maybeEncrypt(t.RefreshToken),
		"id_token":      p.maybeEncrypt(t.IDToken),
		"scope":         strings.Join(t.Scopes, ","),
	}
	if t.AccessTokenExpiresAt != nil {
		fields["access_token_expires_at"] = *t.AccessTokenExpiresAt
	}
	if t.RefreshTokenExpiresAt != nil {
		fields["refresh_token_expires_at"] = *t.RefreshTokenExpiresAt
	}
	return fields
}

// maybeEncrypt encrypts v when token encryption is enabled. The key derived from
// Config.Secret is always valid, so encryption does not fail in practice; on the
// off chance it does, the plaintext is stored rather than aborting sign-in.
func (p *Plugin) maybeEncrypt(v string) string {
	if v == "" || !p.opts.EncryptOAuthTokens {
		return v
	}
	enc, err := crypto.Encrypt(v, crypto.DeriveKey(p.secret()))
	if err != nil {
		return v
	}
	return enc
}

// maybeDecrypt reverses maybeEncrypt. If decryption fails (e.g. a value stored
// before encryption was enabled), the original value is returned.
func (p *Plugin) maybeDecrypt(v string) string {
	if v == "" || !p.opts.EncryptOAuthTokens {
		return v
	}
	dec, err := crypto.Decrypt(v, crypto.DeriveKey(p.secret()))
	if err != nil {
		return v
	}
	return dec
}

// findUserAccount returns the raw accounts row for (userID, providerID), or nil.
func (p *Plugin) findUserAccount(ctx context.Context, userID, providerID string) (map[string]any, error) {
	return p.auth.Adapter().FindOne(ctx, "accounts", goten.Query{
		Where: []goten.Where{
			goten.EQ("user_id", userID),
			goten.EQ("provider_id", providerID),
		},
	})
}

// storedTokens decodes (and decrypts) the provider tokens from an accounts row.
func (p *Plugin) storedTokens(rec map[string]any) *Tokens {
	t := &Tokens{}
	if v, ok := rec["access_token"].(string); ok {
		t.AccessToken = p.maybeDecrypt(v)
	}
	if v, ok := rec["refresh_token"].(string); ok {
		t.RefreshToken = p.maybeDecrypt(v)
	}
	if v, ok := rec["id_token"].(string); ok {
		t.IDToken = p.maybeDecrypt(v)
	}
	if v, ok := rec["scope"].(string); ok && v != "" {
		t.Scopes = strings.Split(v, ",")
	}
	if v, ok := rec["access_token_expires_at"].(time.Time); ok {
		t.AccessTokenExpiresAt = &v
	}
	if v, ok := rec["refresh_token_expires_at"].(time.Time); ok {
		t.RefreshTokenExpiresAt = &v
	}
	return t
}

// getAccessToken returns a valid access token for the user's provider account,
// refreshing it first when expired and a refresh token is available.
func (p *Plugin) getAccessToken(ctx context.Context, provider Provider, userID string) (*Tokens, error) {
	rec, err := p.findUserAccount(ctx, userID, provider.ID())
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrAccountNotFound
	}
	tokens := p.storedTokens(rec)
	if tokens.AccessTokenExpiresAt != nil && time.Now().UTC().After(*tokens.AccessTokenExpiresAt) && tokens.RefreshToken != "" {
		if refreshed, err := p.refreshAccountToken(ctx, provider, rec, tokens); err == nil {
			return refreshed, nil
		}
	}
	return tokens, nil
}

// refreshAccessToken refreshes and persists the user's provider tokens.
func (p *Plugin) refreshAccessToken(ctx context.Context, provider Provider, userID string) (*Tokens, error) {
	rec, err := p.findUserAccount(ctx, userID, provider.ID())
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrAccountNotFound
	}
	return p.refreshAccountToken(ctx, provider, rec, p.storedTokens(rec))
}

func (p *Plugin) refreshAccountToken(ctx context.Context, provider Provider, rec map[string]any, current *Tokens) (*Tokens, error) {
	refresher, ok := provider.(TokenRefresherProvider)
	if !ok {
		return nil, ErrRefreshNotSupported
	}
	if current.RefreshToken == "" {
		return nil, ErrNoRefreshToken
	}
	fresh, err := refresher.RefreshAccessToken(current.RefreshToken)
	if err != nil {
		return nil, err
	}
	// Providers may omit a new refresh token; keep the existing one.
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = current.RefreshToken
	}
	fields := p.accountTokenFields(fresh)
	fields["updated_at"] = time.Now().UTC()
	id, _ := rec["id"].(string)
	if _, err := p.auth.Adapter().Update(ctx, "accounts", goten.Query{Where: []goten.Where{goten.EQ("id", id)}}, fields); err != nil {
		return nil, err
	}
	return fresh, nil
}
