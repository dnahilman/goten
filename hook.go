package goten

import (
	"errors"
	"net/http"
)

// ErrHookHandled signals that a hook already wrote the HTTP response.
// Callers must not write again; just return.
var ErrHookHandled = errors.New("hook handled the response")

// UserCreateHookFn is called before inserting a new user.
// Receives and returns the data map so plugins can add or transform fields.
type UserCreateHookFn func(data map[string]any) map[string]any

// SessionCreateHookFn is called before inserting a new session.
// Return a non-nil error to abort. Return ErrHookHandled if the hook already wrote the response.
type SessionCreateHookFn func(ctx SessionCreateContext) error

// SessionCreateContext carries request context for session-create hooks.
type SessionCreateContext struct {
	UserID  string
	Request *http.Request
	Writer  http.ResponseWriter
}
