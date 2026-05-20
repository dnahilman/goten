package goten_test

import (
	"context"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSession_Roundtrip(t *testing.T) {
	sess := &models.Session{ID: "g10_sess-1", Token: "g10_tok", UserID: "g10_user-1"}
	user := &models.User{ID: "g10_user-1", Email: "a@b.com", Name: "Alice"}

	ctx := goten.WithSession(context.Background(), sess, user)

	gotSess, ok := goten.SessionFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, sess.ID, gotSess.ID)

	gotUser, ok := goten.UserFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, user.Email, gotUser.Email)
}

func TestSessionFromContext_Empty(t *testing.T) {
	sess, ok := goten.SessionFromContext(context.Background())
	assert.False(t, ok)
	assert.Nil(t, sess)
}

func TestUserFromContext_Empty(t *testing.T) {
	user, ok := goten.UserFromContext(context.Background())
	assert.False(t, ok)
	assert.Nil(t, user)
}

func TestWithSession_DoesNotLeakAcrossContexts(t *testing.T) {
	sess := &models.Session{ID: "g10_sess-2"}
	user := &models.User{ID: "g10_user-2"}

	ctxA := goten.WithSession(context.Background(), sess, user)
	ctxB := context.Background()

	_, okA := goten.SessionFromContext(ctxA)
	_, okB := goten.SessionFromContext(ctxB)

	assert.True(t, okA)
	assert.False(t, okB, "ctxB should not have session from ctxA")
}
