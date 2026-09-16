// =============================================================================
// 文件: internal/module/po/handler_deliver.go
// 模块: PO 工作台
// 类型: handler
// 职责: 发起交付 HTTP 控制器（表单加载与提交）。
// =============================================================================

package po

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// GetDemandDeliver 获取发起交付表单初始化元数据。
// GET /demands/:id/deliver
func (h *Handler) GetDemandDeliver(c *gin.Context) {
	id, err := parseDeliverDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, svcErr := h.svc.GetDemandDeliverMeta(c.Request.Context(), actor, id)
	if svcErr != nil {
		if h.logger != nil {
			h.logger.Error("po get demand deliver meta failed", zap.Error(svcErr), zap.Uint("id", id))
		}
		status := http.StatusInternalServerError
		if svcErr == errHomeActionNotFound {
			status = http.StatusNotFound
		} else if svcErr == errHomeActionForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"success": false, "message": svcErr.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeliverDemand 处理发起交付提交。
// POST /demands/:id/deliver
// 兼容原有的简单 JSON 提交 {comment: ...} 与新版的完整表单 JSON 提交。
func (h *Handler) DeliverDemand(c *gin.Context) {
	id, err := parseDeliverDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}

	var req DemandDeliverReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}
	req.ID = id

	var actionErr error
	fullForm := req.DeliverDate != "" || req.VerifyPlan != "" || req.Verifier != ""
	if fullForm {
		if errs := req.Validate(); len(errs) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": "参数校验失败", "errors": errs})
			return
		}
		actionErr = h.svc.DeliverDemand(c.Request.Context(), middleware.CurrentUser(c), req)
	} else {
		actionErr = h.svc.DeliverHomeDemand(c.Request.Context(), middleware.CurrentUser(c), id, req.Comment)
	}
	if actionErr != nil {
		if h.logger != nil {
			h.logger.Error("po deliver demand failed", zap.Error(actionErr), zap.Uint("id", id))
		}
		status := http.StatusInternalServerError
		switch {
		case errors.Is(actionErr, errHomeActionNotFound):
			status = http.StatusNotFound
		case errors.Is(actionErr, errHomeActionForbidden):
			status = http.StatusForbidden
		case errors.Is(actionErr, errHomeActionConflict):
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "message": actionErr.Error()})
		return
	}
	response := gin.H{"success": true, "message": "发起交付成功"}
	if !fullForm {
		response["redirectUrl"] = "/home"
	}
	c.JSON(http.StatusOK, response)
}

func parseDeliverDemandID(raw string) (uint, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "US")
	s = strings.TrimPrefix(s, "us")
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}
