package goten

import (
	"io/fs"
	"net/http"
)

// Plugin is the base interface — all plugins must implement at minimum ID().
type Plugin interface {
	ID() string
}

// AuthAware — plugin receives a reference to *Auth after New() sets up core state.
type AuthAware interface {
	SetAuth(a *Auth)
}

// Initializer — plugin needs a one-time init step (validate config, connect to external service, etc.).
type Initializer interface {
	Init() error
}

// EndpointProvider — plugin registers additional HTTP endpoints.
type EndpointProvider interface {
	Endpoints() []Endpoint
}

// Endpoint describes a single HTTP endpoint a plugin registers.
type Endpoint struct {
	Method  string // "GET", "POST", etc.
	Path    string // relative path prefixed with BasePath in the router
	Handler http.HandlerFunc
}

// SchemaProvider — plugin declares the DB columns it adds (for CLI introspection).
type SchemaProvider interface {
	Schema() map[string]TableSchema
}

// TableSchema describes columns a plugin adds to a table.
type TableSchema struct {
	Fields []FieldDef
}

// FieldDef describes a single column.
type FieldDef struct {
	Name     string
	Type     string // "text", "boolean", "integer", "timestamp"
	Required bool
	Unique   bool
	Ref      string // FK reference, e.g. "users.id"
}

// MigrationProvider — plugin has embedded SQL migration files.
// The CLI (Issue 006) collects these alongside core migrations, ordered by timestamp.
type MigrationProvider interface {
	Migrations() fs.FS
}

// SessionCreateHookProvider — plugin hooks into session creation.
type SessionCreateHookProvider interface {
	SessionCreateHooks() []SessionCreateHookFn
}

// UserCreateHookProvider — plugin hooks into user creation.
type UserCreateHookProvider interface {
	UserCreateHooks() []UserCreateHookFn
}
