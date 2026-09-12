// =============================================================================
// 文件: internal/module/follow/handler.go
// 模块: 我的关注
// 类型: action
// 职责: 我的关注列表页 HTTP 请求（当前为静态快照页）。
// 依赖: internal/middleware
//       internal/pkg/render
// =============================================================================

package follow

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// Handler 处理我的关注页面请求。
type Handler struct {
	logger *zap.Logger
}

// NewHandler 创建关注模块 Handler。
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{logger: logger}
}

// RegisterRoutes 注册我的关注路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/follow")
	g.Use(middleware.ActiveNav("/follow"))
	g.GET("", h.List)
}

// List 渲染我的关注静态列表页。
func (h *Handler) List(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_FOLLOW_LIST, gin.H{
		"Title":     "我的关注",
		"PageTitle": "我的关注",
	})
}
