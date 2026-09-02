// =============================================================================
// 文件: internal/module/schedule/handler_story_tasks.go
// 模块: 排期工作台
// 类型: action
// 职责: 处理研发需求任务列表与维护任务弹窗 HTTP 请求。
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

func parseStoryID(c *gin.Context) (uint, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// GetStoryTasks 返回维护任务弹窗与只读任务列表弹窗研发需求详情和任务列表（JSON）。
func (h *Handler) GetStoryTasks(c *gin.Context) {
	storyID, ok := parseStoryID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "研发需求 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.GetStoryTasks(c.Request.Context(), actor, storyID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get story tasks failed",
				zap.Error(err),
				zap.Uint("story_id", storyID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	out := gin.H{
		"success":  true,
		"tasks":    []StoryTaskItem{},
		"projects": []DemandSchedulingProjectOption{},
		"users":    []SchedulingUserOption{},
		"summary":  StoryTaskSummary{},
	}
	if resp != nil {
		out["story"] = resp.Story
		if resp.Tasks != nil {
			out["tasks"] = resp.Tasks
		}
		if resp.Projects != nil {
			out["projects"] = resp.Projects
		}
		if resp.Users != nil {
			out["users"] = resp.Users
		}
		out["summary"] = resp.Summary
		out["defaultProjectId"] = resp.DefaultProjectID
		out["defaultExecutionId"] = resp.DefaultExecutionID
	}
	c.JSON(http.StatusOK, out)
}

// SaveStoryTasks 保存维护任务弹窗任务变更（JSON）。
func (h *Handler) SaveStoryTasks(c *gin.Context) {
	storyID, ok := parseStoryID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	var req SaveStoryTasksReq
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

	actor := middleware.CurrentUser(c)
	if err := h.svc.SaveStoryTasks(c.Request.Context(), actor, storyID, &req); err != nil {
		if h.logger != nil {
			h.logger.Error("save story tasks failed",
				zap.Error(err),
				zap.Uint("story_id", storyID),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
