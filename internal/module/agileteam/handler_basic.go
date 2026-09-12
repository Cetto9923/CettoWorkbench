// =============================================================================
// 文件: internal/module/agileteam/handler_basic.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 基本信息保存接口。
// 依赖: internal/middleware
//       internal/pkg/perm
//
// 2026-08-27 B 决策（AGENTS.md §4.7）- PMO 编辑基本信息绕过组织级教练确认流：
//   - 本接口是 PMO 直接编辑敏捷小组基本信息的入口；
//   - 持 AgileTeamConfirm 权限（PMO）的账号直接生效，**不**走成员调整 confirm 流程；
//   - 详情 / 列表"待确认"UI 应按 history.eventType='updatedByPmo' 直接排除；
//   - 其他模块 / 其他阶段（如 SubmitAdjustment / 提测 / 验收）不得套用本旁路。
// =============================================================================

package agileteam

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
)

func (h *Handler) UpdateBasic(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "小组 ID 无效"})
		return
	}
	var req UpdateBasicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	// hasPerm(AgileTeamConfirm) 仅 PMO / 超管为 true——这是 PMO 旁路的粗粒度入口。
	// 对象级校验（PO/Manager 命中）由 Service.requireTeamgroupObjectEdit 兜底。
	if err := h.svc.UpdateBasicInfo(
		c.Request.Context(),
		middleware.CurrentUser(c),
		req,
		hasPerm(c, perm.AgileTeamConfirm),
	); err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, "基本信息已保存")
}
