package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuth(t *testing.T) *goten.Auth {
	t.Helper()
	a, err := goten.New(goten.Config{
		BaseURL: "http://localhost:8080",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
	})
	require.NoError(t, err)
	return a
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestSignUp_Success(t *testing.T) {
	a := newAuth(t)
	w := post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"alice@example.com","password":"secret123","name":"Alice"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body["user"])
	assert.NotNil(t, body["session"])

	// Cookie must be set
	resp := w.Result()
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "goten_session" {
			assert.True(t, strings.HasPrefix(c.Value, "g10_"))
			found = true
		}
	}
	assert.True(t, found, "goten_session cookie should be set")
}

func TestSignUp_DuplicateEmail(t *testing.T) {
	a := newAuth(t)
	body := `{"email":"bob@example.com","password":"secret123","name":"Bob"}`
	post(t, a.Handler(), "/api/auth/sign-up/email", body) // first
	w := post(t, a.Handler(), "/api/auth/sign-up/email", body) // duplicate

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "EMAIL_EXISTS", resp["code"])
}

func TestSignUp_InvalidEmail(t *testing.T) {
	a := newAuth(t)
	w := post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"notanemail","password":"secret123"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "INVALID_EMAIL", resp["code"])
}

func TestSignUp_PasswordTooShort(t *testing.T) {
	a := newAuth(t)
	w := post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"carol@example.com","password":"short"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "PASSWORD_TOO_SHORT", resp["code"])
}

func TestSignIn_Success(t *testing.T) {
	a := newAuth(t)
	// Sign up first
	post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"dave@example.com","password":"secret123","name":"Dave"}`)

	w := post(t, a.Handler(), "/api/auth/sign-in/email",
		`{"email":"dave@example.com","password":"secret123"}`)
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body["user"])
	assert.NotNil(t, body["session"])
}

func TestSignIn_WrongPassword(t *testing.T) {
	a := newAuth(t)
	post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"eve@example.com","password":"secret123","name":"Eve"}`)

	w := post(t, a.Handler(), "/api/auth/sign-in/email",
		`{"email":"eve@example.com","password":"wrongpassword"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "INVALID_CREDENTIALS", resp["code"])
}

func TestSignIn_UnknownEmail(t *testing.T) {
	a := newAuth(t)
	w := post(t, a.Handler(), "/api/auth/sign-in/email",
		`{"email":"nobody@example.com","password":"secret123"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Same error code as wrong password — anti-enumeration
	assert.Equal(t, "INVALID_CREDENTIALS", resp["code"])
}

func TestSignOut(t *testing.T) {
	a := newAuth(t)
	// Sign up to get a session cookie
	signUpResp := post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"frank@example.com","password":"secret123","name":"Frank"}`)
	var token string
	for _, c := range signUpResp.Result().Cookies() {
		if c.Name == "goten_session" {
			token = c.Value
		}
	}
	require.NotEmpty(t, token)

	// Sign out
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-out", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: token})
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Cookie should be cleared (MaxAge=-1)
	for _, c := range w.Result().Cookies() {
		if c.Name == "goten_session" {
			assert.Equal(t, -1, c.MaxAge, "cookie should be cleared")
		}
	}
}
