// =============================================================================
// 文件: internal/module/schedule/handler_reviewtostory.go
// 模块: 排期工作台
// 类型: action
// 职责: 转发禅道主管部门审批提醒检查，供排期弹窗添加研发需求前调用。
// 依赖: internal/module/schedule/service_reviewtostory.go
//       internal/middleware
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// CheckReviewToStoryNotice 返回业需转研发需求前的主管部门审批提醒。
func (h *Handler) CheckReviewToStoryNotice(c *gin.Context) {
	demandID, ok := parseDemandID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "业需 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	notice, err := h.svc.CheckReviewToStoryNotice(c.Request.Context(), actor, demandID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("check review to story notice failed",
				zap.Error(err),
				zap.Uint("demand_id", demandID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	if notice != "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"blocked": true,
			"message": notice,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"blocked": false,
	})
}
