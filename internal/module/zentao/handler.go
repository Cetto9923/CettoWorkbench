// =============================================================================
// 文件: internal/module/zentao/handler.go
// 模块: 禅道集成
// 类型: action
// 职责: 处理禅道产品接口的 HTTP 请求与响应编排。
// 依赖: internal/pkg/render
//
//	internal/module/zentao/service
//
// =============================================================================
package zentao

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/pkg/render"
)

// Handler 处理禅道模块 HTTP 请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	{
		g.GET("/products", h.Products)
	}
}

// Products 获取禅道产品列表。
func (h *Handler) Products(c *gin.Context) {
	resp, err := h.svc.Products(c.Request.Context(), ProductsReq{})
	if err != nil {
		h.logger.Error("fetch zentao products failed", zap.Error(err))
		render.Error(c, http.StatusBadGateway, "获取禅道产品失败", err)
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", resp.Raw)
}
