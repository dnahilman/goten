package adminplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	goten "github.com/dnahilman/goten"
	adp "github.com/dnahilman/goten/adapter"
	"github.com/dnahilman/goten/plugins/admin/access"
)

// --- in-memory adapter (supports "=" and "like") ---

type memAdapter struct {
	mu   sync.RWMutex
	data map[string][]map[string]any
}

var _ adp.Adapter = (*memAdapter)(nil)

func newMem() *memAdapter { return &memAdapter{data: map[string][]map[string]any{}} }

func (m *memAdapter) FindOne(_ context.Context, model string, q adp.Query) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, row := range m.data[model] {
		if match(row, q.Where) {
			return clone(row), nil
		}
	}
	return nil, nil
}

func (m *memAdapter) FindMany(_ context.Context, model string, q adp.Query) ([]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []map[string]any
	for _, row := range m.data[model] {
		if match(row, q.Where) {
			out = append(out, clone(row))
		}
	}
	return out, nil
}

func (m *memAdapter) Create(_ context.Context, model string, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[model] = append(m.data[model], clone(data))
	return clone(data), nil
}

func (m *memAdapter) Update(_ context.Context, model string, q adp.Query, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, row := range m.data[model] {
		if match(row, q.Where) {
			for k, v := range data {
				m.data[model][i][k] = v
			}
			return clone(m.data[model][i]), nil
		}
	}
	return nil, nil
}

func (m *memAdapter) Delete(_ context.Context, model string, q adp.Query) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []map[string]any
	for _, row := range m.data[model] {
		if !match(row, q.Where) {
			kept = append(kept, row)
		}
	}
	m.data[model] = kept
	return nil
}

func (m *memAdapter) Count(ctx context.Context, model string, q adp.Query) (int64, error) {
	rows, err := m.FindMany(ctx, model, q)
	return int64(len(rows)), err
}

