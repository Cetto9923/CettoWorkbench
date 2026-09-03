// =============================================================================
// 文件: internal/module/po/handler.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 HTTP 请求。
// 依赖: internal/middleware
//       internal/pkg/render
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

// Handler 处理 PO 工作台页面请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 PO 模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册 PO 工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.Use(middleware.ActiveNav("/home"))

	g.GET("/home", middleware.RequirePerm(perm.PoHome), h.Home)
	g.GET("/demands", middleware.RequirePerm(perm.PoHome), h.Demands)

	// 我的待办 / 我的已办（M3 阶段）
	g.GET("/todos", middleware.RequirePerm(perm.PoTodo), h.Todos)
	g.GET("/todos/items", middleware.RequirePerm(perm.PoTodo), h.TodosItems)
	g.GET("/done", middleware.RequirePerm(perm.PoDone), h.Done)
	g.GET("/done/items", middleware.RequirePerm(perm.PoDone), h.DoneItems)
}

// Home 渲染 PO 工作台首页。
func (h *Handler) Home(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}

	resp, err := h.svc.Home(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("po home value stream", zap.Error(err))
		}
		resp = &HomeResp{Stages: emptyValueStreamStages()}
	}

	if h.logger != nil {
		h.logger.Info("po home version windows render",
			zap.String("account", account),
			zap.Int("render_count", len(resp.VersionWindows)),
		)
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":             "工作台首页",
		"PageTitle":         "工作台首页",
		"ValueStreamStages": resp.Stages,
		"VersionWindows":    resp.VersionWindows,
		"KPI":               resp.KPI,
	})
}

// Demands 按价值流状态返回当前用户的需求/故事详情（JSON）。
func (h *Handler) Demands(c *gin.Context) {
	var req DemandsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数解析失败",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}

	resp, err := h.svc.Demands(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("po demand details", zap.Error(err), zap.String("status", req.Status))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取需求详情失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   resp.Items,
	})
}

func emptyValueStreamStages() []ValueStreamStage {
	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	for _, def := range valueStreamStages {
		stages = append(stages, ValueStreamStage{
			Label:  def.label,
			Status: def.status,
		})
	}
	return stages
}

// Todos 渲染"我的待办"页面。
func (h *Handler) Todos(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_TODOS, gin.H{
		"Title":     "我的待办",
		"PageTitle": "我的待办",
		"CurrentUser": actor,
	})
}

// TodosItems 返回"我的待办"列表 JSON。
func (h *Handler) TodosItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req TodoListReq
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
	resp, err := h.svc.TodoList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po todo list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取待办列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   resp.Items,
		"total":   resp.Total,
		"page":    resp.Page,
		"pageSize": resp.PageSize,
	})
}

// Done 渲染"我的已办"页面。
func (h *Handler) Done(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_DONE, gin.H{
		"Title":     "我的已办",
		"PageTitle": "我的已办",
		"CurrentUser": actor,
	})
}

// DoneItems 返回"我的已办"列表 JSON。
func (h *Handler) DoneItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req DoneListReq
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
	resp, err := h.svc.DoneList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po done list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取已办列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   resp.Items,
		"total":   resp.Total,
		"page":    resp.Page,
		"pageSize": resp.PageSize,
	})
}
