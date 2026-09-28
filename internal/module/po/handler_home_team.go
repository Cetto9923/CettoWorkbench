package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TeamHomeVersionWindows returns version-window data only after resolving the
// selected scope to agile groups the current actor is authorized to view.
func (h *Handler) TeamHomeVersionWindows(c *gin.Context) {
	groupIDs, _, _, ok := h.resolveDashboardGroupSelection(c)
	if !ok {
		return
	}
	scope := c.Query("scope")
	scopeID, _ := parseTeamScopeID(c.Query("scopeId"))
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
	groupIDs, accounts, _, ok := h.resolveDashboardSelection(c)
	if !ok {
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
