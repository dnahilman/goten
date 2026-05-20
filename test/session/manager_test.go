package session_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dnahilman/goten/crypto"
	"github.com/dnahilman/goten/session"
	"github.com/dnahilman/goten/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManager(t *testing.T, cfg session.Config) *session.Manager {
	t.Helper()
	return session.NewManager(testutil.NewMockAdapter(), cfg)
}

func TestManager_Create(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	sess, err := m.Create(ctx, "g10_user-1", "127.0.0.1", "Mozilla/5.0")
	require.NoError(t, err)
	require.NotNil(t, sess)

	assert.True(t, strings.HasPrefix(sess.ID, crypto.Prefix), "session ID must have g10_ prefix")
	assert.True(t, strings.HasPrefix(sess.Token, crypto.Prefix), "token must have g10_ prefix")
	assert.Equal(t, "g10_user-1", sess.UserID)
	assert.False(t, sess.ExpiresAt.IsZero())
	assert.True(t, sess.ExpiresAt.After(time.Now()), "expires_at must be in the future")
	require.NotNil(t, sess.IPAddress)
	assert.Equal(t, "127.0.0.1", *sess.IPAddress)
}

func TestManager_Create_DefaultExpiry(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	before := time.Now().UTC().Add(7 * 24 * time.Hour).Add(-time.Second)
	sess, err := m.Create(ctx, "g10_user-2", "", "")
	after := time.Now().UTC().Add(7 * 24 * time.Hour).Add(time.Second)

	require.NoError(t, err)
	assert.True(t, sess.ExpiresAt.After(before) && sess.ExpiresAt.Before(after),
		"default expiry should be ~7 days")
}

func TestManager_FindByToken(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	created, err := m.Create(ctx, "g10_user-3", "", "")
	require.NoError(t, err)

	found, err := m.FindByToken(ctx, created.Token)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

func TestManager_FindByToken_NotFound(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	found, err := m.FindByToken(ctx, "g10_nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestManager_Validate_InvalidPrefix(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	_, err := m.Validate(ctx, "invalid-token-no-prefix")
	assert.ErrorIs(t, err, session.ErrInvalidToken)
}

func TestManager_Validate_NotFound(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	_, err := m.Validate(ctx, "g10_doesnotexist")
	assert.ErrorIs(t, err, session.ErrSessionNotFound)
}

func TestManager_Validate_Expired(t *testing.T) {
	m := newManager(t, session.Config{ExpiresIn: -time.Hour}) // already expired
	ctx := context.Background()

	sess, err := m.Create(ctx, "g10_user-4", "", "")
	require.NoError(t, err)

	_, err = m.Validate(ctx, sess.Token)
	assert.ErrorIs(t, err, session.ErrSessionExpired)

	// Verify cleanup: record should be gone
	found, err := m.FindByToken(ctx, sess.Token)
	require.NoError(t, err)
	assert.Nil(t, found, "expired session should be deleted after Validate")
}

func TestManager_Validate_SlidingRefresh(t *testing.T) {
	// updateAge=0 means every Validate triggers refresh
	m := newManager(t, session.Config{
		ExpiresIn: 7 * 24 * time.Hour,
		UpdateAge: 0,
	})
	ctx := context.Background()

	sess, err := m.Create(ctx, "g10_user-5", "", "")
	require.NoError(t, err)

	original := sess.ExpiresAt

	refreshed, err := m.Validate(ctx, sess.Token)
	require.NoError(t, err)
	require.NotNil(t, refreshed)

	assert.True(t, refreshed.ExpiresAt.After(original) || refreshed.ExpiresAt.Equal(original),
		"expires_at should be extended after sliding refresh")
}

func TestManager_Validate_Valid(t *testing.T) {
	m := newManager(t, session.Config{UpdateAge: 24 * time.Hour})
	ctx := context.Background()

	sess, err := m.Create(ctx, "g10_user-6", "", "")
	require.NoError(t, err)

	validated, err := m.Validate(ctx, sess.Token)
	require.NoError(t, err)
	require.NotNil(t, validated)
	assert.Equal(t, sess.ID, validated.ID)
}

func TestManager_Revoke(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	sess, err := m.Create(ctx, "g10_user-7", "", "")
	require.NoError(t, err)

	require.NoError(t, m.Revoke(ctx, sess.Token))

	found, err := m.FindByToken(ctx, sess.Token)
	require.NoError(t, err)
	assert.Nil(t, found, "revoked session should not be found")
}

func TestManager_RevokeAllForUser(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	s1, _ := m.Create(ctx, "g10_user-8", "", "")
	s2, _ := m.Create(ctx, "g10_user-8", "", "")
	s3, _ := m.Create(ctx, "g10_user-8", "", "")

	// Revoke all except s2
	err := m.RevokeAllForUser(ctx, "g10_user-8", s2.Token)
	require.NoError(t, err)

	f1, _ := m.FindByToken(ctx, s1.Token)
	f2, _ := m.FindByToken(ctx, s2.Token)
	f3, _ := m.FindByToken(ctx, s3.Token)

	assert.Nil(t, f1, "s1 should be revoked")
	assert.NotNil(t, f2, "s2 (except) should still exist")
	assert.Nil(t, f3, "s3 should be revoked")
}

func TestManager_ListByUserID(t *testing.T) {
	m := newManager(t, session.Config{})
	ctx := context.Background()

	_, _ = m.Create(ctx, "g10_user-9", "", "")
	_, _ = m.Create(ctx, "g10_user-9", "", "")
	_, _ = m.Create(ctx, "g10_other-user", "", "")

	sessions, err := m.ListByUserID(ctx, "g10_user-9")
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}
