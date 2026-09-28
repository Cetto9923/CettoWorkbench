// =============================================================================
// 文件: internal/module/po/handler_done.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办页面与数据接口（/done, /done/items, /done/meta, /done/detail/:actionId）。
// =============================================================================

package po

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/render"
)

// Done 渲染"我的已办"页面。
func (h *Handler) Done(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_DONE, gin.H{
		"Title":           "我的已办",
		"PageTitle":       "我的已办",
		"PageDescription": "查看由我推进、确认、流转或结项的历史事项及动作记录",
		"BaseUrl":         "/done",
	})
}

// DoneItems 返回"我的已办"列表 JSON。
func (h *Handler) DoneItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req DoneListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	resp, err := h.svc.DoneList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po done list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取已办列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"data":     resp,
		"items":    resp.Items,
		"total":    resp.Total,
		"summary":  resp.Summary,
		"facets":   resp.Facets,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

// DoneMeta 返回"我的已办"筛选项元数据 JSON。
func (h *Handler) DoneMeta(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.DoneMeta(c.Request.Context(), actor, c.Query("objectType"))
	if err != nil {
		h.logger.Error("po done meta", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取已办筛选项失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// DoneDetail 返回单个已办动作详情及邻近时间线。
// F01：DoneDetail 在 Handler 出口区分 401/403/404/500。
func (h *Handler) DoneDetail(c *gin.Context) {
	actionIdStr := c.Param("actionId")
	actionId, _ := strconv.ParseInt(actionIdStr, 10, 64)
	if actionId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的动作ID"})
		return
	}
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.DoneDetail(c.Request.Context(), actor, actionId)
	if err != nil {
		if bizErr, ok := errorx.IsBizError(err); ok {
			switch bizErr.Code {
			case errorx.ErrCodeInvalidParam:
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false, "message": bizErr.Msg,
				})
				return
			case errorx.ErrCodeNotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"success": false, "message": "未找到相关已办记录",
				})
				return
			case errorx.ErrCodeForbidden:
				c.JSON(http.StatusForbidden, gin.H{
					"success": false, "message": "无权查看该已办动作",
				})
				return
			}
		}
		h.logger.Error("po done detail", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false, "message": "已办详情查询失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}