func match(row map[string]any, wheres []adp.Where) bool {
	for _, w := range wheres {
		switch w.Operator {
		case "like":
			pat, _ := w.Value.(string)
			s, _ := row[w.Field].(string)
			if !strings.Contains(s, strings.Trim(pat, "%")) {
				return false
			}
		default: // "="
			if row[w.Field] != w.Value {
				return false
			}
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

// --- test harness ---

func newAuth(t *testing.T, opts Options) (*goten.Auth, *memAdapter, *Plugin) {
	t.Helper()
	mem := newMem()
	pl := New(opts)
	auth, err := goten.New(goten.Config{
		BaseURL: "http://localhost:8080",
		Secret:  strings.Repeat("x", 32),
		Adapter: mem,
		Plugins: []goten.Plugin{pl},
	})
	if err != nil {
		t.Fatalf("goten.New: %v", err)
	}
	return auth, mem, pl
}

func seedUser(mem *memAdapter, id, email, role string) {
	now := time.Now().UTC()
	_, _ = mem.Create(context.Background(), "users", map[string]any{
		"id": id, "email": email, "name": "", "email_verified": false,
		"role": role, "created_at": now, "updated_at": now,
	})
}

func tokenFor(t *testing.T, auth *goten.Auth, userID string) string {
	t.Helper()
	sess, err := auth.Sessions().Create(context.Background(), userID, "", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return sess.Token
}

func do(t *testing.T, h http.Handler, method, path, token string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

// --- tests ---

func TestSchemaAddsColumns(t *testing.T) {
	pl := New(Options{})
	sch := pl.Schema()
	users, ok := sch["users"]
	if !ok {
		t.Fatal("missing users schema")
	}
	want := map[string]bool{"role": false, "banned": false, "ban_reason": false, "ban_expires": false}
	for _, f := range users.Fields {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("users schema missing %q", name)
		}
	}
	if len(sch["sessions"].Fields) != 1 || sch["sessions"].Fields[0].Name != "impersonated_by" {
		t.Errorf("sessions schema should add impersonated_by, got %+v", sch["sessions"].Fields)
	}
}

func TestAccessControl(t *testing.T) {
	if ok, _ := access.AdminRole.Authorize(access.Statements{"user": {"ban"}}, "AND"); !ok {
		t.Error("admin should be allowed to ban")
	}
	if ok, _ := access.AdminRole.Authorize(access.Statements{"user": {"impersonate-admins"}}, "AND"); ok {
		t.Error("admin should NOT have impersonate-admins by default")
	}
	if ok, _ := access.UserRole.Authorize(access.Statements{"user": {"ban"}}, "AND"); ok {
		t.Error("user should not be allowed to ban")
	}
}

func TestHasPermissionAndAdminUserIDsBypass(t *testing.T) {
	_, _, pl := newAuth(t, Options{AdminUserIDs: []string{"boss"}})
	if !pl.hasPermission("admin-1", "admin", access.Statements{"user": {"ban"}}) {
		t.Error("admin role should pass")
	}
	if pl.hasPermission("nobody", "user", access.Statements{"user": {"ban"}}) {
		t.Error("user role should fail")
	}
	if !pl.hasPermission("boss", "user", access.Statements{"user": {"ban"}}) {
		t.Error("AdminUserIDs should bypass role")
	}
}

func TestSetRolePermission(t *testing.T) {
	auth, _, _ := newAuth(t, Options{})
	seedUser(authMem(auth), "admin-1", "admin@x.io", "admin")
	seedUser(authMem(auth), "user-1", "user@x.io", "user")
	seedUser(authMem(auth), "user-2", "user2@x.io", "user")
	h := auth.Handler()

	// normal user cannot (checked first, before any promotion)
	rec, _ := do(t, h, "POST", "/api/auth/admin/set-role", tokenFor(t, auth, "user-2"),
		map[string]any{"userId": "admin-1", "role": "user"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("user set-role: want 403, got %d (%s)", rec.Code, rec.Body)
	}
	// admin can set-role
	rec, _ = do(t, h, "POST", "/api/auth/admin/set-role", tokenFor(t, auth, "admin-1"),
		map[string]any{"userId": "user-1", "role": "admin"})
	if rec.Code != http.StatusOK {
		t.Fatalf("admin set-role: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	// unauthenticated
	rec, _ = do(t, h, "POST", "/api/auth/admin/set-role", "", map[string]any{"userId": "x", "role": "user"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon set-role: want 401, got %d", rec.Code)
	}
}

func TestBanRevokesSessionsAndHook(t *testing.T) {
	auth, mem, pl := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()

	// give the target an active session
	victimToken := tokenFor(t, auth, "user-1")

	rec, _ := do(t, h, "POST", "/api/auth/admin/ban-user", tokenFor(t, auth, "admin-1"),
		map[string]any{"userId": "user-1", "banReason": "spam"})
	if rec.Code != http.StatusOK {
		t.Fatalf("ban-user: want 200, got %d (%s)", rec.Code, rec.Body)
	}

	// victim's existing session is revoked
	if _, err := auth.Sessions().Validate(context.Background(), victimToken); err == nil {
		t.Error("banned user's session should be revoked")
	}

	// ban hook vetoes new sessions
	req := httptest.NewRequest("POST", "/", nil)
	if err := pl.enforceBan(goten.SessionCreateContext{UserID: "user-1", Request: req}); err == nil {
		t.Error("enforceBan should veto banned user")
	}
	if err := pl.enforceBan(goten.SessionCreateContext{UserID: "admin-1", Request: req}); err != nil {
		t.Errorf("enforceBan should allow non-banned user, got %v", err)
	}

	// expired ban auto-unbans
	_, _ = mem.Update(context.Background(), "users",
		goten.Query{Where: []goten.Where{goten.EQ("id", "user-1")}},
		map[string]any{"ban_expires": time.Now().UTC().Add(-time.Hour)})
	if err := pl.enforceBan(goten.SessionCreateContext{UserID: "user-1", Request: req}); err != nil {
		t.Errorf("expired ban should auto-unban, got %v", err)
	}
	rec2, out := do(t, h, "POST", "/api/auth/admin/get-user", tokenFor(t, auth, "admin-1"),
		map[string]any{"userId": "user-1"})
	if rec2.Code != http.StatusOK {
		t.Fatalf("get-user: %d", rec2.Code)
	}
	if u, _ := out["user"].(map[string]any); u["banned"] == true {
		t.Error("user should be unbanned after expiry")
	}
}

func TestImpersonateRoundTrip(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()

	rec, out := do(t, h, "POST", "/api/auth/admin/impersonate-user", tokenFor(t, auth, "admin-1"),
		map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonate: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	sess, _ := out["session"].(map[string]any)
	impToken, _ := sess["token"].(string)
	if impToken == "" {
		t.Fatal("no impersonation token returned")
	}

	// the impersonation session belongs to the target and records the admin
	row, _ := mem.FindOne(context.Background(), "sessions",
		goten.Query{Where: []goten.Where{goten.EQ("token", impToken)}})
	if row["user_id"] != "user-1" || row["impersonated_by"] != "admin-1" {
		t.Errorf("impersonation session wrong: %+v", row)
	}

	// stop → fresh admin session
	rec, out = do(t, h, "POST", "/api/auth/admin/stop-impersonating", impToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	u, _ := out["user"].(map[string]any)
	if u["id"] != "admin-1" {
		t.Errorf("stop should return admin user, got %v", u["id"])
	}
}

func TestImpersonateAdminGuard(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "admin-2", "admin2@x.io", "admin")
	h := auth.Handler()

	// default admin role lacks impersonate-admins → 403
	rec, _ := do(t, h, "POST", "/api/auth/admin/impersonate-user", tokenFor(t, auth, "admin-1"),
		map[string]any{"userId": "admin-2"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("impersonating admin should be 403, got %d", rec.Code)
	}
}

// authMem extracts the mem adapter from an Auth for tests that didn't capture it.
func authMem(a *goten.Auth) *memAdapter { return a.Adapter().(*memAdapter) }
