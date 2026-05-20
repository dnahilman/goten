package usernameplugin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
	usernameplugin "github.com/dnahilman/goten/plugins/username"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthWithUsername(t *testing.T) *goten.Auth {
	t.Helper()
	p := usernameplugin.New(usernameplugin.Options{})
	a, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
		Plugins: []goten.Plugin{p},
	})
	require.NoError(t, err)
	return a
}

func postJSON(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestSignUpUsername_Success(t *testing.T) {
	a := newAuthWithUsername(t)
	w := postJSON(t, a.Handler(), "/api/auth/sign-up/username",
		`{"username":"alice","password":"secret123","name":"Alice"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body["user"])
	assert.NotNil(t, body["session"])

	// Cookie must be set
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "goten_session" {
			assert.True(t, strings.HasPrefix(c.Value, "g10_"))
			found = true
		}
	}
	assert.True(t, found, "goten_session cookie should be set")
}

func TestSignUpUsername_DuplicateUsername(t *testing.T) {
	a := newAuthWithUsername(t)
	body := `{"username":"bob","password":"secret123"}`
	postJSON(t, a.Handler(), "/api/auth/sign-up/username", body)
	w := postJSON(t, a.Handler(), "/api/auth/sign-up/username", body)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "USERNAME_EXISTS", resp["code"])
}

func TestSignUpUsername_InvalidUsername(t *testing.T) {
	a := newAuthWithUsername(t)

	cases := []struct {
		username string
		desc     string
	}{
		{"ab", "too short"},
		{"a@b", "invalid char"},
		{strings.Repeat("a", 33), "too long"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			w := postJSON(t, a.Handler(), "/api/auth/sign-up/username",
				`{"username":"`+tc.username+`","password":"secret123"}`)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			var resp map[string]string
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
			assert.Equal(t, "INVALID_USERNAME", resp["code"])
		})
	}
}

func TestSignUpUsername_InvalidPassword(t *testing.T) {
	a := newAuthWithUsername(t)
	w := postJSON(t, a.Handler(), "/api/auth/sign-up/username",
		`{"username":"carol","password":"short"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "INVALID_PASSWORD", resp["code"])
}

func TestSignInUsername_Success(t *testing.T) {
	a := newAuthWithUsername(t)
	postJSON(t, a.Handler(), "/api/auth/sign-up/username",
		`{"username":"dave","password":"secret123"}`)

	w := postJSON(t, a.Handler(), "/api/auth/sign-in/username",
		`{"username":"dave","password":"secret123"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotNil(t, body["user"])
	assert.NotNil(t, body["session"])
}

func TestSignInUsername_WrongPassword(t *testing.T) {
	a := newAuthWithUsername(t)
	postJSON(t, a.Handler(), "/api/auth/sign-up/username",
		`{"username":"eve","password":"secret123"}`)

	w := postJSON(t, a.Handler(), "/api/auth/sign-in/username",
		`{"username":"eve","password":"wrongpassword"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "INVALID_CREDENTIALS", resp["code"])
}

func TestSignInUsername_UnknownUser(t *testing.T) {
	a := newAuthWithUsername(t)
	w := postJSON(t, a.Handler(), "/api/auth/sign-in/username",
		`{"username":"nobody","password":"secret123"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Same code as wrong password — anti-enumeration
	assert.Equal(t, "INVALID_CREDENTIALS", resp["code"])
}
