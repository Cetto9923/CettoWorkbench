// =============================================================================
// 文件: internal/module/cronjob/handler.go
// 模块: 定时任务
// 类型: action
// 职责: 处理任务列表、启停、手动触发与执行日志页面请求。
// 依赖: internal/middleware
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package cronjob

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 定时任务模块 Handler。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/cron-jobs")
	g.Use(middleware.ActiveNav("/admin/cron-jobs"))
	{
		g.GET("", middleware.RequirePerm(perm.CronJobList), h.List)
		g.POST("/:id/toggle", middleware.RequirePerm(perm.CronJobEdit), h.Toggle)
		g.POST("/:id/trigger", middleware.RequirePerm(perm.CronJobTrigger), h.Trigger)
		g.GET("/logs", middleware.RequirePerm(perm.CronJobList), h.Logs)
	}
}

// List 定时任务列表。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	var req ListReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("list cron jobs failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取定时任务失败", err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_CRONJOB_LIST, gin.H{
		"Title":     "定时任务",
		"PageTitle": "定时任务",
		"Jobs":      resp.Items,
		"Pager":     resp.Pager,
	})
}

// Toggle 启用/禁用任务。
func (h *Handler) Toggle(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "任务 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
		return
	}
	var req ToggleReq
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "参数解析失败")
		c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
		return
	}
	req.ID = id
	actor := middleware.CurrentUser(c)
	if err := h.svc.Toggle(c.Request.Context(), actor, req); err != nil {
		if IsNotFound(err) {
			flash.Error(c, "任务不存在")
		} else {
			flash.Error(c, "更新任务状态失败: "+err.Error())
		}
		c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
		return
	}
	if req.IsEnabled {
		flash.Success(c, "任务已启用")
	} else {
		flash.Success(c, "任务已禁用")
	}
	c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
}

// Trigger 手动触发任务。
func (h *Handler) Trigger(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "任务 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
		return
	}
	req := TriggerReq{ID: id}
	actor := middleware.CurrentUser(c)
	if err := h.svc.Trigger(c.Request.Context(), actor, req); err != nil {
		if IsNotFound(err) {
			flash.Error(c, "任务不存在")
		} else {
			flash.Error(c, "触发任务失败: "+err.Error())
		}
		c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
		return
	}
	flash.Success(c, "任务触发成功")
	c.Redirect(http.StatusSeeOther, "/admin/cron-jobs")
}

// Logs 执行日志列表。
func (h *Handler) Logs(c *gin.Context) {
	h.bindRenderer(c)
	var req LogListReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.JobName = strings.TrimSpace(req.JobName)
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.ListLogs(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("list cron logs failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取执行日志失败", err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_CRONJOB_LOGS, gin.H{
		"Title":     "任务执行日志",
		"PageTitle": "任务执行日志",
		"Form":      &req,
		"Logs":      resp.Items,
		"Pager":     resp.Pager,
		"JobName":   req.JobName,
	})
}

func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}

func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}
