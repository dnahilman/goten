package access_test

import (
	"testing"

	"github.com/dnahilman/goten/plugins/admin/access"
)

func TestAuthorizeAND(t *testing.T) {
	role := access.NewRole(access.Statements{
		"user":    {"list", "get", "ban"},
		"session": {"list"},
	})

	cases := []struct {
		name string
		req  access.Statements
		want bool
	}{
		{"single allowed", access.Statements{"user": {"ban"}}, true},
		{"multiple allowed", access.Statements{"user": {"list", "get"}}, true},
		{"one disallowed action", access.Statements{"user": {"ban", "delete"}}, false},
		{"unknown resource", access.Statements{"org": {"read"}}, false},
		{"empty actions", access.Statements{"user": {}}, false},
		{"cross resource all allowed", access.Statements{"user": {"list"}, "session": {"list"}}, true},
		{"cross resource one fails", access.Statements{"user": {"list"}, "session": {"revoke"}}, false},
	}
	for _, c := range cases {
		if ok, _ := role.Authorize(c.req, "AND"); ok != c.want {
			t.Errorf("%s: want %v, got %v", c.name, c.want, ok)
		}
	}
}

func TestAuthorizeOR(t *testing.T) {
	role := access.NewRole(access.Statements{"user": {"list"}})

	// OR: at least one resource satisfied
	if ok, _ := role.Authorize(access.Statements{"user": {"list"}, "org": {"read"}}, "OR"); !ok {
		t.Error("OR with one satisfied resource should pass")
	}
	if ok, _ := role.Authorize(access.Statements{"org": {"read"}, "billing": {"read"}}, "OR"); ok {
		t.Error("OR with no satisfied resource should fail")
	}
}

func TestDefaultRoles(t *testing.T) {
	if ok, _ := access.AdminRole.Authorize(access.Statements{"user": {"create", "ban", "delete"}}, "AND"); !ok {
		t.Error("admin should have create/ban/delete")
	}
	if ok, _ := access.AdminRole.Authorize(access.Statements{"user": {"impersonate-admins"}}, "AND"); ok {
		t.Error("admin should NOT have impersonate-admins by default")
	}
	if ok, _ := access.UserRole.Authorize(access.Statements{"user": {"get"}}, "AND"); ok {
		t.Error("user role should grant nothing")
	}
	if _, ok := access.DefaultRoles["admin"]; !ok {
		t.Error("DefaultRoles must contain admin")
	}
	if _, ok := access.DefaultRoles["user"]; !ok {
		t.Error("DefaultRoles must contain user")
	}
}

func TestNewRoleFromController(t *testing.T) {
	ac := access.New(access.Statements{"thing": {"a", "b", "c"}})
	role := ac.NewRole(access.Statements{"thing": {"a", "b"}})
	if ok, _ := role.Authorize(access.Statements{"thing": {"a"}}, "AND"); !ok {
		t.Error("granted action should authorize")
	}
	if ok, _ := role.Authorize(access.Statements{"thing": {"c"}}, "AND"); ok {
		t.Error("ungranted action should not authorize")
	}
}
