package goten_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAuthForMiddleware builds an Auth with TrustedOrigins set (strict CSRF mode).
func newAuthForMiddleware(t *testing.T) *goten.Auth {
	t.Helper()
	a, err := goten.New(goten.Config{
		BaseURL:        "http://example.com",
		Secret:         "test-secret-key-that-is-32-bytes!",
		Adapter:        testutil.NewMockAdapter(),
		TrustedOrigins: []string{"http://example.com"},
	})
	require.NoError(t, err)
	return a
}

// newAuthPermissive builds an Auth without TrustedOrigins (permissive CSRF mode).
func newAuthPermissive(t *testing.T) *goten.Auth {
	t.Helper()
	a, err := goten.New(goten.Config{
		BaseURL: "http://example.com",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
	})
	require.NoError(t, err)
	return a
}

func TestRequireAuth_NoToken(t *testing.T) {
	a := newAuthForMiddleware(t)
	protected := a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_ValidSession(t *testing.T) {
	a := newAuthForMiddleware(t)
	// Sign up with matching Origin to satisfy strict CSRF mode.
	signUpReq := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"mw@example.com","password":"secret123","name":"MW"}`))
	signUpReq.Header.Set("Content-Type", "application/json")
	signUpReq.Header.Set("Origin", "http://example.com")
	signUpW := httptest.NewRecorder()
	a.Handler().ServeHTTP(signUpW, signUpReq)

	var token string
	for _, c := range signUpW.Result().Cookies() {
		if c.Name == "goten_session" {
			token = c.Value
		}
	}
	require.NotEmpty(t, token)

	var gotUser bool
	protected := a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := goten.UserFromContext(r.Context())
		gotUser = ok
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: token})
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, gotUser, "user should be in context")
}

// TestOriginCheck_StrictMode — when TrustedOrigins is set, POST without Origin → 403.
func TestOriginCheck_StrictMode_NoOrigin(t *testing.T) {
	a := newAuthForMiddleware(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"x@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	a.Handler().ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestOriginCheck_PermissiveMode — when TrustedOrigins is empty, POST without Origin → allowed.
func TestOriginCheck_PermissiveMode_NoOrigin(t *testing.T) {
	a := newAuthPermissive(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"x@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	a.Handler().ServeHTTP(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestOriginCheck_PostWithUntrustedOrigin(t *testing.T) {
	a := newAuthForMiddleware(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"x@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://evil.com")
	a.Handler().ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOriginCheck_PostWithTrustedOrigin(t *testing.T) {
	a := newAuthForMiddleware(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"z@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://example.com")
	a.Handler().ServeHTTP(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestOriginCheck_PostWithBearer(t *testing.T) {
	a := newAuthForMiddleware(t)
	// Bearer bypasses origin check entirely.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"y@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://evil.com")
	r.Header.Set("Authorization", "Bearer g10_sometoken")
	a.Handler().ServeHTTP(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestOriginCheck_RefererFallback(t *testing.T) {
	a := newAuthForMiddleware(t)
	// Referer header used when Origin is missing.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-up/email",
		strings.NewReader(`{"email":"ref@example.com","password":"secret123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Referer", "http://example.com/page")
	a.Handler().ServeHTTP(w, r)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}
