// =============================================================================
// 文件: internal/module/po/handler_clarify.go
// 模块: PO 工作台
// 类型: handler
// 职责: 需求澄清 HTTP 控制器。
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

// GetDemandClarify 获取澄清表单初始化数据。
// GET /demands/:id/clarify
func (h *Handler) GetDemandClarify(c *gin.Context) {
	id, err := parseClarifyDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, svcErr := h.svc.GetDemandClarifyForm(c.Request.Context(), actor, id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po get demand clarify", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := clarifyHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ClarifyDemand 提交需求澄清。
// POST /demands/:id/clarify
func (h *Handler) ClarifyDemand(c *gin.Context) {
	id, err := parseClarifyDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req DemandClarifySubmitReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	req.ID = id

	actor := middleware.CurrentUser(c)
	if svcErr := h.svc.ClarifyDemand(c.Request.Context(), actor, req); svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po clarify demand", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := clarifyHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "需求澄清成功",
		"redirectUrl": "/home",
	})
}

// GenerateAIUserStory AI 辅助生成用户故事。
// POST /demands/:id/clarify/ai-generate
func (h *Handler) GenerateAIUserStory(c *gin.Context) {
	id, err := parseClarifyDemandID(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	var req AIUserStoryReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	req.DemandID = uint(id)

	actor := middleware.CurrentUser(c)
	resp, svcErr := h.svc.GenerateAIUserStory(c.Request.Context(), actor, req)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po clarify ai generate", zap.Error(svcErr), zap.Int64("id", id))
		}
		status, msg := clarifyHTTPError(svcErr)
		c.JSON(status, gin.H{"message": msg})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func parseClarifyDemandID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && strings.EqualFold(raw[:2], "US") {
		raw = raw[2:]
	}
	return strconv.ParseInt(raw, 10, 64)
}

func clarifyHTTPError(err error) (int, string) {
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
	return http.StatusInternalServerError, "系统开小差了，请稍后重试"
}
