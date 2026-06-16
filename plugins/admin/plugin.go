// Package adminplugin implements goten's admin plugin: role management,
// banning, admin-side user CRUD, session management, impersonation, and a
// reusable RBAC model. Modeled after better-auth's admin plugin.
//
// Because goten's typed models.User/Session do not carry plugin columns, the
// plugin reads and writes its columns (role, banned, …, impersonated_by) via
// the raw adapter map API.
package adminplugin

import (
	"errors"
	"fmt"
	"time"

	goten "github.com/dnahilman/goten"
)

// Plugin implements the admin plugin.
type Plugin struct {
	opts Options
	auth *goten.Auth
}

// New creates the admin plugin with the given options.
func New(opts Options) *Plugin {
	opts.applyDefaults()
	return &Plugin{opts: opts}
}

func (p *Plugin) ID() string { return "admin" }

func (p *Plugin) SetAuth(a *goten.Auth) { p.auth = a }

// Init validates that every configured admin role exists in Roles.
func (p *Plugin) Init() error {
	for _, ar := range p.opts.AdminRoles {
		if _, ok := p.opts.Roles[ar]; !ok {
			return fmt.Errorf("admin: adminRole %q is not defined in Roles", ar)
		}
	}
	return nil
}

// Schema adds the admin columns to the core tables.
func (p *Plugin) Schema() map[string]goten.TableSchema {
	return map[string]goten.TableSchema{
		"users": {Fields: []goten.FieldDef{
			{Name: "role", Type: "text", Default: "'user'"},
			{Name: "banned", Type: "boolean", Default: "false"},
			{Name: "ban_reason", Type: "text"},
			{Name: "ban_expires", Type: "timestamp"},
		}},
		"sessions": {Fields: []goten.FieldDef{
			{Name: "impersonated_by", Type: "text"},
		}},
	}
}

// Endpoints registers the admin routes (mounted under BasePath + /admin/*).
func (p *Plugin) Endpoints() []goten.Endpoint {
	return []goten.Endpoint{
		{Method: "POST", Path: "/admin/set-role", Handler: p.handleSetRole},
		{Method: "POST", Path: "/admin/create-user", Handler: p.handleCreateUser},
		{Method: "POST", Path: "/admin/get-user", Handler: p.handleGetUser},
		{Method: "GET", Path: "/admin/list-users", Handler: p.handleListUsers},
		{Method: "POST", Path: "/admin/update-user", Handler: p.handleUpdateUser},
		{Method: "POST", Path: "/admin/set-user-password", Handler: p.handleSetUserPassword},
		{Method: "POST", Path: "/admin/remove-user", Handler: p.handleRemoveUser},
		{Method: "POST", Path: "/admin/ban-user", Handler: p.handleBanUser},
		{Method: "POST", Path: "/admin/unban-user", Handler: p.handleUnbanUser},
		{Method: "POST", Path: "/admin/impersonate-user", Handler: p.handleImpersonate},
		{Method: "POST", Path: "/admin/stop-impersonating", Handler: p.handleStopImpersonating},
		{Method: "POST", Path: "/admin/list-user-sessions", Handler: p.handleListUserSessions},
		{Method: "POST", Path: "/admin/revoke-user-session", Handler: p.handleRevokeUserSession},
		{Method: "POST", Path: "/admin/revoke-user-sessions", Handler: p.handleRevokeUserSessions},
		{Method: "POST", Path: "/admin/has-permission", Handler: p.handleHasPermission},
	}
}

// UserCreateHooks assigns the default role to new users that don't set one.
func (p *Plugin) UserCreateHooks() []goten.UserCreateHookFn {
	return []goten.UserCreateHookFn{
		func(data map[string]any) map[string]any {
			if _, ok := data["role"]; !ok {
				data["role"] = p.opts.DefaultRole
			}
			return data
		},
	}
}

// SessionCreateHooks enforces bans at sign-in time.
func (p *Plugin) SessionCreateHooks() []goten.SessionCreateHookFn {
	return []goten.SessionCreateHookFn{p.enforceBan}
}

// enforceBan vetoes session creation for a banned user. When the ban has
// expired it auto-unbans and allows the session.
func (p *Plugin) enforceBan(ctx goten.SessionCreateContext) error {
	c := ctx.Request.Context()
	rec, err := p.userRecord(c, ctx.UserID)
	if err != nil || rec == nil {
		return nil
	}
	if banned, _ := rec["banned"].(bool); !banned {
		return nil
	}
	if exp, ok := rec["ban_expires"].(time.Time); ok && !exp.IsZero() && time.Now().UTC().After(exp) {
		_, _ = p.auth.InternalAdapter().UpdateUser(c, ctx.UserID, map[string]any{
			"banned":      false,
			"ban_reason":  nil,
			"ban_expires": nil,
		})
		return nil
	}
	return errors.New(p.opts.BannedUserMessage)
}

// Compile-time interface checks.
var (
	_ goten.Plugin                    = (*Plugin)(nil)
	_ goten.AuthAware                 = (*Plugin)(nil)
	_ goten.Initializer               = (*Plugin)(nil)
	_ goten.SchemaProvider            = (*Plugin)(nil)
	_ goten.EndpointProvider          = (*Plugin)(nil)
	_ goten.UserCreateHookProvider    = (*Plugin)(nil)
	_ goten.SessionCreateHookProvider = (*Plugin)(nil)
)
