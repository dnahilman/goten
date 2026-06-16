package adminplugin

import (
	"strings"

	"github.com/dnahilman/goten/plugins/admin/access"
)

// hasPermission reports whether a caller (identified by id + role) may perform
// the requested actions. Order: (1) AdminUserIDs bypass everything; (2) each
// comma-separated role is looked up in Options.Roles and asked to authorize.
func (p *Plugin) hasPermission(callerUserID, callerRole string, perms access.Statements) bool {
	for _, id := range p.opts.AdminUserIDs {
		if id == callerUserID {
			return true
		}
	}
	role := callerRole
	if role == "" {
		role = p.opts.DefaultRole
	}
	for _, name := range splitAndTrim(role) {
		r, ok := p.opts.Roles[name]
		if !ok {
			continue
		}
		if ok, _ := r.Authorize(perms, "AND"); ok {
			return true
		}
	}
	return false
}

// isAdmin reports whether the user (by id or role) is considered an admin —
// used to gate impersonating other admins.
func (p *Plugin) isAdmin(userID, role string) bool {
	for _, id := range p.opts.AdminUserIDs {
		if id == userID {
			return true
		}
	}
	roleSet := splitAndTrim(role)
	for _, ar := range p.opts.AdminRoles {
		for _, r := range roleSet {
			if r == ar {
				return true
			}
		}
	}
	return false
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
