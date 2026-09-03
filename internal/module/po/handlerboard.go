// =============================================================================
// 文件: internal/module/po/handlerboard.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作看板 HTTP 请求。
// 依赖: internal/middleware
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// RegisterRoutes 注册看板路由（需求 + 任务 + 各自 items）。。
// Rules §1: 路由必须挂 RequirePerm（拆分 boarddemand 与 boardtask 两个权限便于角色授权）。
func (h *BoardHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/board/demand", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardDemand)
	rg.GET("/board/demand/items", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardDemandItems)
	rg.GET("/board/task", middleware.RequirePerm(perm.PoBoardTaskList), h.BoardTask)
	rg.GET("/board/task/items", middleware.RequirePerm(perm.PoBoardTaskList), h.BoardTaskItems)
}

// BoardHandler 看板 HTTP 处理器（独立 struct 与 service 其它 handler 解耦）。
type BoardHandler struct {
	svc    *Service
	logger *zap.Logger
}

// NewBoardHandler 构造看板 handler（与 service 其它 handler 同模式）。
func NewBoardHandler(svc *Service, logger *zap.Logger) *BoardHandler {
	return &BoardHandler{svc: svc, logger: logger}
}

// BoardDemand 渲染需求看板页。
func (h *BoardHandler) BoardDemand(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_BOARD_DEMAND, gin.H{
		"Title":     "需求看板",
		"PageTitle": "需求看板",
		"BaseUrl":   "/board/demand",
	})
}

// BoardDemandItems 返回需求看板 JSON。
func (h *BoardHandler) BoardDemandItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req BoardDemandReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	resp, err := h.svc.BoardDemand(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po board demand", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取需求看板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"tree":       resp.Tree,
		"summary":    resp.Summary,
		"teamgroups": resp.Teamgroups,
	})
}

// BoardTask 渲染任务看板页。
func (h *BoardHandler) BoardTask(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_BOARD_TASK, gin.H{
		"Title":     "任务看板",
		"PageTitle": "任务看板",
		"BaseUrl":   "/board/task",
	})
}

// BoardTaskItems 返回任务看板 JSON。
func (h *BoardHandler) BoardTaskItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req BoardTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	resp, err := h.svc.BoardTask(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po board task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取任务看板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"columns":             resp.Columns,
		"owners":              resp.Owners,
		"teamgroups":          resp.Teamgroups,
		"selectedTeamgroupId": resp.SelectedTeamgroupID,
		"summary":             resp.Summary,
	})
}
