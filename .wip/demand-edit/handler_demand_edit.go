// =============================================================================
// 文件: internal/module/po/handler_demand_edit.go
// 模块: PO 工作台
// 类型: handler
// 职责: 业务需求草稿/驳回状态编辑与删除 HTTP 控制器（JSON 接口及独立编辑页面）。
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/render"
)

// GetDemandEditData 获取业务需求编辑初始化数据与字典选项。
// GET /demands/:id/edit-data
func (h *Handler) GetDemandEditData(c *gin.Context) {
	id, err := parseReviewDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, svcErr := h.svc.GetDemandEditData(c.Request.Context(), actor, id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Warn("get demand edit data failed", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := demandEditHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateDemand 更新业务需求草稿/驳回内容。
// POST /demands/:id/edit
func (h *Handler) UpdateDemand(c *gin.Context) {
	id, err := parseReviewDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	var req UpdateDemandReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}
	req.ID = id
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  fieldErrs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if svcErr := h.svc.UpdateDemand(c.Request.Context(), actor, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("update demand failed", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := demandEditHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	msg := "需求更新成功"
	if req.SubmitReview {
		msg = "需求保存并已成功提交业务评审"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": msg,
		"id":      id,
	})
}

// DeleteDemand 安全删除业务需求草稿/驳回。
// POST /demands/:id/delete
func (h *Handler) DeleteDemand(c *gin.Context) {
	id, err := parseReviewDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	var req DeleteDemandReq
	_ = c.ShouldBindJSON(&req)
	req.ID = id
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  fieldErrs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if svcErr := h.svc.DeleteDemand(c.Request.Context(), actor, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("delete demand failed", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := demandEditHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "需求已成功删除",
		"id":      id,
	})
}

// DemandEditView 渲染独立全屏业务需求编辑页。
// GET /demands/:id/edit
func (h *Handler) DemandEditView(c *gin.Context) {
	id, err := parseReviewDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, svcErr := h.svc.GetDemandEditData(c.Request.Context(), actor, id)
	if svcErr != nil {
		status, msg := demandEditHTTPError(svcErr)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_DEMAND_EDIT, gin.H{
		"Title":     "编辑业务需求 - " + resp.Demand.Code,
		"PageTitle": "编辑业务需求",
		"Data":      resp,
	})
}

func demandEditHTTPError(err error) (int, string) {
	if biz, ok := errorx.IsBizError(err); ok {
		switch biz.Code {
		case errorx.ErrCodeForbidden:
			return http.StatusForbidden, biz.Msg
		case errorx.ErrCodeNotFound:
			return http.StatusNotFound, biz.Msg
		case errorx.ErrCodeConflict, errorx.ErrCodeInvalidParam:
			return http.StatusUnprocessableEntity, biz.Msg
		}
		return http.StatusBadRequest, biz.Msg
	}
	return http.StatusInternalServerError, "操作失败，请稍后重试"
}
