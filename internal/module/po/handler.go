// =============================================================================
// 文件: internal/module/po/handler.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 HTTP 请求。
// 依赖: internal/middleware
//       internal/pkg/render
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// Handler 处理 PO 工作台页面请求。
type Handler struct{}

// NewHandler 创建 PO 模块 Handler。
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes 注册 PO 工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.Use(middleware.ActiveNav("/po/home"))
	g.GET("/home", h.Home)
}

// Home 渲染 PO 工作台首页
func (h *Handler) Home(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":     "工作台首页",
		"PageTitle": "工作台首页",
	})
}
