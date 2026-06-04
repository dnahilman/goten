package oauth

import (
	"context"
	"net/url"
	"sync"
	"testing"

	goten "github.com/dnahilman/goten"
	adp "github.com/dnahilman/goten/adapter"
)

// mockAdapter is a tiny in-memory Adapter for white-box tests. It lives here
// (rather than reusing test/testutil) to avoid an oauth → test → oauth module cycle.
type mockAdapter struct {
	mu   sync.RWMutex
	data map[string][]map[string]any
}

func newMockAdapter() *mockAdapter { return &mockAdapter{data: map[string][]map[string]any{}} }

func (m *mockAdapter) FindOne(_ context.Context, model string, q adp.Query) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, row := range m.data[model] {
		if matches(row, q.Where) {
			return clone(row), nil
		}
	}
	return nil, nil
}

func (m *mockAdapter) FindMany(_ context.Context, model string, q adp.Query) ([]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []map[string]any
	for _, row := range m.data[model] {
		if matches(row, q.Where) {
			out = append(out, clone(row))
		}
	}
	return out, nil
}

func (m *mockAdapter) Create(_ context.Context, model string, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := clone(data)
	m.data[model] = append(m.data[model], row)
	return clone(row), nil
}

func (m *mockAdapter) Update(_ context.Context, model string, q adp.Query, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, row := range m.data[model] {
		if matches(row, q.Where) {
			for k, v := range data {
				m.data[model][i][k] = v
			}
			return clone(m.data[model][i]), nil
		}
	}
	return nil, nil
}

func (m *mockAdapter) Delete(_ context.Context, model string, q adp.Query) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []map[string]any
	for _, row := range m.data[model] {
		if !matches(row, q.Where) {
			kept = append(kept, row)
		}
	}
	m.data[model] = kept
	return nil
}

func (m *mockAdapter) Count(ctx context.Context, model string, q adp.Query) (int64, error) {
	rows, err := m.FindMany(ctx, model, q)
	return int64(len(rows)), err
}

func matches(row map[string]any, wheres []adp.Where) bool {
	for _, w := range wheres {
		if w.Operator == "=" && row[w.Field] != w.Value {
			return false
		}
	}
	return true
}

func clone(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// newTestPlugin builds an Auth wired with the oauth plugin over an in-memory adapter.
func newTestPlugin(t *testing.T, opts Options) (*Plugin, *goten.Auth) {
	t.Helper()
	p := New(opts)
	a, err := goten.New(goten.Config{
		BaseURL:        "http://localhost:8080",
		Secret:         "test-secret-key-that-is-32-bytes!",
		Adapter:        newMockAdapter(),
		TrustedOrigins: []string{"http://localhost:3000"},
		Plugins:        []goten.Plugin{p},
	})
	if err != nil {
		t.Fatalf("goten.New: %v", err)
	}
	return p, a
}

// fakeProvider is a network-free oauth.Provider for tests.
type fakeProvider struct {
	id       string
	tokens   *Tokens
	tokenErr error
	info     *UserInfo
	infoErr  error
}

func (f *fakeProvider) ID() string { return f.id }

func (f *fakeProvider) CreateAuthorizationURL(p AuthURLParams) (string, error) {
	return "https://idp.example/auth?state=" + url.QueryEscape(p.State) +
		"&code_challenge=" + CodeChallengeS256(p.CodeVerifier) +
		"&redirect_uri=" + url.QueryEscape(p.RedirectURI), nil
}

func (f *fakeProvider) ValidateAuthorizationCode(p CodeExchangeParams) (*Tokens, error) {
	return f.tokens, f.tokenErr
}

func (f *fakeProvider) GetUserInfo(t *Tokens) (*UserInfo, error) { return f.info, f.infoErr }

var _ Provider = (*fakeProvider)(nil)
