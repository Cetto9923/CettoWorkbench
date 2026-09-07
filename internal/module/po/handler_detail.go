// =============================================================================
// 文件: internal/module/po/handler_detail.go
// 模块: PO 工作台
// 类型: action
// 职责: 业务需求统一详情（JSON API + 独立查看页面）。
//       F01：DemandDetail 在 Handler 出口区分 401/403/404/500，避免
//       「DB 故障 → 404」与「无权 → 404」两种语义被合并。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/render
// =============================================================================

package po

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
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
		// F01：先识别业务错误码，再向上抛其他错误；防止 DB 故障被吞为 404。
		// InvalidParam 需区分「缺登录身份」(401) 与「入参无效」(422)，避免误报 401。
		if bizErr, ok := errorx.IsBizError(err); ok {
			switch bizErr.Code {
			case errorx.ErrCodeInvalidParam:
				if actor == nil || strings.TrimSpace(actor.Account) == "" {
					c.JSON(http.StatusUnauthorized, gin.H{
						"success": false,
						"message": "未登录或会话已过期",
					})
					return
				}
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"success": false,
					"message": bizErr.Msg,
				})
				return
			case errorx.ErrCodeNotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "需求不存在",
				})
				return
			case errorx.ErrCodeForbidden:
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "无权查看该业务需求",
				})
				return
			}
		}
		if h.logger != nil {
			h.logger.Error("demand detail failed", zap.Uint("demandId", demandID), zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "需求详情查询失败",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DemandDetailView 将旧详情直链收敛到首页抽屉，避免独立页面与抽屉重复渲染。
func (h *Handler) DemandDetailView(c *gin.Context) {
	var req DemandDetailReq
	if err := c.ShouldBindUri(&req); err != nil || req.ID == "" {
		_ = c.ShouldBindQuery(&req)
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.Status(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusFound, "/home?openDemand="+url.QueryEscape(req.ID))
}
