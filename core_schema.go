package goten

// CoreSchema returns the declarative schema for goten's built-in tables
// (users, sessions, accounts, verification). It is the source of truth for the
// `goten generate` CLI, which merges it with each enabled plugin's
// SchemaProvider.Schema() to emit ORM models. The map-based runtime adapter
// reads/writes these same column names, kept in sync via this declaration.
func CoreSchema() map[string]TableSchema {
	return map[string]TableSchema{
		"users": {Fields: []FieldDef{
			{Name: "id", Type: "text", Required: true, PrimaryKey: true},
			{Name: "email", Type: "text", Required: true, Unique: true},
			{Name: "name", Type: "text", Required: true, Default: "''"},
			{Name: "email_verified", Type: "boolean", Required: true, Default: "false"},
			{Name: "image", Type: "text"},
			{Name: "created_at", Type: "timestamp", Required: true},
			{Name: "updated_at", Type: "timestamp", Required: true},
		}},
		"sessions": {Fields: []FieldDef{
			{Name: "id", Type: "text", Required: true, PrimaryKey: true},
			{Name: "token", Type: "text", Required: true, Unique: true},
			{Name: "user_id", Type: "text", Required: true, Index: true, Ref: "users.id"},
			{Name: "expires_at", Type: "timestamp", Required: true, Index: true},
			{Name: "ip_address", Type: "text"},
			{Name: "user_agent", Type: "text"},
			{Name: "created_at", Type: "timestamp", Required: true},
			{Name: "updated_at", Type: "timestamp", Required: true},
		}},
		"accounts": {
			Fields: []FieldDef{
				{Name: "id", Type: "text", Required: true, PrimaryKey: true},
				{Name: "user_id", Type: "text", Required: true, Index: true, Ref: "users.id"},
				{Name: "account_id", Type: "text", Required: true},
				{Name: "provider_id", Type: "text", Required: true},
				{Name: "password", Type: "text"},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
			UniqueTogether: [][]string{{"provider_id", "account_id"}},
		},
		"verification": {Fields: []FieldDef{
			{Name: "id", Type: "text", Required: true, PrimaryKey: true},
			{Name: "identifier", Type: "text", Required: true, Index: true},
			{Name: "value", Type: "text", Required: true},
			{Name: "expires_at", Type: "timestamp", Required: true, Index: true},
			{Name: "created_at", Type: "timestamp", Required: true},
			{Name: "updated_at", Type: "timestamp", Required: true},
		}},
	}
}

// CoreTableOrder lists the core tables in dependency order (referenced tables
// first), so generated models and AllModels() can be emitted/migrated safely.
var CoreTableOrder = []string{"users", "sessions", "accounts", "verification"}
