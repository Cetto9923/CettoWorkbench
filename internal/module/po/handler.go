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
	"context"
	"errors"
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
	g.POST("/demands/:id/review", middleware.RequirePerm(perm.PoHomeList), h.ReviewDemand)
	g.POST("/demands/:id/withdraw-review", middleware.RequirePerm(perm.PoHomeList), h.WithdrawDemandReview)
	g.POST("/demands/:id/submit-review", middleware.RequirePerm(perm.PoHomeList), h.SubmitDemandReview)
	g.GET("/demands/:id/clarify", middleware.RequirePerm(perm.PoHomeList), h.GetDemandClarify)
	g.POST("/demands/:id/clarify", middleware.RequirePerm(perm.PoHomeList), h.ClarifyDemand)
	g.POST("/demands/:id/clarify/ai-generate", middleware.RequirePerm(perm.PoHomeList), h.GenerateAIUserStory)
	g.POST("/demands/:id/acceptance", middleware.RequirePerm(perm.PoHomeList), h.AcceptHomeDemand)
	g.POST("/demands/:id/urge", middleware.RequirePerm(perm.PoHomeList), h.UrgeHomeDemand)
	g.GET("/demands/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.GetDemandDeliver)
	g.POST("/demands/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.DeliverDemand)
	// 详情读接口：首页与需求看板均可打开；对象级授权仍由 DetailService 执行。
	g.GET("/demands/:id/detail", middleware.RequireAnyPerm(perm.PoHomeList, perm.PoBoardDemandList), h.DemandDetail)
	g.GET("/demands/:id/submit-test", middleware.RequirePerm(perm.PoHomeList), h.SubmitTestView)
	g.GET("/demands/:id", middleware.RequireAnyPerm(perm.PoHomeList, perm.PoBoardDemandList), h.DemandDetailView)
	g.GET("/todos", middleware.RequirePerm(perm.PoTodoList), h.Todos)
	g.GET("/todos/items", middleware.RequirePerm(perm.PoTodoList), h.TodosItems)
	g.GET("/done", middleware.RequirePerm(perm.PoDoneList), h.Done)
	g.GET("/done/items", middleware.RequirePerm(perm.PoDoneList), h.DoneItems)
	g.GET("/done/meta", middleware.RequirePerm(perm.PoDoneList), h.DoneMeta)
	g.GET("/done/detail/:actionId", middleware.RequirePerm(perm.PoDoneList), h.DoneDetail)

	// 兼容老版本 API 路径
	wbApi := rg.Group("/workbench/api")
	wbApi.GET("/done", middleware.RequirePerm(perm.PoDoneList), h.DoneItems)
	wbApi.GET("/done/meta", middleware.RequirePerm(perm.PoDoneList), h.DoneMeta)
	wbApi.GET("/done/detail/:actionId", middleware.RequirePerm(perm.PoDoneList), h.DoneDetail)
	wbApi.GET("/watches/project-weeklies", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklies)
	wbApi.GET("/watches/project-weeklies/teams", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyTeams)
	wbApi.GET("/watches/project-weeklies/:id", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyDetail)
	wbApi.GET("/watches/project-weeklies/:id/history", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyHistory)
	g.GET("/notice", middleware.RequirePerm(perm.PoNoticeList), h.Notice)
	g.GET("/notice/items", middleware.RequirePerm(perm.PoNoticeList), h.NoticeItems)
	g.GET("/follow", middleware.RequirePerm(perm.PoFollowList), h.Follow)
	g.GET("/follow/items", middleware.RequirePerm(perm.PoFollowList), h.FollowItems)
	g.GET("/follow/project-weeklies", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklies)
	g.GET("/follow/project-weeklies/teams", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyTeams)
	g.GET("/follow/project-weeklies/:id", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyDetail)
	g.GET("/follow/project-weeklies/:id/history", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyHistory)
	// Keep the sidebar's public path aligned with the page capability name.
	// The singular path remains as a compatibility alias for existing links.
	for _, path := range []string{"/issues/risk", "/issue-risk"} {
		g.GET(path, middleware.RequirePerm(perm.PoBoardDemandList), h.IssueRisk)
		g.GET(path+"/items", middleware.RequirePerm(perm.PoBoardDemandList), h.IssueRiskItems)
	}

	g.PUT("/notice/:id/read", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkRead)
	g.PUT("/notice/read-all", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkAllRead)
	g.PUT("/follow/demand/:id", middleware.RequirePerm(perm.PoFollowUpdate), h.FollowSetDemand)
	g.PUT("/follow/project-report/:id", middleware.RequirePerm(perm.PoFollowUpdate), h.FollowRemoveProjectReport)

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
		"AllCount":            resp.AllCount,
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
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// 客户端取消或服务端超时，不写 error 日志、不返回 5xx。
			c.Status(499)
			return
		}
		if h.logger != nil {
			h.logger.Error("po demand details", zap.Error(err), zap.String("status", req.Status))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取需求详情失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"items":        resp.Items,
		"total":        resp.Total,
		"page":         resp.Page,
		"pageSize":     resp.PageSize,
		"stageSummary": resp.StageSummary,
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
		"facets":   resp.Facets,
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
		"data":     resp,
		"items":    resp.Items,
		"total":    resp.Total,
		"summary":  resp.Summary,
		"facets":   resp.Facets,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

// DoneMeta 返回"我的已办"筛选项元数据 JSON。
func (h *Handler) DoneMeta(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.DoneMeta(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("po done meta", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取已办筛选项失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// DoneDetail 返回单个已办动作详情及邻近时间线。
//
// F01：DoneDetail 在 Handler 出口区分 401/403/404/500，与 demand detail
// 统一语义；不再把 DB 故障或权限拒绝都映射为 404。
func (h *Handler) DoneDetail(c *gin.Context) {
	actionIdStr := c.Param("actionId")
	actionId, _ := strconv.ParseInt(actionIdStr, 10, 64)
	if actionId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的动作ID"})
		return
	}
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.DoneDetail(c.Request.Context(), actor, actionId)
	if err != nil {
		if bizErr, ok := errorx.IsBizError(err); ok {
			switch bizErr.Code {
			case errorx.ErrCodeInvalidParam:
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false, "message": bizErr.Msg,
				})
				return
			case errorx.ErrCodeNotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"success": false, "message": "未找到相关已办记录",
				})
				return
			case errorx.ErrCodeForbidden:
				c.JSON(http.StatusForbidden, gin.H{
					"success": false, "message": "无权查看该已办动作",
				})
				return
			}
		}
		h.logger.Error("po done detail", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false, "message": "已办详情查询失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
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
	var body struct {
		Filters *NoticeListReq `json:"filters"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Filters == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "必须提供当前筛选条件"})
		return
	}
	req := *body.Filters
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "筛选条件无效", "errors": errs})
		return
	}
	n, err := h.svc.NoticeMarkAllRead(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po notice mark all read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "全部标记已读失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "全部已读", "redirectUrl": "/notice", "affected": n})
}
