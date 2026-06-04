package usernameplugin_test

import (
	"testing"

	usernameplugin "github.com/dnahilman/goten/plugins/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ID(t *testing.T) {
	p := usernameplugin.New(usernameplugin.Options{})
	assert.Equal(t, "username", p.ID())
}

func TestPlugin_Schema(t *testing.T) {
	p := usernameplugin.New(usernameplugin.Options{})
	schema := p.Schema()
	users, ok := schema["users"]
	require.True(t, ok, "schema must have 'users' table")
	require.Len(t, users.Fields, 1)
	assert.Equal(t, "username", users.Fields[0].Name)
	assert.Equal(t, "text", users.Fields[0].Type)
	assert.True(t, users.Fields[0].Unique)
}

func TestPlugin_Endpoints(t *testing.T) {
	p := usernameplugin.New(usernameplugin.Options{})
	endpoints := p.Endpoints()
	require.Len(t, endpoints, 2)

	paths := map[string]string{}
	for _, e := range endpoints {
		paths[e.Path] = e.Method
	}
	assert.Equal(t, "POST", paths["/sign-up/username"])
	assert.Equal(t, "POST", paths["/sign-in/username"])
}
