// Package access provides a small, reusable role-based access-control (RBAC)
// model: statements declare the actions available on each resource, and roles
// grant a subset of those actions. It is modeled after better-auth's access
// controller so the admin plugin (and future plugins) can share one mechanism.
package access

// Statements maps a resource name to the actions defined/allowed on it,
// e.g. {"user": {"create", "ban"}, "session": {"revoke"}}.
type Statements map[string][]string

// Controller is the full set of statements you derive roles from.
type Controller struct {
	statements Statements
}

// Role grants a subset of a Controller's actions to a principal.
type Role struct {
	statements Statements
}

// New creates an access controller from the full statement set.
func New(statements Statements) *Controller {
	return &Controller{statements: statements}
}

// Statements returns the controller's full statement set.
func (ac *Controller) Statements() Statements { return ac.statements }

// NewRole derives a role granting the given subset of actions.
func (ac *Controller) NewRole(perms Statements) *Role {
	return &Role{statements: perms}
}

// NewRole builds a standalone role from an explicit grant. Useful for default
// roles defined without a parent controller.
func NewRole(perms Statements) *Role {
	return &Role{statements: perms}
}

// Authorize reports whether the role satisfies every (connector "AND", the
// default) or any (connector "OR") requested resource/action pair. The returned
// string is a human-readable reason when access is denied.
func (r *Role) Authorize(request Statements, connector string) (bool, string) {
	if connector == "" {
		connector = "AND"
	}
	success := false
	for resource, actions := range request {
		allowed, ok := r.statements[resource]
		if !ok {
			if connector == "AND" {
				return false, "not allowed to access resource: " + resource
			}
			success = false
			continue
		}
		success = len(actions) > 0 && everyIn(actions, allowed)
		if success && connector == "OR" {
			return true, ""
		}
		if !success && connector == "AND" {
			return false, "unauthorized to access resource: " + resource
		}
	}
	if success {
		return true, ""
	}
	return false, "not authorized"
}

func everyIn(needles, haystack []string) bool {
	for _, n := range needles {
		if !contains(haystack, n) {
			return false
		}
	}
	return true
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
