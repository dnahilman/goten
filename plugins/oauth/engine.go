package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// httpClient is the shared client for token-endpoint and JWKS calls.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// AuthURLInput configures CreateAuthorizationURL. Providers fill this in and call
// the engine, mirroring better-auth's createAuthorizationURL helper.
type AuthURLInput struct {
	AuthorizationEndpoint string
	ClientID              string
	State                 string
	CodeVerifier          string // when set → PKCE S256 (code_challenge added)
	Scopes                []string
	RedirectURI           string
	ResponseType          string            // default "code"
	ScopeJoiner           string            // default " "
	ExtraParams           map[string]string // access_type, prompt, display, include_granted_scopes, ...
}

// CreateAuthorizationURL builds an OAuth2 authorization URL by hand (no oauth2 lib).
func CreateAuthorizationURL(in AuthURLInput) (string, error) {
	u, err := url.Parse(in.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("oauth: bad authorization endpoint: %w", err)
	}
	q := u.Query()
	respType := in.ResponseType
	if respType == "" {
		respType = "code"
	}
	q.Set("response_type", respType)
	q.Set("client_id", in.ClientID)
	q.Set("state", in.State)
	q.Set("redirect_uri", in.RedirectURI)
	if len(in.Scopes) > 0 {
		joiner := in.ScopeJoiner
		if joiner == "" {
			joiner = " "
		}
		q.Set("scope", strings.Join(in.Scopes, joiner))
	}
	if in.CodeVerifier != "" {
		q.Set("code_challenge_method", "S256")
		q.Set("code_challenge", CodeChallengeS256(in.CodeVerifier))
	}
	for k, v := range in.ExtraParams {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// CodeExchangeInput configures ValidateAuthorizationCode.
type CodeExchangeInput struct {
	TokenEndpoint  string
	ClientID       string
	ClientSecret   string
	Code           string
	CodeVerifier   string
	RedirectURI    string
	Authentication string // "post" (default) or "basic"
	ExtraParams    map[string]string
}

// ValidateAuthorizationCode exchanges an authorization code for tokens via a
// manual application/x-www-form-urlencoded POST (mirror of better-auth).
func ValidateAuthorizationCode(ctx context.Context, in CodeExchangeInput) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", in.Code)
	if in.CodeVerifier != "" {
		form.Set("code_verifier", in.CodeVerifier)
	}
	form.Set("redirect_uri", in.RedirectURI)
	return tokenRequest(ctx, in.TokenEndpoint, form, in.ClientID, in.ClientSecret, in.Authentication, in.ExtraParams)
}

// RefreshInput configures RefreshAccessToken.
type RefreshInput struct {
	TokenEndpoint  string
	ClientID       string
	ClientSecret   string
	RefreshToken   string
	Authentication string
	ExtraParams    map[string]string
}

// RefreshAccessToken exchanges a refresh token for new tokens (manual POST).
func RefreshAccessToken(ctx context.Context, in RefreshInput) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", in.RefreshToken)
	return tokenRequest(ctx, in.TokenEndpoint, form, in.ClientID, in.ClientSecret, in.Authentication, in.ExtraParams)
}

func tokenRequest(ctx context.Context, endpoint string, form url.Values, clientID, clientSecret, auth string, extra map[string]string) (*Tokens, error) {
	if auth == "basic" {
		// Credentials go in the Authorization header.
	} else {
		form.Set("client_id", clientID)
		if clientSecret != "" {
			form.Set("client_secret", clientSecret)
		}
	}
	for k, v := range extra {
		if form.Get(k) == "" {
			form.Set(k, v)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if auth == "basic" {
		creds := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
		req.Header.Set("Authorization", "Basic "+creds)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("oauth: decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth: token endpoint returned %d: %v", resp.StatusCode, raw["error"])
	}
	return tokensFromResponse(raw), nil
}

// tokensFromResponse maps a raw token JSON response to Tokens (mirror getOAuth2Tokens).
func tokensFromResponse(data map[string]any) *Tokens {
	t := &Tokens{Raw: data}
	t.TokenType, _ = data["token_type"].(string)
	t.AccessToken, _ = data["access_token"].(string)
	t.RefreshToken, _ = data["refresh_token"].(string)
	t.IDToken, _ = data["id_token"].(string)
	if s, ok := data["scope"].(string); ok && s != "" {
		t.Scopes = strings.Fields(s)
	}
	if exp, ok := asSeconds(data["expires_in"]); ok {
		at := time.Now().UTC().Add(time.Duration(exp) * time.Second)
		t.AccessTokenExpiresAt = &at
	}
	if exp, ok := asSeconds(data["refresh_token_expires_in"]); ok {
		rt := time.Now().UTC().Add(time.Duration(exp) * time.Second)
		t.RefreshTokenExpiresAt = &rt
	}
	return t
}

func asSeconds(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}

// --- PKCE & random ---

// GenerateCodeVerifier returns a high-entropy PKCE code_verifier (base64url, no padding).
func GenerateCodeVerifier() string { return randomBase64URL(64) }

// GenerateState returns a random opaque state/nonce string.
func generateStateToken() string { return randomBase64URL(32) }

// CodeChallengeS256 returns base64url(SHA-256(verifier)) with no padding.
func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomBase64URL(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// --- id_token ---

// DecodeIDTokenClaims decodes a JWT payload WITHOUT verifying its signature.
// Safe only for tokens received directly from a provider's token endpoint over
// TLS (back-channel). For client-supplied id_tokens use a verifying path (JWKS).
func DecodeIDTokenClaims(idToken string) (map[string]any, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrIDTokenInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrIDTokenInvalid
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrIDTokenInvalid
	}
	return claims, nil
}
