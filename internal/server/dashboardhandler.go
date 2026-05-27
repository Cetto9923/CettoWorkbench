// =============================================================================
// 文件: internal/server/dashboardhandler.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供仪表盘占位页面 Handler。
// 依赖: internal/pkg/render
// =============================================================================

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbrench/internal/pkg/render"
)

// DashboardHandler 渲染仪表盘占位页。
func DashboardHandler(c *gin.Context) {
	render.Page(c, http.StatusOK, "dashboard/index", gin.H{
		"Title":     "仪表盘",
		"PageTitle": "仪表盘",
	})
}
