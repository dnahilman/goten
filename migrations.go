package goten

import "embed"

// CoreMigrationsFS holds the core auth schema migrations (users, sessions, accounts).
// The CLI's `goten init` command reads from this FS to bootstrap new projects.
//
//go:embed migrations/*.sql
var CoreMigrationsFS embed.FS
