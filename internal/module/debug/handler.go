// =============================================================================
// 文件: internal/module/debug/handler.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 处理 SQL 性能分析页面与请求汇总数据 API。
// 依赖: internal/middleware
//       internal/pkg/perm
// =============================================================================

package debug

import (
	"net/http"
	"workbench/internal/constants"

	"github.com/gin-gonic/gin"
)

// Handler SQL 性能分析 Handler。
type Handler struct {
	svc *Service
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/sqlperf")
	g.GET("", h.List)
	g.GET("/requests", h.Requests)
}

// List 性能分析页面。
func (h *Handler) List(c *gin.Context) {
	c.File(constants.TEMPLATE_DEBUG_SQLPERF)
}

// Requests 返回 sql.log 中的请求汇总数据。
func (h *Handler) Requests(c *gin.Context) {
	var req RequestsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数解析失败",
		})
		return
	}

	resp, err := h.svc.Requests(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取 SQL 日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"requests": resp.Requests,
	})
}
