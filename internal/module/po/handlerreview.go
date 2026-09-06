// =============================================================================
// 文件: internal/module/po/handlerreview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需评审 HTTP 接口（POST JSON，成功返回 redirectUrl）。
// 依赖: internal/middleware
//       internal/pkg/errorx
// =============================================================================

package po

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
)

// ReviewDemand 处理「评审需求」提交。
//
// 请求：POST /demands/:id/review ，Body 是 JSON（不是表单）。
// 成功：{ success, message, redirectUrl } —— 前端可刷新列表或跳转首页。
func (h *Handler) ReviewDemand(c *gin.Context) {
	id, err := parseReviewDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req ReviewDemandReq
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
	if _, svcErr := h.svc.ReviewDemand(c.Request.Context(), actor, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po review demand", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := reviewHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "评审成功",
		"redirectUrl": "/home",
	})
}

// parseReviewDemandID 接受数字主键，或列表展示用的 US{id}。
func parseReviewDemandID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && strings.EqualFold(raw[:2], "US") {
		raw = raw[2:]
	}
	return strconv.ParseInt(raw, 10, 64)
}

func reviewHTTPError(err error) (int, string) {
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
	return http.StatusInternalServerError, "评审失败，请稍后重试"
}
