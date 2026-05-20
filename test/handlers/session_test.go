package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signUpAndGetToken(t *testing.T) (http.Handler, string) {
	t.Helper()
	a := newAuth(t)
	w := post(t, a.Handler(), "/api/auth/sign-up/email",
		`{"email":"sess@example.com","password":"secret123","name":"Sess"}`)
	var token string
	for _, c := range w.Result().Cookies() {
		if c.Name == "goten_session" {
			token = c.Value
		}
	}
	require.NotEmpty(t, token)
	return a.Handler(), token
}

func TestGetSession_NoToken(t *testing.T) {
	a := newAuth(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/get-session", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetSession_ValidCookie(t *testing.T) {
	h, token := signUpAndGetToken(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/get-session", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body["user"])
	assert.NotNil(t, body["session"])
}

func TestGetSession_ValidBearer(t *testing.T) {
	h, token := signUpAndGetToken(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/get-session", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListSessions_NoAuth(t *testing.T) {
	a := newAuth(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/list-sessions", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListSessions_Authenticated(t *testing.T) {
	h, token := signUpAndGetToken(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/list-sessions", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	sessions, ok := body["sessions"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(sessions), 1)
}
