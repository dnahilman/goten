package adminplugin_test

import (
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
	adminplugin "github.com/dnahilman/goten/plugins/admin"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ID(t *testing.T) {
	p := adminplugin.New(adminplugin.Options{})
	assert.Equal(t, "admin", p.ID())
}

func TestPlugin_Schema(t *testing.T) {
	p := adminplugin.New(adminplugin.Options{})
	schema := p.Schema()

	users, ok := schema["users"]
	require.True(t, ok, "schema must have 'users' table")
	names := map[string]bool{}
	for _, f := range users.Fields {
		names[f.Name] = true
	}
	for _, want := range []string{"role", "banned", "ban_reason", "ban_expires"} {
		assert.True(t, names[want], "users schema must add %q", want)
	}

	sessions, ok := schema["sessions"]
	require.True(t, ok, "schema must have 'sessions' table")
	require.Len(t, sessions.Fields, 1)
	assert.Equal(t, "impersonated_by", sessions.Fields[0].Name)
}

func TestPlugin_Endpoints(t *testing.T) {
	p := adminplugin.New(adminplugin.Options{})
	eps := p.Endpoints()
	require.Len(t, eps, 15)

	paths := map[string]string{}
	for _, e := range eps {
		paths[e.Path] = e.Method
	}
	assert.Equal(t, "POST", paths["/admin/set-role"])
	assert.Equal(t, "POST", paths["/admin/ban-user"])
	assert.Equal(t, "POST", paths["/admin/impersonate-user"])
	assert.Equal(t, "GET", paths["/admin/list-users"])
}

func TestPlugin_InitRejectsUndefinedAdminRole(t *testing.T) {
	p := adminplugin.New(adminplugin.Options{AdminRoles: []string{"ghost"}})
	_, err := goten.New(goten.Config{
		BaseURL: "http://localhost",
		Secret:  strings.Repeat("x", 32),
		Adapter: testutil.NewMockAdapter(),
		Plugins: []goten.Plugin{p},
	})
	require.Error(t, err, "admin role not defined in Roles should fail Init")
}
