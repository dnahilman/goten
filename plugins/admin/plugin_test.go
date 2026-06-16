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

func TestCreateUserGetUserAndSignIn(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	// create
	rec, out := do(t, h, "POST", "/api/auth/admin/create-user", adminTok, map[string]any{
		"email": "new@x.io", "password": "supersecret", "name": "New", "role": "user",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("create-user: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	u, _ := out["user"].(map[string]any)
	if u["email"] != "new@x.io" || u["role"] != "user" {
		t.Fatalf("created user wrong: %v", u)
	}
	newID, _ := u["id"].(string)

	// the new user can sign in via the core email endpoint
	rec, _ = do(t, h, "POST", "/api/auth/sign-in/email", "", map[string]any{
		"email": "new@x.io", "password": "supersecret",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("sign-in created user: want 200, got %d (%s)", rec.Code, rec.Body)
	}

	// get-user round-trips
	rec, out = do(t, h, "POST", "/api/auth/admin/get-user", adminTok, map[string]any{"userId": newID})
	if rec.Code != http.StatusOK {
		t.Fatalf("get-user: want 200, got %d", rec.Code)
	}
	if g, _ := out["user"].(map[string]any); g["id"] != newID {
		t.Errorf("get-user returned wrong id: %v", g["id"])
	}

	// get-user not found
	rec, _ = do(t, h, "POST", "/api/auth/admin/get-user", adminTok, map[string]any{"userId": "nope"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get-user missing: want 404, got %d", rec.Code)
	}
}

func TestCreateUserValidationAndPermission(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"invalid email", map[string]any{"email": "nope", "password": "supersecret"}, http.StatusBadRequest},
		{"short password", map[string]any{"email": "a@x.io", "password": "short"}, http.StatusBadRequest},
		{"unknown role", map[string]any{"email": "b@x.io", "password": "supersecret", "role": "wizard"}, http.StatusBadRequest},
	}
	for _, c := range cases {
		rec, _ := do(t, h, "POST", "/api/auth/admin/create-user", adminTok, c.body)
		if rec.Code != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, rec.Code)
		}
	}

	// duplicate email → 409
	rec, _ := do(t, h, "POST", "/api/auth/admin/create-user", adminTok,
		map[string]any{"email": "user@x.io", "password": "supersecret"})
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate email: want 409, got %d", rec.Code)
	}

	// non-admin cannot create
	rec, _ = do(t, h, "POST", "/api/auth/admin/create-user", tokenFor(t, auth, "user-1"),
		map[string]any{"email": "c@x.io", "password": "supersecret"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-admin create: want 403, got %d", rec.Code)
	}
}

func TestListUsersAndSearch(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "alice", "alice@x.io", "user")
	seedUser(mem, "bob", "bob@x.io", "user")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	rec, out := do(t, h, "GET", "/api/auth/admin/list-users", adminTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list-users: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	if total, _ := out["total"].(float64); int(total) != 3 {
		t.Errorf("total: want 3, got %v", out["total"])
	}
	if users, _ := out["users"].([]any); len(users) != 3 {
		t.Errorf("users len: want 3, got %d", len(users))
	}

	// search filters by email substring
	rec, out = do(t, h, "GET", "/api/auth/admin/list-users?search=alice", adminTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list-users search: %d", rec.Code)
	}
	if users, _ := out["users"].([]any); len(users) != 1 {
		t.Errorf("search alice: want 1, got %d", len(users))
	}

	// non-admin denied
	seedUser(mem, "user-1", "u1@x.io", "user")
	rec, _ = do(t, h, "GET", "/api/auth/admin/list-users", tokenFor(t, auth, "user-1"), nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-admin list: want 403, got %d", rec.Code)
	}
}

func TestUpdateUserWhitelist(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	// name is updatable; role must be ignored (it's set via set-role)
	rec, _ := do(t, h, "POST", "/api/auth/admin/update-user", adminTok, map[string]any{
		"userId": "user-1",
		"data":   map[string]any{"name": "Renamed", "role": "admin"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update-user: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	rec, out := do(t, h, "POST", "/api/auth/admin/get-user", adminTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("get-user: %d", rec.Code)
	}
	u, _ := out["user"].(map[string]any)
	if u["name"] != "Renamed" {
		t.Errorf("name not updated: %v", u["name"])
	}
	if u["role"] != "user" {
		t.Errorf("role should not be updatable via update-user, got %v", u["role"])
	}

	// no updatable fields → 400
	rec, _ = do(t, h, "POST", "/api/auth/admin/update-user", adminTok, map[string]any{
		"userId": "user-1", "data": map[string]any{"banned": true},
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("update with no allowed fields: want 400, got %d", rec.Code)
	}
}

func TestSetUserPassword(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	// create a user with a known password
	rec, out := do(t, h, "POST", "/api/auth/admin/create-user", adminTok, map[string]any{
		"email": "pw@x.io", "password": "oldpassword", "name": "PW",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("create-user: %d (%s)", rec.Code, rec.Body)
	}
	id, _ := out["user"].(map[string]any)["id"].(string)

	// change the password
	rec, _ = do(t, h, "POST", "/api/auth/admin/set-user-password", adminTok, map[string]any{
		"userId": id, "newPassword": "newpassword1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("set-user-password: %d (%s)", rec.Code, rec.Body)
	}

	// old password fails, new password works
	rec, _ = do(t, h, "POST", "/api/auth/sign-in/email", "", map[string]any{"email": "pw@x.io", "password": "oldpassword"})
	if rec.Code == http.StatusOK {
		t.Error("old password should no longer work")
	}
	rec, _ = do(t, h, "POST", "/api/auth/sign-in/email", "", map[string]any{"email": "pw@x.io", "password": "newpassword1"})
	if rec.Code != http.StatusOK {
		t.Errorf("new password should work, got %d (%s)", rec.Code, rec.Body)
	}

	// too-short password rejected
	rec, _ = do(t, h, "POST", "/api/auth/admin/set-user-password", adminTok, map[string]any{
		"userId": id, "newPassword": "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("short password: want 400, got %d", rec.Code)
	}
}

func TestRemoveUserAndSelfGuards(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	victimTok := tokenFor(t, auth, "user-1")
	rec, _ := do(t, h, "POST", "/api/auth/admin/remove-user", adminTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("remove-user: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	// user gone + sessions revoked
	rec, _ = do(t, h, "POST", "/api/auth/admin/get-user", adminTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("removed user get: want 404, got %d", rec.Code)
	}
	if _, err := auth.Sessions().Validate(context.Background(), victimTok); err == nil {
		t.Error("removed user's session should be revoked")
	}

	// cannot remove self
	rec, _ = do(t, h, "POST", "/api/auth/admin/remove-user", adminTok, map[string]any{"userId": "admin-1"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("remove self: want 400, got %d", rec.Code)
	}
	// cannot ban self
	rec, _ = do(t, h, "POST", "/api/auth/admin/ban-user", adminTok, map[string]any{"userId": "admin-1"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ban self: want 400, got %d", rec.Code)
	}
}

func TestSessionManagementEndpoints(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	s1 := tokenFor(t, auth, "user-1")
	_ = tokenFor(t, auth, "user-1") // second session

	// list → 2
	rec, out := do(t, h, "POST", "/api/auth/admin/list-user-sessions", adminTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("list-user-sessions: %d (%s)", rec.Code, rec.Body)
	}
	sessions, _ := out["sessions"].([]any)
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}

	// revoke one by id
	s1row, _ := mem.FindOne(context.Background(), "sessions", goten.Query{Where: []goten.Where{goten.EQ("token", s1)}})
	rec, _ = do(t, h, "POST", "/api/auth/admin/revoke-user-session", adminTok, map[string]any{"sessionId": s1row["id"]})
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke-user-session: %d", rec.Code)
	}
	_, out = do(t, h, "POST", "/api/auth/admin/list-user-sessions", adminTok, map[string]any{"userId": "user-1"})
	if sessions, _ := out["sessions"].([]any); len(sessions) != 1 {
		t.Errorf("after revoke one: want 1, got %d", len(sessions))
	}

	// revoke all
	rec, _ = do(t, h, "POST", "/api/auth/admin/revoke-user-sessions", adminTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke-user-sessions: %d", rec.Code)
	}
	_, out = do(t, h, "POST", "/api/auth/admin/list-user-sessions", adminTok, map[string]any{"userId": "user-1"})
	if sessions, _ := out["sessions"].([]any); len(sessions) != 0 {
		t.Errorf("after revoke all: want 0, got %d", len(sessions))
	}
}

func TestHasPermissionEndpoint(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "admin-1", "admin@x.io", "admin")
	h := auth.Handler()
	adminTok := tokenFor(t, auth, "admin-1")

	// caller (admin) can ban
	rec, out := do(t, h, "POST", "/api/auth/admin/has-permission", adminTok, map[string]any{
		"permissions": map[string]any{"user": []any{"ban"}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("has-permission: %d (%s)", rec.Code, rec.Body)
	}
	if out["success"] != true {
		t.Error("admin should be allowed to ban")
	}

	// explicit role=user cannot ban
	_, out = do(t, h, "POST", "/api/auth/admin/has-permission", adminTok, map[string]any{
		"permissions": map[string]any{"user": []any{"ban"}},
		"role":        "user",
	})
	if out["success"] != false {
		t.Error("user role should not be allowed to ban")
	}
}

func TestCustomRoleGranularity(t *testing.T) {
	moderator := access.DefaultAC.NewRole(access.Statements{
		"user":    {"list", "get", "ban"},
		"session": {"list"},
	})
	auth, mem, _ := newAuth(t, Options{
		AdminRoles: []string{"admin"},
		Roles: map[string]*access.Role{
			"admin":     access.AdminRole,
			"moderator": moderator,
			"user":      access.UserRole,
		},
	})
	seedUser(mem, "mod-1", "mod@x.io", "moderator")
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()
	modTok := tokenFor(t, auth, "mod-1")

	// moderator can ban
	rec, _ := do(t, h, "POST", "/api/auth/admin/ban-user", modTok, map[string]any{"userId": "user-1"})
	if rec.Code != http.StatusOK {
		t.Errorf("moderator ban: want 200, got %d (%s)", rec.Code, rec.Body)
	}
	// but cannot set-role (lacks user:set-role)
	rec, _ = do(t, h, "POST", "/api/auth/admin/set-role", modTok, map[string]any{"userId": "user-1", "role": "admin"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("moderator set-role: want 403, got %d", rec.Code)
	}
}

func TestStopImpersonatingWhenNotImpersonating(t *testing.T) {
	auth, mem, _ := newAuth(t, Options{})
	seedUser(mem, "user-1", "user@x.io", "user")
	h := auth.Handler()

	rec, _ := do(t, h, "POST", "/api/auth/admin/stop-impersonating", tokenFor(t, auth, "user-1"), nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("stop when not impersonating: want 400, got %d", rec.Code)
	}
}

// authMem extracts the mem adapter from an Auth for tests that didn't capture it.
func authMem(a *goten.Auth) *memAdapter { return a.Adapter().(*memAdapter) }
