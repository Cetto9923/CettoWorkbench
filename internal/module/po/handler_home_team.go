package po

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// TeamHomeVersionWindows returns version-window data only after resolving the
// selected scope to agile groups the current actor is authorized to view.
func (h *Handler) TeamHomeVersionWindows(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if actor == nil || h.svc == nil || h.teamViewAccess == nil || h.teamScopeGroupIDs == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	allowed, err := h.teamViewAccess(c.Request.Context(), actor)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if !allowed {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	scopeID, err := parseTeamScopeID(c.Query("scopeId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "团队范围参数无效"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	groupIDs, err := h.teamScopeGroupIDs(c.Request.Context(), actor, scope, scopeID)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return
	}
	limit := 5
	if scope == "dept" && scopeID == 0 {
		limit = 8
	}
	cards, err := h.svc.TeamHomeVersionWindows(c.Request.Context(), groupIDs, limit)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load team home version windows", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "版本窗口数据暂不可用"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cards})
}

func (h *Handler) TeamHomeValueStream(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if actor == nil || h.svc == nil || h.teamViewAccess == nil || h.teamScopeGroupIDs == nil || h.teamScopeAccounts == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	allowed, err := h.teamViewAccess(c.Request.Context(), actor)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if !allowed {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	scopeID, err := parseTeamScopeID(c.Query("scopeId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "团队范围参数无效"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	groupIDs, err := h.teamScopeGroupIDs(c.Request.Context(), actor, scope, scopeID)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return
	}
	accounts, err := h.teamScopeAccounts(c.Request.Context(), actor, scope, scopeID)
	if err != nil {
		writeTeamScopeError(c, h.logger, err)
		return
	}
	stages, err := h.svc.TeamHomeValueStream(c.Request.Context(), groupIDs, accounts)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load team home value stream", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "价值流统计暂不可用"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stages})
}
