package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCreateAuthorizationURL(t *testing.T) {
	raw, err := CreateAuthorizationURL(AuthURLInput{
		AuthorizationEndpoint: "https://idp.example/auth",
		ClientID:              "client-123",
		State:                 "state-abc",
		CodeVerifier:          "verifier-xyz",
		Scopes:                []string{"openid", "email"},
		RedirectURI:           "http://localhost:8080/api/auth/callback/google",
		ExtraParams:           map[string]string{"access_type": "offline"},
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationURL: %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "client-123",
		"state":                 "state-abc",
		"scope":                 "openid email",
		"redirect_uri":          "http://localhost:8080/api/auth/callback/google",
		"code_challenge_method": "S256",
		"access_type":           "offline",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("query %q = %q, want %q", k, got, want)
		}
	}
	if q.Get("code_challenge") != CodeChallengeS256("verifier-xyz") {
		t.Errorf("code_challenge mismatch")
	}
}

func TestValidateAuthorizationCode_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if form.Get("grant_type") != "authorization_code" || form.Get("code") != "the-code" {
			t.Errorf("unexpected form: %s", body)
		}
		if form.Get("client_id") != "cid" || form.Get("client_secret") != "secret" {
			t.Errorf("client creds missing from body (post auth): %s", body)
		}
		if form.Get("code_verifier") != "ver" {
			t.Errorf("code_verifier missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"AT","refresh_token":"RT","expires_in":3600,"scope":"openid email","token_type":"Bearer","id_token":"IDT"}`)
	}))
	defer srv.Close()

	tok, err := ValidateAuthorizationCode(context.Background(), CodeExchangeInput{
		TokenEndpoint:  srv.URL,
		ClientID:       "cid",
		ClientSecret:   "secret",
		Code:           "the-code",
		CodeVerifier:   "ver",
		RedirectURI:    "http://localhost/cb",
		Authentication: "post",
	})
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" || tok.IDToken != "IDT" {
		t.Errorf("tokens = %+v", tok)
	}
	if len(tok.Scopes) != 2 || tok.Scopes[0] != "openid" {
		t.Errorf("scopes = %v", tok.Scopes)
	}
	if tok.AccessTokenExpiresAt == nil {
		t.Errorf("expected access token expiry")
	}
}

func TestValidateAuthorizationCode_Basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "cid" || pass != "secret" {
			t.Errorf("expected basic auth, got ok=%v user=%q", ok, user)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"AT","token_type":"Bearer"}`)
	}))
	defer srv.Close()

	_, err := ValidateAuthorizationCode(context.Background(), CodeExchangeInput{
		TokenEndpoint:  srv.URL,
		ClientID:       "cid",
		ClientSecret:   "secret",
		Code:           "c",
		RedirectURI:    "http://localhost/cb",
		Authentication: "basic",
	})
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
}

func TestValidateAuthorizationCode_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
	}))
	defer srv.Close()

	_, err := ValidateAuthorizationCode(context.Background(), CodeExchangeInput{
		TokenEndpoint: srv.URL,
		ClientID:      "cid",
		Code:          "c",
		RedirectURI:   "http://localhost/cb",
	})
	if err == nil {
		t.Fatalf("expected error on non-200 token response")
	}
}

func TestDecodeIDTokenClaims(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"123","email":"a@b.com","email_verified":true}`))
	token := "header." + payload + ".sig"
	claims, err := DecodeIDTokenClaims(token)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if claims["sub"] != "123" || claims["email"] != "a@b.com" || claims["email_verified"] != true {
		t.Errorf("claims = %v", claims)
	}
}

func TestDecodeIDTokenClaims_Malformed(t *testing.T) {
	if _, err := DecodeIDTokenClaims("not-a-jwt"); err == nil {
		t.Errorf("expected error for malformed token")
	}
}

func TestPKCE(t *testing.T) {
	v := GenerateCodeVerifier()
	if len(v) < 43 {
		t.Errorf("verifier too short: %d", len(v))
	}
	if strings.ContainsAny(v, "+/=") {
		t.Errorf("verifier not base64url: %q", v)
	}
	if GenerateCodeVerifier() == v {
		t.Errorf("verifiers should be random")
	}
}

var _ = json.Marshal
