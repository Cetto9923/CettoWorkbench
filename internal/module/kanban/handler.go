// =============================================================================
// 文件: internal/module/kanban/handler.go
// 模块: 工作看板
// 类型: action
// 职责: 需求/任务看板静态页、业需/任务/问题 JSON，及任务状态拖拽更新。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/perm
//       internal/pkg/render
//       internal/pkg/zentao
// =============================================================================

package kanban

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
	"workbench/internal/pkg/zentao"
)

// Handler 处理工作看板页面请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建看板模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册工作看板路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/kanban")
	g.Use(middleware.ActiveNav("/kanban/story"))
	g.GET("/story", middleware.RequirePerm(perm.KanbanStory), h.Story)
	g.GET("/story/demands", middleware.RequirePerm(perm.KanbanStory), h.Demands)
	g.GET("/task", middleware.RequirePerm(perm.KanbanStory), h.Task)
	g.GET("/task/items", middleware.RequirePerm(perm.KanbanStory), h.Tasks)
	g.GET("/issues", middleware.RequirePerm(perm.KanbanStory), h.Issues)

	g.PUT("/tasks/:id", middleware.RequirePerm(perm.KanbanStory), h.UpdateTaskStatus)
}

// Story 渲染需求看板静态页。
func (h *Handler) Story(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	teamgroups, err := h.svc.ListMyTeamgroups(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("list kanban teamgroups failed", zap.Error(err))
		}
		teamgroups = []TeamgroupItem{}
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_KANBAN_STORY, gin.H{
		"Title":          "工作看板",
		"PageTitle":      "工作看板",
		"BaseUrl":        "/kanban/story",
		"IssueCreateURL": zentao.URL("issue", "create"),
		"Teamgroups":     teamgroups,
	})
}

// Task 渲染任务看板静态页。
func (h *Handler) Task(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	teamgroups, err := h.svc.ListMyTeamgroups(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("list kanban teamgroups failed", zap.Error(err))
		}
		teamgroups = []TeamgroupItem{}
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_KANBAN_TASK, gin.H{
		"Title":          "工作看板",
		"PageTitle":      "工作看板",
		"BaseUrl":        "/kanban/task",
		"IssueCreateURL": zentao.URL("issue", "create"),
		"TaskCreateURL":  zentao.URL("task", "create", "executionID=0&storyID=0&moduleID=0"),
		"Teamgroups":     teamgroups,
	})
}

// Demands 按选中负责人返回价值流业需与研需列表（JSON）。
func (h *Handler) Demands(c *gin.Context) {
	var req ListDemandsReq
	_ = c.ShouldBindQuery(&req)

	resp, err := h.svc.ListValueStreamBizDemands(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if biz, ok := errorx.IsBizError(err); ok {
			status := http.StatusBadRequest
			switch biz.Code {
			case errorx.ErrCodeForbidden:
				status = http.StatusForbidden
			case errorx.ErrCodeInvalidParam:
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"success": false, "message": biz.Msg})
			return
		}
		if h.logger != nil {
			h.logger.Error("list kanban value stream demands failed", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取需求失败",
		})
		return
	}
	if resp.Items == nil {
		resp.Items = []BizDemandItem{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"items":        resp.Items,
		"summary":      resp.Summary,
		"memberCounts": resp.MemberCounts,
	})
}

// Tasks 按选中负责人返回任务看板三列（JSON）。
func (h *Handler) Tasks(c *gin.Context) {
	var req ListTasksReq
	_ = c.ShouldBindQuery(&req)

	resp, err := h.svc.ListTasks(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if biz, ok := errorx.IsBizError(err); ok {
			status := http.StatusBadRequest
			switch biz.Code {
			case errorx.ErrCodeForbidden:
				status = http.StatusForbidden
			case errorx.ErrCodeInvalidParam:
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"success": false, "message": biz.Msg})
			return
		}
		if h.logger != nil {
			h.logger.Error("list kanban tasks failed", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取任务失败",
		})
		return
	}
	if resp.Columns == nil {
		resp.Columns = newEmptyTaskColumns()
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"columns": resp.Columns,
		"summary": resp.Summary,
	})
}

// Issues 按选中提出人与问题 tab 返回问题列表（JSON）。
func (h *Handler) Issues(c *gin.Context) {
	var req ListIssuesReq
	_ = c.ShouldBindQuery(&req)
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}

	resp, err := h.svc.ListIssues(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if biz, ok := errorx.IsBizError(err); ok {
			status := http.StatusBadRequest
			switch biz.Code {
			case errorx.ErrCodeForbidden:
				status = http.StatusForbidden
			case errorx.ErrCodeInvalidParam:
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"success": false, "message": biz.Msg})
			return
		}
		if h.logger != nil {
			h.logger.Error("list kanban issues failed", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取问题失败",
		})
		return
	}
	if resp.Items == nil {
		resp.Items = []IssueItem{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   resp.Items,
		"counts":  resp.Counts,
	})
}

// UpdateTaskStatus 拖拽更新任务状态（wait↔doing），成功返回 JSON。
func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "任务 ID 无效"})
		return
	}

	var req UpdateTaskStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}

	if err := h.svc.UpdateTaskStatus(c.Request.Context(), middleware.CurrentUser(c), req); err != nil {
		if biz, ok := errorx.IsBizError(err); ok {
			status := http.StatusBadRequest
			switch biz.Code {
			case errorx.ErrCodeForbidden:
				status = http.StatusForbidden
			case errorx.ErrCodeNotFound:
				status = http.StatusNotFound
			case errorx.ErrCodeInvalidParam:
				status = http.StatusBadRequest
			case errorx.ErrCodeInternal:
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"success": false, "message": biz.Msg})
			return
		}
		if h.logger != nil {
			h.logger.Error("update kanban task status failed", zap.Error(err), zap.Int64("taskId", id))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新任务状态失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "任务状态已更新",
		"redirectUrl": "/kanban/task",
	})
}
