// =============================================================================
// 文件: internal/module/schedule/handler_execution.go
// 模块: 排期工作台
// 类型: action
// 职责: 快速创建迭代 HTTP 处理。
// 依赖: internal/middleware
// =============================================================================

package schedule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

func parseCreateExecutionProjectID(c *gin.Context) (uint, bool) {
	idStr := strings.TrimSpace(c.Param("projectId"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// CreateExecution 创建项目下的执行（迭代）并返回新建执行（JSON）。
func (h *Handler) CreateExecution(c *gin.Context) {
	projectID, ok := parseCreateExecutionProjectID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "项目 ID 无效",
		})
		return
	}

	var req CreateExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	req.ProjectID = projectID
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": formatFieldErrors(errs),
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.CreateExecution(c.Request.Context(), actor, req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("create execution failed",
				zap.Error(err),
				zap.Uint("project_id", projectID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "创建成功",
		"execution": resp,
	})
}
