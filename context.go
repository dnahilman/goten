package goten

import (
	"context"

	"github.com/dnahilman/goten/models"
)

type contextKey string

const (
	sessionContextKey contextKey = "goten.session"
	userContextKey    contextKey = "goten.user"
)

func WithSession(ctx context.Context, sess *models.Session, user *models.User) context.Context {
	ctx = context.WithValue(ctx, sessionContextKey, sess)
	ctx = context.WithValue(ctx, userContextKey, user)
	return ctx
}

func SessionFromContext(ctx context.Context) (*models.Session, bool) {
	sess, ok := ctx.Value(sessionContextKey).(*models.Session)
	return sess, ok && sess != nil
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userContextKey).(*models.User)
	return u, ok && u != nil
}
