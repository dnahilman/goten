package adminplugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goten "github.com/dnahilman/goten"
	adminplugin "github.com/dnahilman/goten/plugins/admin"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAdminAuth(t *testing.T) (*goten.Auth, *testutil.MockAdapter) {
	t.Helper()
	mem := testutil.NewMockAdapter()
	p := adminplugin.New(adminplugin.Options{})
	a, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  "test-secret-key-that-is-32-bytes!",
		Adapter: mem,
		Plugins: []goten.Plugin{p},
	})
	require.NoError(t, err)
	return a, mem
}

func seed(t *testing.T, mem *testutil.MockAdapter, id, email, role string) {
	t.Helper()
	now := time.Now().UTC()
	_, err := mem.Create(context.Background(), "users", map[string]any{
		"id": id, "email": email, "name": "", "email_verified": false,
		"role": role, "created_at": now, "updated_at": now,
	})
	require.NoError(t, err)
}

func bearer(t *testing.T, a *goten.Auth, userID string) string {
	t.Helper()
	s, err := a.Sessions().Create(context.Background(), userID, "", "")
	require.NoError(t, err)
	return s.Token
}

func call(t *testing.T, h http.Handler, method, path, token string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(method, path, rdr)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestSetRole_PermissionMatrix(t *testing.T) {
	a, mem := newAdminAuth(t)
	seed(t, mem, "admin-1", "admin@x.io", "admin")
	seed(t, mem, "user-1", "user@x.io", "user")
	seed(t, mem, "user-2", "user2@x.io", "user")
	h := a.Handler()

	// non-admin → 403 (checked before promotion)
	w, _ := call(t, h, http.MethodPost, "/api/auth/admin/set-role", bearer(t, a, "user-2"),
		map[string]any{"userId": "admin-1", "role": "user"})
	assert.Equal(t, http.StatusForbidden, w.Code)

	// admin → 200
	w, _ = call(t, h, http.MethodPost, "/api/auth/admin/set-role", bearer(t, a, "admin-1"),
		map[string]any{"userId": "user-1", "role": "admin"})
	assert.Equal(t, http.StatusOK, w.Code)

	// anonymous → 401
	w, _ = call(t, h, http.MethodPost, "/api/auth/admin/set-role", "",
		map[string]any{"userId": "user-1", "role": "user"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBanBlocksSignIn(t *testing.T) {
	a, mem := newAdminAuth(t)
	seed(t, mem, "admin-1", "admin@x.io", "admin")
	h := a.Handler()
	adminTok := bearer(t, a, "admin-1")

	// create a normal user with a password
	w, out := call(t, h, http.MethodPost, "/api/auth/admin/create-user", adminTok, map[string]any{
		"email": "victim@x.io", "password": "supersecret", "name": "Victim",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	victimID, _ := out["user"].(map[string]any)["id"].(string)
	require.NotEmpty(t, victimID)

	// sign-in works before ban
	w, _ = call(t, h, http.MethodPost, "/api/auth/sign-in/email", "",
		map[string]any{"email": "victim@x.io", "password": "supersecret"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// ban
	w, _ = call(t, h, http.MethodPost, "/api/auth/admin/ban-user", adminTok,
		map[string]any{"userId": victimID, "banReason": "spam"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// sign-in now blocked by the ban hook
	w, _ = call(t, h, http.MethodPost, "/api/auth/sign-in/email", "",
		map[string]any{"email": "victim@x.io", "password": "supersecret"})
	assert.Equal(t, http.StatusForbidden, w.Code)

	// unban → sign-in works again
	w, _ = call(t, h, http.MethodPost, "/api/auth/admin/unban-user", adminTok,
		map[string]any{"userId": victimID})
	require.Equal(t, http.StatusOK, w.Code)
	w, _ = call(t, h, http.MethodPost, "/api/auth/sign-in/email", "",
		map[string]any{"email": "victim@x.io", "password": "supersecret"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestImpersonateRoundTrip(t *testing.T) {
	a, mem := newAdminAuth(t)
	seed(t, mem, "admin-1", "admin@x.io", "admin")
	seed(t, mem, "user-1", "user@x.io", "user")
	h := a.Handler()

	w, out := call(t, h, http.MethodPost, "/api/auth/admin/impersonate-user", bearer(t, a, "admin-1"),
		map[string]any{"userId": "user-1"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	impTok, _ := out["session"].(map[string]any)["token"].(string)
	require.NotEmpty(t, impTok)

	row, err := mem.FindOne(context.Background(), "sessions",
		goten.Query{Where: []goten.Where{goten.EQ("token", impTok)}})
	require.NoError(t, err)
	assert.Equal(t, "user-1", row["user_id"])
	assert.Equal(t, "admin-1", row["impersonated_by"])

	w, out = call(t, h, http.MethodPost, "/api/auth/admin/stop-impersonating", impTok, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "admin-1", out["user"].(map[string]any)["id"])
}
