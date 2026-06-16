package adminplugin

import (
	"net/http"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/plugins/admin/access"
)

// handleHasPermission reports whether a principal may perform the requested
// actions — useful for gating admin UI. Without userId/role it checks the
// caller; with them it checks that subject (caller must still be authenticated).
func (p *Plugin) handleHasPermission(w http.ResponseWriter, r *http.Request) {
	c, ok := p.requireCaller(w, r)
	if !ok {
		return
	}
	var req struct {
		Permissions map[string][]string `json:"permissions"`
		UserID      string              `json:"userId"`
		Role        string              `json:"role"`
	}
	if err := goten.DecodeJSON(r, &req); err != nil || len(req.Permissions) == 0 {
		goten.WriteError(w, http.StatusBadRequest, codeInvalidBody, "permissions are required")
		return
	}

	checkUserID, checkRole := c.userID, c.role
	if req.UserID != "" {
		checkUserID = req.UserID
		role, _ := p.userRole(r.Context(), req.UserID)
		checkRole = role
	}
	if req.Role != "" {
		checkRole = req.Role
	}

	allowed := p.hasPermission(checkUserID, checkRole, access.Statements(req.Permissions))
	goten.WriteJSON(w, http.StatusOK, map[string]any{"success": allowed})
}
