// =============================================================================
// 文件: internal/module/po/handler_badges.go
// 模块: PO 工作台
// 类型: action
// 职责: 只读补取导航计数，不阻塞页面首次渲染。
// =============================================================================
package po

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"workbench/internal/middleware"
)

func (h *Handler) NavigationBadges(c *gin.Context) {
	badges, err := h.svc.SidebarBadges(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "计数加载失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": badges})
}
