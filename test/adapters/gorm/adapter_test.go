//go:build integration

package gormadapter_test

import (
	"context"
	"os"
	"strings"
	"testing"

	gormadapter "github.com/dnahilman/goten/adapters/gorm"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goten "github.com/dnahilman/goten"
)

func setupDB(t *testing.T) (*gormadapter.Adapter, func()) {
	t.Helper()
	db, cleanup := testutil.StartPostgres(t)

	// Drop existing tables so tests on a shared DB start with a clean slate.
	require.NoError(t, db.Exec(`
		DROP TABLE IF EXISTS accounts;
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS goten_migrations;
	`).Error)

	// Apply migration
	sql, err := os.ReadFile("../../../migrations/20260520120000_core_initial.up.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(sql)).Error)

	return gormadapter.New(db), cleanup
}

func TestGORMAdapter_CreateAndFindOne(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	ia := goten.NewInternalAdapter(adp)
	u, err := ia.CreateUser(ctx, "test@example.com", "Test User", false)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.True(t, strings.HasPrefix(u.ID, "g10_"))

	found, err := ia.FindUserByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, u.ID, found.ID)
	assert.Equal(t, "Test User", found.Name)
}

func TestGORMAdapter_FindOne_NotFound(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	result, err := adp.FindOne(ctx, "users", goten.Query{
		Where: []goten.Where{goten.EQ("email", "nobody@example.com")},
	})
	require.NoError(t, err)
	assert.Nil(t, result, "FindOne should return nil when not found")
}

func TestGORMAdapter_Update_ZeroValue(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	ia := goten.NewInternalAdapter(adp)
	u, err := ia.CreateUser(ctx, "update@example.com", "Before", true)
	require.NoError(t, err)

	// Update email_verified to false (zero-value bool) — must not be skipped
	updated, err := ia.UpdateUser(ctx, u.ID, map[string]any{"email_verified": false})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.False(t, updated.EmailVerified, "zero-value update should not be skipped")
}

func TestGORMAdapter_Delete(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	ia := goten.NewInternalAdapter(adp)
	u, err := ia.CreateUser(ctx, "delete@example.com", "ToDelete", false)
	require.NoError(t, err)

	require.NoError(t, ia.DeleteUser(ctx, u.ID))

	found, err := ia.FindUserByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGORMAdapter_Count(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	ia := goten.NewInternalAdapter(adp)
	_, _ = ia.CreateUser(ctx, "count1@example.com", "A", false)
	_, _ = ia.CreateUser(ctx, "count2@example.com", "B", false)

	count, err := adp.Count(ctx, "users", goten.Query{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(2))
}

func TestGORMAdapter_InvalidOperator(t *testing.T) {
	adp, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := adp.FindOne(ctx, "users", goten.Query{
		Where: []goten.Where{{Field: "email", Operator: "DROP TABLE", Value: "x"}},
	})
	assert.Error(t, err, "invalid operator should return error")
}
