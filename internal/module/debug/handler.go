// =============================================================================
// 文件: internal/module/debug/handler.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 处理 SQL 性能分析 / 慢 SQL 明细页面与数据 API。
// 依赖: internal/constants
// =============================================================================

package debug

import (
	"net/http"
	"strings"
	"time"
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
	perf := rg.Group("/sqlperf")
	perf.GET("", h.List)
	perf.GET("/requests", h.Requests)

	sqllog := rg.Group("/sqllog")
	sqllog.GET("", h.SQLLogPage)
	sqllog.GET("/queries", h.Queries)
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

// SQLLogPage 慢 SQL 明细页面。
func (h *Handler) SQLLogPage(c *gin.Context) {
	c.File(constants.TEMPLATE_DEBUG_SQLLOG)
}

// Queries 返回指定日期 sql-YYYY-MM-DD.log 中的 SQL 明细（按耗时降序）。
func (h *Handler) Queries(c *gin.Context) {
	var req QueriesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数解析失败",
		})
		return
	}
	if strings.TrimSpace(req.Date) == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	resp, err := h.svc.Queries(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取 SQL 日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"date":    resp.Date,
		"total":   resp.Total,
		"queries": resp.Queries,
	})
}
