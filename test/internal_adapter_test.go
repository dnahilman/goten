package goten_test

import (
	"context"
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIA(t *testing.T) *goten.InternalAdapter {
	t.Helper()
	return goten.NewInternalAdapter(testutil.NewMockAdapter())
}

func TestInternalAdapter_CreateUser(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, err := ia.CreateUser(ctx, "alice@example.com", "Alice", false)
	require.NoError(t, err)
	require.NotNil(t, u)

	assert.True(t, strings.HasPrefix(u.ID, "g10_"), "ID must have g10_ prefix, got %s", u.ID)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "Alice", u.Name)
	assert.False(t, u.EmailVerified)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
}

func TestInternalAdapter_FindUserByEmail(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	created, err := ia.CreateUser(ctx, "bob@example.com", "Bob", false)
	require.NoError(t, err)

	found, err := ia.FindUserByEmail(ctx, "bob@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestInternalAdapter_FindUserByEmail_NotFound(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	found, err := ia.FindUserByEmail(ctx, "nobody@example.com")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestInternalAdapter_FindUserByID(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	created, err := ia.CreateUser(ctx, "carol@example.com", "Carol", true)
	require.NoError(t, err)

	found, err := ia.FindUserByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.True(t, found.EmailVerified)
}

func TestInternalAdapter_UpdateUser(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, err := ia.CreateUser(ctx, "dave@example.com", "Dave", false)
	require.NoError(t, err)

	updated, err := ia.UpdateUser(ctx, u.ID, map[string]any{
		"name":           "David",
		"email_verified": true,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "David", updated.Name)
	assert.True(t, updated.EmailVerified)
}

func TestInternalAdapter_DeleteUser(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, err := ia.CreateUser(ctx, "eve@example.com", "Eve", false)
	require.NoError(t, err)

	err = ia.DeleteUser(ctx, u.ID)
	require.NoError(t, err)

	found, err := ia.FindUserByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestInternalAdapter_CreateUserWithExtra(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, err := ia.CreateUserWithExtra(ctx, "frank@example.com", "Frank", false, map[string]any{
		"username": "frank123",
	})
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "frank@example.com", u.Email)
}

func TestInternalAdapter_CreateAccount(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, err := ia.CreateUser(ctx, "grace@example.com", "Grace", false)
	require.NoError(t, err)

	acc, err := ia.CreateAccount(ctx, u.ID, u.ID, "credential", map[string]any{
		"password": "hashed_pw",
	})
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.True(t, strings.HasPrefix(acc.ID, "g10_"))
	assert.Equal(t, u.ID, acc.UserID)
	assert.Equal(t, "credential", acc.ProviderID)
	assert.NotNil(t, acc.Password)
	assert.Equal(t, "hashed_pw", *acc.Password)
}

func TestInternalAdapter_FindAccountByProviderAndID(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, _ := ia.CreateUser(ctx, "henry@example.com", "Henry", false)
	created, err := ia.CreateAccount(ctx, u.ID, u.ID, "credential", nil)
	require.NoError(t, err)

	found, err := ia.FindAccountByProviderAndID(ctx, "credential", u.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestInternalAdapter_FindAccountsByUserID(t *testing.T) {
	ia := newIA(t)
	ctx := context.Background()

	u, _ := ia.CreateUser(ctx, "iris@example.com", "Iris", false)
	_, _ = ia.CreateAccount(ctx, u.ID, u.ID, "credential", nil)
	_, _ = ia.CreateAccount(ctx, u.ID, "google-123", "google", nil)

	accounts, err := ia.FindAccountsByUserID(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
}
