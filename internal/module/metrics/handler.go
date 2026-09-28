// =============================================================================
// 文件: internal/module/metrics/handler.go
// 模块: 指标管理 (metrics)
// 类型: handler
// 职责: 指标管理 / 雷达页面渲染与 List JSON；按 AGENTS §3 MUST 挂 capability permission。
// 依赖: internal/middleware
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package metrics

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理 /metrics/{manage,radar,api} 等路由；capability 复用 perm.PoHomeList。
type Handler struct {
	renderer *render.Renderer
	service  *Service
	logger   *zap.Logger
}

// NewHandler 装配 Handler。
func NewHandler(renderer *render.Renderer, service *Service, logger *zap.Logger) *Handler {
	return &Handler{renderer: renderer, service: service, logger: logger}
}

// RegisterRoutes 注册路由到 rg（已带 RequireLogin + RecordOperationLog）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/metrics")
	g.GET("/manage", middleware.RequirePerm(perm.PoHomeList), h.Manage)
	g.GET("/radar", middleware.RequirePerm(perm.PoHomeList), h.Radar)
	g.GET("/api", middleware.RequirePerm(perm.PoHomeList), h.API)
}

// Manage 渲染指标管理页面（占位骨架在 metrics/manage.html）。
func (h *Handler) Manage(c *gin.Context) {
	h.page(c, "指标管理", "业务与研发效能度量指标配置与生命周期管理", "metrics/manage")
}

// Radar 渲染指标雷达页面（雷达页 agent 维护）。
func (h *Handler) Radar(c *gin.Context) {
	h.page(c, "指标雷达", "多维度敏捷研发效能雷达大屏与异常监控", "metrics/radar")
}

func (h *Handler) page(c *gin.Context, title, desc, template string) {
	if h.renderer == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	render.Page(c, http.StatusOK, template, gin.H{"Title": title, "PageTitle": title, "PageDescription": desc})
}

// API 返回指标管理列表 JSON：success/summary/items/filter/total/page/pageSize。
// 失败统一 503 + 短消息，区别于正常 200。
func (h *Handler) API(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))
	req := ManageListReq{
		Category: c.Query("category"),
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: size,
	}
	resp, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("metrics list failed", zap.Error(err))
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "禅道指标数据暂不可用",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"summary":  resp.Summary,
		"items":    resp.Items,
		"filter":   resp.Filter,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}
