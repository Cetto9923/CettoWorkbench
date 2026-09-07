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

// Handler 处理 /metrics/{manage,radar,api} 三条路由。
// 三条路由均挂 middleware.RequirePerm(perm.PoHomeList) 复用现有 capability，
// 不新增 perm 常量（避免破坏 TestAllPermissionsComplete）。
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
// 注意：模板名以字符串形式直接传入（与 handler_issue_risk.go 一致），
// 未走 constants/templates.go，符合"不触碰 constants/templates.go"边界。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/metrics")
	permMW := middleware.RequirePerm(perm.PoHomeList)
	g.GET("/manage", permMW, h.Manage)
	g.GET("/radar", permMW, h.Radar)
	g.GET("/api", permMW, h.API)
	g.GET("/radar/data", permMW, h.RadarData)
}

// Manage 渲染指标管理页面（占位骨架在 metrics/manage.html）。
func (h *Handler) Manage(c *gin.Context) { h.page(c, "指标管理", "metrics/manage") }

// Radar 渲染指标雷达页面（雷达页 agent 维护）。
func (h *Handler) Radar(c *gin.Context) { h.page(c, "指标雷达", "metrics/radar") }

func (h *Handler) page(c *gin.Context, title, template string) {
	if h.renderer == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	render.Page(c, http.StatusOK, template, gin.H{"Title": title, "PageTitle": title})
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

// RadarData 返回 /metrics/radar 的 JSON 数据：5 分类 KPI + 异常指标分组列表。
// 失败统一 503 + 短消息，与 API 行为一致。
func (h *Handler) RadarData(c *gin.Context) {
	resp, err := h.service.Radar(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("metrics radar failed", zap.Error(err))
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "禅道指标数据暂不可用",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"categories":    resp.Categories,
		"abnormalItems": resp.AbnormalItems,
		"generatedAt":   resp.GeneratedAt,
	})
}
