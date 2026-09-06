// =============================================================================
// 文件: internal/module/po/handler_detail.go
// 模块: PO 工作台
// 类型: action
// 职责: 业务需求统一详情（JSON API + 独立查看页面）。
// 依赖: internal/middleware
//       internal/pkg/render
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// DemandDetail 处理 GET /demands/:id/detail，返回去重优化后的业务需求完整聚合数据。
func (h *Handler) DemandDetail(c *gin.Context) {
	var req DemandDetailReq
	if err := c.ShouldBindUri(&req); err != nil || req.ID == "" {
		_ = c.ShouldBindQuery(&req)
	}

	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}

	demandID := req.ExtractDemandID()
	actor := middleware.CurrentUser(c)

	if h.detailSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "详情服务未初始化",
		})
		return
	}

	resp, err := h.detailSvc.GetDemandDetail(c.Request.Context(), actor, demandID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("demand detail failed", zap.Uint("demandId", demandID), zap.Error(err))
		}
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "需求不存在或无权查看",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DemandDetailView 处理 GET /demands/:id 独立页面查看请求。
func (h *Handler) DemandDetailView(c *gin.Context) {
	var req DemandDetailReq
	if err := c.ShouldBindUri(&req); err != nil || req.ID == "" {
		_ = c.ShouldBindQuery(&req)
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_DEMAND_DETAIL, gin.H{
		"Title":     "业务需求详情",
		"PageTitle": "业务需求详情",
		"DemandID":  req.ID,
	})
}
