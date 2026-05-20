package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dnahilman/goten/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCookie(t *testing.T) {
	w := httptest.NewRecorder()
	cfg := session.DefaultCookieConfig()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	session.SetCookie(w, cfg, "g10_testtoken", expiresAt)

	resp := w.Result()
	cookies := resp.Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, "goten_session", c.Name)
	assert.Equal(t, "g10_testtoken", c.Value)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, "/", c.Path)
}

func TestSetCookie_CustomName(t *testing.T) {
	w := httptest.NewRecorder()
	cfg := session.CookieConfig{
		Name:     "my_auth",
		Path:     "/api",
		HTTPOnly: true,
		Secure:   true,
	}
	session.SetCookie(w, cfg, "g10_abc", time.Now().Add(time.Hour))

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "my_auth", cookies[0].Name)
	assert.Equal(t, "/api", cookies[0].Path)
	assert.True(t, cookies[0].Secure)
}

func TestClearCookie(t *testing.T) {
	w := httptest.NewRecorder()
	cfg := session.DefaultCookieConfig()

	session.ClearCookie(w, cfg)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, "goten_session", c.Name)
	assert.Equal(t, "", c.Value)
	assert.Equal(t, -1, c.MaxAge)
}

func TestGetSessionToken_FromCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: "g10_cookietoken"})

	tok := session.GetSessionToken(r, "")
	assert.Equal(t, "g10_cookietoken", tok)
}

func TestGetSessionToken_FromBearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer g10_bearertoken")

	tok := session.GetSessionToken(r, "")
	assert.Equal(t, "g10_bearertoken", tok)
}

func TestGetSessionToken_CookiePriorityOverBearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "goten_session", Value: "g10_cookiefirst"})
	r.Header.Set("Authorization", "Bearer g10_bearersecond")

	tok := session.GetSessionToken(r, "")
	assert.Equal(t, "g10_cookiefirst", tok, "cookie should take priority over Bearer")
}

func TestGetSessionToken_NonePresent(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	tok := session.GetSessionToken(r, "")
	assert.Equal(t, "", tok)
}

func TestGetSessionToken_CustomCookieName(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "my_session", Value: "g10_custom"})

	tok := session.GetSessionToken(r, "my_session")
	assert.Equal(t, "g10_custom", tok)
}
