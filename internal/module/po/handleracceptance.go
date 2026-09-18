// =============================================================================
// 文件: internal/module/po/handleracceptance.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需验收 HTTP 接口（POST JSON，成功返回 redirectUrl）。
// 依赖: internal/middleware
//       internal/pkg/errorx
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
)

// AcceptDemand 处理「业需验收」提交。
//
// 请求：POST /demands/:id/acceptance ，Body 是 JSON。
// 成功：{ success, message, redirectUrl }。
func (h *Handler) AcceptDemand(c *gin.Context) {
	id, err := parseDeliverDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req AcceptanceReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	req.ID = id
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  fieldErrs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if _, svcErr := h.svc.AcceptDemand(c.Request.Context(), actor, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po accept demand", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := acceptanceHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "验收成功",
		"redirectUrl": "/home",
	})
}

func acceptanceHTTPError(err error) (int, string) {
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
	return http.StatusInternalServerError, "验收失败，请稍后重试"
}
