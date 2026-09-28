package po

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"workbench/internal/middleware"
)

// resolveDashboardGroupSelection authorizes the requested parent scope first,
// then restricts an optional subgroup to that parent's resolved group set.
func (h *Handler) resolveDashboardGroupSelection(c *gin.Context) ([]uint, string, uint, bool) {
	actor := middleware.CurrentUser(c)
	if actor == nil || h.teamViewAccess == nil || h.teamScopeGroupIDs == nil {
		c.AbortWithStatus(403)
		return nil, "", 0, false
	}
	allowed, err := h.teamViewAccess(c.Request.Context(), actor)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return nil, "", 0, false
	}
	if !allowed {
		c.AbortWithStatus(403)
		return nil, "", 0, false
	}
	id, err := parseTeamScopeID(c.Query("scopeId"))
	if err != nil {
		c.AbortWithStatus(400)
		return nil, "", 0, false
	}
	scope := c.Query("scope")
	if scope == "" {
		scope = "team"
	}
	ids, err := h.teamScopeGroupIDs(c.Request.Context(), actor, scope, id)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return nil, "", 0, false
	}
	sub, err := parseTeamScopeID(c.Query("subteamgroupId"))
	if err != nil {
		c.AbortWithStatus(400)
		return nil, "", 0, false
	}
	if sub > 0 {
		found := false
		for _, group := range ids {
			if group == sub {
				found = true
			}
		}
		if !found {
			c.AbortWithStatusJSON(403, gin.H{"message": "敏捷小组不在所选授权范围内"})
			return nil, "", 0, false
		}
		ids = []uint{sub}
		scope, id = "team", sub
	}
	return ids, scope, id, true
}

// resolveDashboardSelection adds the authorized member accounts required by
// team views that include task assignments.
func (h *Handler) resolveDashboardSelection(c *gin.Context) ([]uint, []string, uint, bool) {
	ids, scope, id, ok := h.resolveDashboardGroupSelection(c)
	if !ok {
		return nil, nil, 0, false
	}
	actor := middleware.CurrentUser(c)
	if h.teamScopeAccounts == nil {
		c.AbortWithStatus(403)
		return nil, nil, 0, false
	}
	accounts, err := h.teamScopeAccounts(c.Request.Context(), actor, scope, id)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return nil, nil, 0, false
	}
	return ids, accounts, id, true
}

func (h *Handler) TeamDashboard(c *gin.Context) {
	ids, accounts, detail, ok := h.resolveDashboardSelection(c)
	if !ok {
		return
	}
	data, err := h.svc.TeamDashboard(c.Request.Context(), ids, accounts, detail)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
