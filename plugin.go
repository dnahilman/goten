package goten

import (
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

// SchemaProvider — plugin declares the DB columns/tables it adds. This is the
// source of truth for the `goten generate` CLI, which merges these with the core
// schema to emit ORM models (e.g. GORM structs).
type SchemaProvider interface {
	Schema() map[string]TableSchema
}

// TableSchema describes the columns (and table-level constraints) for a table.
type TableSchema struct {
	Fields []FieldDef
	// UniqueTogether lists composite-unique column groups, e.g.
	// {{"provider_id", "account_id"}} for the accounts table.
	UniqueTogether [][]string
}

// FieldDef describes a single column.
type FieldDef struct {
	Name       string
	Type       string // "text", "boolean", "integer", "timestamp"
	Required   bool   // NOT NULL
	Unique     bool   // unique index
	Index      bool   // non-unique index
	PrimaryKey bool   // primary key column (e.g. id)
	Ref        string // foreign key target "table.column" (onDelete CASCADE assumed)
	Default    string // optional DDL default literal, e.g. "false", "''"
}

// SessionCreateHookProvider — plugin hooks into session creation.
type SessionCreateHookProvider interface {
	SessionCreateHooks() []SessionCreateHookFn
}

// UserCreateHookProvider — plugin hooks into user creation.
type UserCreateHookProvider interface {
	UserCreateHooks() []UserCreateHookFn
}
