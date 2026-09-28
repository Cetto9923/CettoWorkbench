// =============================================================================
// 文件: internal/module/agileteam/handler_adjustment.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 提交 / 确认 / 驳回调整单。
// 依赖: internal/middleware
// =============================================================================

package agileteam

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
)

func (h *Handler) SubmitAdjustment(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "小组 ID 无效"})
		return
	}
	var req SubmitAdjustmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	req.TeamgroupID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	resp, err := h.svc.SubmitAdjustment(
		c.Request.Context(),
		middleware.CurrentUser(c),
		req,
		hasPerm(c, perm.AgileTeamConfirm),
	)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "成员调整已提交，待组织级敏捷教练确认。新增成员已经可以在任务看板中参与协作。",
		"redirectUrl": agileteamRedirect,
		"data":        resp,
	})
}

func (h *Handler) ConfirmAdjustment(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "调整单 ID 无效"})
		return
	}
	if err := h.svc.ConfirmAdjustment(
		c.Request.Context(),
		middleware.CurrentUser(c),
		ConfirmReq{AdjustmentID: id},
		hasPerm(c, perm.AgileTeamConfirm),
	); err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, "成员调整已确认生效")
}

func (h *Handler) RejectAdjustment(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "调整单 ID 无效"})
		return
	}
	var req RejectReq
	_ = c.ShouldBindJSON(&req)
	req.AdjustmentID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	if err := h.svc.RejectAdjustment(
		c.Request.Context(),
		middleware.CurrentUser(c),
		req,
		hasPerm(c, perm.AgileTeamConfirm),
	); err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, "成员调整已驳回")
}

func (h *Handler) AdjustmentDetail(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "调整单 ID 无效"})
		return
	}
	canConfirm := hasPerm(c, perm.AgileTeamConfirm) && strings.ToLower(c.Query("view")) != "lead"
	resp, err := h.svc.GetAdjustment(c.Request.Context(), middleware.CurrentUser(c), id, canConfirm)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}
