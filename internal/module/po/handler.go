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
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理 PO 工作台页面请求。
type Handler struct {
	svc       *Service
	detailSvc *DetailService
	logger    *zap.Logger
}

// NewHandler 创建 PO 模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	var detailSvc *DetailService
	if svc != nil {
		detailSvc = svc.DetailService()
	}
	return &Handler{svc: svc, detailSvc: detailSvc, logger: logger}
}

// RegisterRoutes 注册 PO 工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.Use(middleware.ActiveNav("/home"))

	g.GET("/home", middleware.RequirePerm(perm.PoHomeList), h.Home)
	g.GET("/demands", middleware.RequirePerm(perm.PoHomeList), h.Demands)
	g.GET("/demands/:id/detail", middleware.RequirePerm(perm.PoHomeList), h.DemandDetail)
	g.GET("/demands/:id", middleware.RequirePerm(perm.PoHomeList), h.DemandDetailView)
	g.GET("/todos", middleware.RequirePerm(perm.PoTodoList), h.Todos)
	g.GET("/todos/items", middleware.RequirePerm(perm.PoTodoList), h.TodosItems)
	g.GET("/done", middleware.RequirePerm(perm.PoDoneList), h.Done)
	g.GET("/done/items", middleware.RequirePerm(perm.PoDoneList), h.DoneItems)
	g.GET("/notice", middleware.RequirePerm(perm.PoNoticeList), h.Notice)
	g.GET("/notice/items", middleware.RequirePerm(perm.PoNoticeList), h.NoticeItems)
	g.GET("/follow", middleware.RequirePerm(perm.PoFollowList), h.Follow)
	g.GET("/follow/items", middleware.RequirePerm(perm.PoFollowList), h.FollowItems)

	g.PUT("/notice/:id/read", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkRead)
	g.PUT("/notice/read-all", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkAllRead)
	g.PUT("/follow/demand/:id", middleware.RequirePerm(perm.PoFollowUpdate), h.FollowSetDemand)

	// PO 工作看板（V1.3 需求+任务双视图）
	NewBoardHandler(h.svc, h.logger).RegisterRoutes(g)
}

// Home 渲染 PO 工作台首页。
func (h *Handler) Home(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}

	pageError := ""
	versionWindowsError := ""
	resp, err := h.svc.Home(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("po home value stream", zap.Error(err))
		}
		pageError = "数据统计暂不可用"
		versionWindowsError = "版本窗口查询失败"
		resp = &HomeResp{
			Stages:              emptyValueStreamStages(),
			StagesValid:         false,
			StagesError:         pageError,
			VersionWindowsError: versionWindowsError,
		}
	} else {
		versionWindowsError = resp.VersionWindowsError
	}

	if h.logger != nil {
		h.logger.Info("po home version windows render",
			zap.String("account", account),
			zap.Int("render_count", len(resp.VersionWindows)),
		)
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":               "工作台首页",
		"PageTitle":           "工作台首页",
		"ValueStreamStages":   resp.Stages,
		"StagesValid":         resp.StagesValid,
		"VersionWindows":      resp.VersionWindows,
		"VersionWindowsError": versionWindowsError,
		"KPI":                 resp.KPI,
		"PageError":           pageError,
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
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

func emptyValueStreamStages() []ValueStreamStage {
	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	for _, def := range valueStreamStages {
		stages = append(stages, ValueStreamStage{
			Label:  def.label,
			Status: def.status,
			Valid:  false, // 错误状态下 Valid 显式为 false (ERROR ≠ ZERO)
		})
	}
	return stages
}

// Todos 渲染"我的待办"页面。
func (h *Handler) Todos(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_TODOS, gin.H{
		"Title":     "我的待办",
		"PageTitle": "我的待办",
		"BaseUrl":   "/todos",
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
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
		"summary":  resp.Summary,
		"groups":   resp.Groups,
	})
}

// Done 渲染"我的已办"页面。
func (h *Handler) Done(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_DONE, gin.H{
		"Title":     "我的已办",
		"PageTitle": "我的已办",
		"BaseUrl":   "/done",
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
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"summary":  resp.Summary,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

// Notice 渲染"通知中心"页面。
func (h *Handler) Notice(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_NOTICE, gin.H{
		"Title":     "通知中心",
		"PageTitle": "通知中心",
		"BaseUrl":   "/notice",
	})
}

// NoticeItems 返回"通知中心"列表 + quick view 计数 JSON。
func (h *Handler) NoticeItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req NoticeListReq
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
	resp, err := h.svc.NoticeList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po notice list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取通知列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"items":         resp.Items,
		"total":         resp.Total,
		"filteredTotal": resp.Filtered,
		"unread":        resp.Unread,
		"action":        resp.Action,
		"abnormal":      resp.Abnormal,
		"today":         resp.Today,
		"categories":    resp.Categories,
		"page":          resp.Page,
		"pageSize":      resp.PageSize,
	})
}

// NoticeMarkRead 标记单条通知已读。
func (h *Handler) NoticeMarkRead(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的通知 ID"})
		return
	}
	if err := h.svc.NoticeMarkRead(c.Request.Context(), actor, id); err != nil {
		if bizErr, ok := errorx.IsBizError(err); ok {
			switch bizErr.Code {
			case errorx.ErrCodeNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": bizErr.Msg})
				return
			case errorx.ErrCodeForbidden:
				c.JSON(http.StatusForbidden, gin.H{"message": bizErr.Msg})
				return
			}
		}
		h.logger.Error("po notice mark read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "标记已读失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已标记已读", "redirectUrl": "/notice"})
}

// NoticeMarkAllRead 全部标记已读。
func (h *Handler) NoticeMarkAllRead(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	n, err := h.svc.NoticeMarkAllRead(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("po notice mark all read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "全部标记已读失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "全部已读", "redirectUrl": "/notice", "affected": n})
}

// Follow 渲染"我的关注"页面。
func (h *Handler) Follow(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_FOLLOW, gin.H{
		"Title":     "我的关注",
		"PageTitle": "我的关注",
		"BaseUrl":   "/follow",
	})
}

// FollowItems 返回"我的关注"列表 JSON。
func (h *Handler) FollowItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req FollowListReq
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
	resp, err := h.svc.FollowList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po follow list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取关注列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

// FollowSetDemand 切换对业务需求的关注状态。
func (h *Handler) FollowSetDemand(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的业务需求 ID"})
		return
	}
	req := FollowSetReq{ID: id}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": errs})
		return
	}
	if err := h.svc.FollowSetDemand(c.Request.Context(), actor, req); err != nil {
		h.logger.Error("po follow set", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "更新关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已更新关注", "redirectUrl": "/follow"})
}
