package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func postSignIn(t *testing.T, a http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in/social", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func TestSignInSocial_RedirectBranch(t *testing.T) {
	_, a := newTestPlugin(t, Options{Providers: map[string]Provider{"fake": &fakeProvider{id: "fake"}}})
	w := postSignIn(t, a.Handler(), `{"provider":"fake","callbackURL":"http://localhost:3000"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}
	var body struct {
		Redirect bool   `json:"redirect"`
		URL      string `json:"url"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Redirect || body.URL == "" {
		t.Errorf("unexpected body: %+v", body)
	}
	if !hasCookie(w, "goten_oauth_state") {
		t.Errorf("state cookie not set")
	}
}

func TestSignInSocial_ProviderNotFound(t *testing.T) {
	_, a := newTestPlugin(t, Options{Providers: map[string]Provider{"fake": &fakeProvider{id: "fake"}}})
	w := postSignIn(t, a.Handler(), `{"provider":"nope","callbackURL":"http://localhost:3000"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

func TestSignInSocial_UntrustedCallback(t *testing.T) {
	_, a := newTestPlugin(t, Options{Providers: map[string]Provider{"fake": &fakeProvider{id: "fake"}}})
	w := postSignIn(t, a.Handler(), `{"provider":"fake","callbackURL":"http://evil.example"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("code = %d, want 403", w.Code)
	}
}

func TestCallback_FullFlow(t *testing.T) {
	provider := &fakeProvider{
		id:     "fake",
		tokens: &Tokens{AccessToken: "AT", IDToken: "IDT"},
		info:   &UserInfo{ID: "g9", Email: "flow@example.com", EmailVerified: true, Name: "Flow"},
	}
	_, a := newTestPlugin(t, Options{Providers: map[string]Provider{"fake": provider}})

	// 1. Begin sign-in to obtain the state token + signed state cookie.
	signIn := postSignIn(t, a.Handler(), `{"provider":"fake","callbackURL":"http://localhost:3000"}`)
	var sb struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(signIn.Body).Decode(&sb); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(sb.URL)
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("no state in auth url")
	}

	// 2. Hit the callback with the code + state, carrying the state cookie.
	cr := httptest.NewRequest(http.MethodGet, "/api/auth/callback/fake?code=abc&state="+state, nil)
	for _, c := range signIn.Result().Cookies() {
		cr.AddCookie(c)
	}
	cw := httptest.NewRecorder()
	a.Handler().ServeHTTP(cw, cr)

	if cw.Code != http.StatusFound {
		t.Fatalf("callback code = %d, body = %s", cw.Code, cw.Body.String())
	}
	if loc := cw.Header().Get("Location"); loc != "http://localhost:3000" {
		t.Errorf("Location = %q, want callbackURL", loc)
	}
	if !hasCookie(cw, "goten_session") {
		t.Errorf("session cookie not set")
	}
	if user, _ := a.InternalAdapter().FindUserByEmail(context.Background(), "flow@example.com"); user == nil {
		t.Errorf("user not created")
	}
}

func TestCallback_InvalidState(t *testing.T) {
	_, a := newTestPlugin(t, Options{Providers: map[string]Provider{"fake": &fakeProvider{id: "fake"}}})
	cr := httptest.NewRequest(http.MethodGet, "/api/auth/callback/fake?code=abc&state=bogus", nil)
	cw := httptest.NewRecorder()
	a.Handler().ServeHTTP(cw, cr)

	if cw.Code != http.StatusFound {
		t.Fatalf("code = %d, want 302", cw.Code)
	}
	loc := cw.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_state") {
		t.Errorf("Location = %q, want error=invalid_state", loc)
	}
}

func hasCookie(w *httptest.ResponseRecorder, name string) bool {
	for _, c := range w.Result().Cookies() {
		if c.Name == name && c.Value != "" {
			return true
		}
	}
	return false
}
