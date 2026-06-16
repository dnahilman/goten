package access

// DefaultStatements lists every admin resource and the actions defined on it.
// Mirrors better-auth's admin defaultStatements.
var DefaultStatements = Statements{
	"user": {
		"create", "list", "set-role", "ban", "impersonate",
		"impersonate-admins", "delete", "set-password", "get", "update",
	},
	"session": {"list", "revoke", "delete"},
}

// DefaultAC is the access controller built from DefaultStatements.
var DefaultAC = New(DefaultStatements)

// AdminRole grants the full admin permission set. "impersonate-admins" is
// deliberately withheld and must be granted explicitly via a custom role.
var AdminRole = DefaultAC.NewRole(Statements{
	"user":    {"create", "list", "set-role", "ban", "impersonate", "delete", "set-password", "get", "update"},
	"session": {"list", "revoke", "delete"},
})

// UserRole grants no admin permissions.
var UserRole = DefaultAC.NewRole(Statements{
	"user":    {},
	"session": {},
})

// DefaultRoles maps role name to role for the built-in roles. Used when
// Options.Roles is not set.
var DefaultRoles = map[string]*Role{
	"admin": AdminRole,
	"user":  UserRole,
}
