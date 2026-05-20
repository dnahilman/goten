package goten_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopPlugin implements every plugin capability interface for testing.
type noopPlugin struct {
	auth         *goten.Auth
	initCalled   bool
	userHooks    []goten.UserCreateHookFn
	sessionHooks []goten.SessionCreateHookFn
	endpoints    []goten.Endpoint
}

func (p *noopPlugin) ID() string                                      { return "noop" }
func (p *noopPlugin) SetAuth(a *goten.Auth)                           { p.auth = a }
func (p *noopPlugin) Init() error                                     { p.initCalled = true; return nil }
func (p *noopPlugin) UserCreateHooks() []goten.UserCreateHookFn       { return p.userHooks }
func (p *noopPlugin) SessionCreateHooks() []goten.SessionCreateHookFn { return p.sessionHooks }
func (p *noopPlugin) Endpoints() []goten.Endpoint                     { return p.endpoints }

// Compile-time interface checks.
var (
	_ goten.Plugin                    = (*noopPlugin)(nil)
	_ goten.AuthAware                 = (*noopPlugin)(nil)
	_ goten.Initializer               = (*noopPlugin)(nil)
	_ goten.UserCreateHookProvider    = (*noopPlugin)(nil)
	_ goten.SessionCreateHookProvider = (*noopPlugin)(nil)
	_ goten.EndpointProvider          = (*noopPlugin)(nil)
)

func newAuthWithPlugin(t *testing.T, p goten.Plugin) *goten.Auth {
	t.Helper()
	a, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
		Plugins: []goten.Plugin{p},
	})
	require.NoError(t, err)
	return a
}

func TestPluginLifecycle_InitCalledAndAuthSet(t *testing.T) {
	p := &noopPlugin{}
	a := newAuthWithPlugin(t, p)

	assert.True(t, p.initCalled, "Init() should be called")
	assert.Same(t, a, p.auth, "SetAuth should pass the same *Auth pointer")
}

func TestPluginLifecycle_UserCreateHooksCompose(t *testing.T) {
	p := &noopPlugin{
		userHooks: []goten.UserCreateHookFn{
			func(d map[string]any) map[string]any { d["a"] = 1; return d },
			func(d map[string]any) map[string]any { d["b"] = 2; return d },
		},
	}
	a := newAuthWithPlugin(t, p)

	result := a.RunUserCreateHooks(map[string]any{})
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
}

func TestPluginLifecycle_SessionCreateHookReject(t *testing.T) {
	p := &noopPlugin{
		sessionHooks: []goten.SessionCreateHookFn{
			func(ctx goten.SessionCreateContext) error {
				return goten.ErrHookHandled
			},
		},
	}
	a := newAuthWithPlugin(t, p)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	err := a.RunSessionCreateHooks(w, r, "user-id")
	assert.ErrorIs(t, err, goten.ErrHookHandled)
}

func TestPluginLifecycle_EndpointRegistered(t *testing.T) {
	p := &noopPlugin{
		endpoints: []goten.Endpoint{
			{
				Method: "GET",
				Path:   "/test/ping",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"pong":true}`))
				},
			},
		},
	}
	a := newAuthWithPlugin(t, p)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/test/ping", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pong")
}

func TestPluginLifecycle_InitError(t *testing.T) {
	type failPlugin struct{ noopPlugin }
	fp := &failPlugin{}
	fp.userHooks = nil
	_, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
		Plugins: []goten.Plugin{&initErrPlugin{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init failed")
	assert.Contains(t, err.Error(), "fail-plugin")
	_ = fp
}

type initErrPlugin struct{}

func (p *initErrPlugin) ID() string   { return "fail-plugin" }
func (p *initErrPlugin) Init() error  { return assert.AnError }

var _ goten.Plugin    = (*initErrPlugin)(nil)
var _ goten.Initializer = (*initErrPlugin)(nil)

func TestPluginLifecycle_NoPlugins(t *testing.T) {
	a, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: testutil.NewMockAdapter(),
	})
	require.NoError(t, err)
	assert.Empty(t, a.Plugins())

	// RunUserCreateHooks with no plugins is a no-op
	data := map[string]any{"key": "value"}
	result := a.RunUserCreateHooks(data)
	assert.Equal(t, "value", result["key"])
}
