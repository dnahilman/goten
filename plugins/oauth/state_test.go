package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	p, _ := newTestPlugin(t, Options{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in/social", nil)

	state, verifier, err := p.generateState(w, r, stateData{CallbackURL: "http://localhost:3000/welcome"})
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if state == "" || verifier == "" {
		t.Fatal("empty state/verifier")
	}

	cr := httptest.NewRequest(http.MethodGet, "/api/auth/callback/fake?state="+state, nil)
	for _, c := range w.Result().Cookies() {
		cr.AddCookie(c)
	}
	sd, err := p.parseState(httptest.NewRecorder(), cr)
	if err != nil {
		t.Fatalf("parseState: %v", err)
	}
	if sd.CallbackURL != "http://localhost:3000/welcome" {
		t.Errorf("callbackURL = %q", sd.CallbackURL)
	}
	if sd.CodeVerifier != verifier {
		t.Errorf("verifier mismatch: %q vs %q", sd.CodeVerifier, verifier)
	}

	// One-time use: the record is consumed, so a second parse must fail.
	cr2 := httptest.NewRequest(http.MethodGet, "/api/auth/callback/fake?state="+state, nil)
	for _, c := range w.Result().Cookies() {
		cr2.AddCookie(c)
	}
	if _, err := p.parseState(httptest.NewRecorder(), cr2); err == nil {
		t.Errorf("expected state to be consumed (one-time use)")
	}
}

func TestParseState_MissingCookie(t *testing.T) {
	p, _ := newTestPlugin(t, Options{})
	w := httptest.NewRecorder()
	state, _, err := p.generateState(w, httptest.NewRequest(http.MethodPost, "/x", nil), stateData{CallbackURL: "http://localhost:3000"})
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	// No cookie attached → invalid.
	cr := httptest.NewRequest(http.MethodGet, "/cb?state="+state, nil)
	if _, err := p.parseState(httptest.NewRecorder(), cr); err != ErrStateInvalid {
		t.Errorf("err = %v, want ErrStateInvalid", err)
	}
}

func TestParseState_UnknownState(t *testing.T) {
	p, _ := newTestPlugin(t, Options{})
	cr := httptest.NewRequest(http.MethodGet, "/cb?state=does-not-exist", nil)
	if _, err := p.parseState(httptest.NewRecorder(), cr); err != ErrStateInvalid {
		t.Errorf("err = %v, want ErrStateInvalid", err)
	}
}

func TestParseState_Expired(t *testing.T) {
	p, _ := newTestPlugin(t, Options{StateTTL: -time.Second})
	w := httptest.NewRecorder()
	state, _, err := p.generateState(w, httptest.NewRequest(http.MethodPost, "/x", nil), stateData{CallbackURL: "http://localhost:3000"})
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	cr := httptest.NewRequest(http.MethodGet, "/cb?state="+state, nil)
	for _, c := range w.Result().Cookies() {
		cr.AddCookie(c)
	}
	if _, err := p.parseState(httptest.NewRecorder(), cr); err != ErrStateInvalid {
		t.Errorf("err = %v, want ErrStateInvalid (expired)", err)
	}
}
