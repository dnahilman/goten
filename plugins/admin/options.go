package adminplugin

import (
	"time"

	"github.com/dnahilman/goten/plugins/admin/access"
)

// Options configures the admin plugin.
type Options struct {
	// DefaultRole is assigned to users created without an explicit role.
	// Default "user".
	DefaultRole string
	// AdminRoles are the roles considered "admin" (used for the
	// impersonate-admins guard). Default ["admin"]. Every entry must exist in
	// Roles or Init() fails.
	AdminRoles []string
	// AdminUserIDs always pass permission checks regardless of role — an escape
	// hatch for bootstrapping the first admin.
	AdminUserIDs []string
	// Roles maps role name to its granted permissions. Defaults to
	// access.DefaultRoles ({admin, user}).
	Roles map[string]*access.Role
	// ImpersonationSessionDuration is how long an impersonation session lasts.
	// Default 1 hour.
	ImpersonationSessionDuration time.Duration
	// DefaultBanReason is recorded when ban-user is called without a reason.
	DefaultBanReason string
	// DefaultBanExpiresIn is the ban duration when ban-user is called without
	// one. Zero means the ban never expires.
	DefaultBanExpiresIn time.Duration
	// BannedUserMessage is returned when a banned user attempts to sign in.
	BannedUserMessage string
}

const defaultBannedMessage = "You have been banned from this application. Please contact support if you believe this is an error."

func (o *Options) applyDefaults() {
	if o.DefaultRole == "" {
		o.DefaultRole = "user"
	}
	if len(o.AdminRoles) == 0 {
		o.AdminRoles = []string{"admin"}
	}
	if o.Roles == nil {
		o.Roles = access.DefaultRoles
	}
	if o.ImpersonationSessionDuration == 0 {
		o.ImpersonationSessionDuration = time.Hour
	}
	if o.BannedUserMessage == "" {
		o.BannedUserMessage = defaultBannedMessage
	}
}
