// =============================================================================
// 文件: internal/module/schedule/handleriteration.go
// 模块: 排期工作台
// 类型: action
// 职责: 快速创建迭代 HTTP 请求。
// 依赖: internal/middleware
// =============================================================================
package schedule

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"workbench/internal/middleware"
)

func (h *Handler) CreateIteration(c *gin.Context) {
	storyID, ok := parseStoryID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "研发需求 ID 无效"})
		return
	}
	var req CreateIterationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	req.StoryID = storyID
	id, err := h.svc.CreateIteration(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		h.writeWriteAuthZError(c, "create iteration failed", storyID, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "迭代已创建", "executionId": id})
}
