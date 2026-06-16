package adminplugin

import (
	"context"
	"net/http"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/models"
)

// caller is the authenticated principal making an admin request. role is read
// from the raw users row because models.User does not carry plugin columns.
type caller struct {
	session *models.Session
	userID  string
	role    string
}

// requireCaller validates the request's session and resolves the caller's role.
// On failure it writes a 401 and returns ok=false.
func (p *Plugin) requireCaller(w http.ResponseWriter, r *http.Request) (caller, bool) {
	sess, user, err := p.auth.CurrentSession(r)
	if err != nil || user == nil {
		goten.WriteError(w, http.StatusUnauthorized, codeUnauthorized, "authentication required")
		return caller{}, false
	}
	role, _ := p.userRole(r.Context(), user.ID)
	return caller{session: sess, userID: user.ID, role: role}, true
}

// userRole reads the role column for a user from the raw users row.
func (p *Plugin) userRole(ctx context.Context, userID string) (string, error) {
	rec, err := p.userRecord(ctx, userID)
	if err != nil || rec == nil {
		return "", err
	}
	role, _ := rec["role"].(string)
	return role, nil
}

// userRecord returns the raw users row (including plugin columns role/banned/…),
// or nil when not found. The users table holds no secrets, so the map is safe
// to return to authorized admins as-is.
func (p *Plugin) userRecord(ctx context.Context, userID string) (map[string]any, error) {
	return p.auth.Adapter().FindOne(ctx, "users", goten.Query{
		Where: []goten.Where{goten.EQ("id", userID)},
	})
}
