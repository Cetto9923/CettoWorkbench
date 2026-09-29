// =============================================================================
// 文件: internal/module/schedule/handler_scheduling_save.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需排期保存入口（写库 + 同步禅道）。
//       依赖 service 层对象级写权限闸门，并在参数解析之前先鉴权：
//       无写权限的账号即便提交缺参/非法请求也返回 403，而不是参数校验错误。
// 依赖: internal/middleware
//       internal/pkg/errorx
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
)

// SaveScheduling 保存排期一体化弹窗数据并同步禅道（JSON）。
func (h *Handler) SaveScheduling(c *gin.Context) {
	demandID, ok := parseDemandID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	// 对象级写权限先于参数解析：403 优先于 400/422，避免用参数校验错误
	// 掩盖「无权写该需求」这一真实原因。service.SaveScheduling 内部仍会再校验一次。
	actor := middleware.CurrentUser(c)
	if err := h.svc.RequireDemandWriteAccess(c.Request.Context(), actor, demandID); err != nil {
		h.writeSchedulingAuthZError(c, demandID, err)
		return
	}

	var req SaveSchedulingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": formatFieldErrors(errs),
			"errors":  errs,
		})
		return
	}

	if err := h.svc.SaveScheduling(c.Request.Context(), actor, demandID, &req); err != nil {
		h.writeSchedulingAuthZError(c, demandID, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// writeSchedulingAuthZError 把 service 层的鉴权拒绝映射为 403，其余错误按 500 兜底。
func (h *Handler) writeSchedulingAuthZError(c *gin.Context, demandID uint, err error) {
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": bizErr.Msg,
		})
		return
	}
	if h.logger != nil {
		h.logger.Error("save demand scheduling failed",
			zap.Error(err),
			zap.Uint("demand_id", demandID),
		)
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"message": err.Error(),
	})
}
