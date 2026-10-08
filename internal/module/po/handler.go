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

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理 PO 工作台页面请求。
type Handler struct {
	svc               *Service
	detailSvc         *DetailService
	logger            *zap.Logger
	teamViewAccess    func(context.Context, *model.User) (bool, error)
	teamScopeAccounts func(context.Context, *model.User, string, uint) ([]string, error)
	teamScopeGroupIDs func(context.Context, *model.User, string, uint) ([]uint, error)
}

// NewHandler 创建 PO 模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	var detailSvc *DetailService
	if svc != nil {
		detailSvc = svc.DetailService()
	}
	return &Handler{svc: svc, detailSvc: detailSvc, logger: logger}
}

// SetTeamViewAccess injects the independent, read-only authorization check for team managers/coaches.
func (h *Handler) SetTeamViewAccess(check func(context.Context, *model.User) (bool, error)) {
	h.teamViewAccess = check
}

// SetTeamScopeAccounts injects the object-scoped membership resolver used by team read views.
func (h *Handler) SetTeamScopeAccounts(resolve func(context.Context, *model.User, string, uint) ([]string, error)) {
	h.teamScopeAccounts = resolve
}

// SetTeamScopeGroupIDs injects the authorized agile-group resolver for team dashboard data.
func (h *Handler) SetTeamScopeGroupIDs(resolve func(context.Context, *model.User, string, uint) ([]uint, error)) {
	h.teamScopeGroupIDs = resolve
}

// RegisterRoutes 注册 PO 工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.Use(middleware.ActiveNav("/home"))

	g.GET("/home", h.homePageAccess(), h.Home)
	g.GET("/navigation/badges", middleware.RequireAnyPerm(perm.PoHomeList, perm.PoTodoList, perm.PoNoticeList), h.NavigationBadges)
	g.GET("/home/team/version-windows", middleware.RequirePerm(perm.KanbanStory), h.TeamHomeVersionWindows)
	g.GET("/home/team/value-stream", middleware.RequirePerm(perm.KanbanStory), h.TeamHomeValueStream)
	g.GET("/home/team/dashboard", middleware.RequirePerm(perm.KanbanStory), h.TeamDashboard)
	g.GET("/demands", middleware.RequirePerm(perm.PoHomeList), h.Demands)
	g.POST("/demands/:id/review", middleware.RequirePerm(perm.PoHomeList), h.ReviewDemand)
	g.POST("/demands/:id/withdraw-review", middleware.RequirePerm(perm.PoHomeList), h.WithdrawDemandReview)
	g.POST("/demands/:id/submit-review", middleware.RequirePerm(perm.PoHomeList), h.SubmitDemandReview)
	g.GET("/demands/:id/review-candidates", middleware.RequirePerm(perm.PoHomeList), h.DemandReviewCandidates)
	g.GET("/demands/:id/clarify", middleware.RequirePerm(perm.PoHomeList), h.GetDemandClarify)
	g.POST("/demands/:id/clarify", middleware.RequirePerm(perm.PoHomeList), h.ClarifyDemand)
	g.POST("/demands/:id/clarify/ai-generate", middleware.RequirePerm(perm.PoHomeList), h.GenerateAIUserStory)
	g.POST("/demands/:id/acceptance", middleware.RequirePerm(perm.PoHomeList), h.AcceptHomeDemand)
	// 催办暂复用首页可见权限；服务层再校验需求阶段与参与关系。
	g.GET("/demands/:id/urge-preview", middleware.RequirePerm(perm.PoHomeList), h.UrgeHomeDemandPreview)
	g.POST("/demands/:id/urge", middleware.RequirePerm(perm.PoHomeList), h.UrgeHomeDemand)
	g.GET("/demands/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.GetDemandDeliver)
	g.POST("/demands/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.DeliverDemand)
	g.GET("/stories/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.GetStoryDeliver)
	g.POST("/stories/:id/deliver", middleware.RequirePerm(perm.PoHomeList), h.DeliverStory)
	// 详情读接口：首页、需求看板与需求查询均可打开；对象级授权仍由 DetailService 执行。
	g.GET("/demands/:id/detail", middleware.RequireAnyPerm(perm.PoHomeList, perm.PoBoardDemandList, perm.ScheduleList), h.DemandDetail)
	g.GET("/demands/:id", middleware.RequireAnyPerm(perm.PoHomeList, perm.PoBoardDemandList, perm.ScheduleList), h.DemandDetailView)
	g.GET("/demands/:id/primary-action", middleware.RequirePerm(perm.PoHomeList), h.GetDemandPrimaryAction)
	g.POST("/demands/:id/participate-action", middleware.RequirePerm(perm.PoHomeList), h.ParticipateAction)

	g.GET("/done", middleware.RequirePerm(perm.PoDoneList), h.Done)
	g.GET("/done/items", middleware.RequirePerm(perm.PoDoneList), h.DoneItems)
	g.GET("/done/meta", middleware.RequirePerm(perm.PoDoneList), h.DoneMeta)
	g.GET("/done/detail/:actionId", middleware.RequirePerm(perm.PoDoneList), h.DoneDetail)

	g.GET("/todos", middleware.RequirePerm(perm.PoTodoList), h.Todos)
	g.GET("/todos/items", middleware.RequirePerm(perm.PoTodoList), h.TodosItems)

	g.GET("/notice", middleware.RequirePerm(perm.PoNoticeList), h.Notice)
	g.GET("/notice/items", middleware.RequirePerm(perm.PoNoticeList), h.NoticeItems)
	g.PUT("/notice/:id/read", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkRead)
	g.PUT("/notice/read-all", middleware.RequirePerm(perm.PoNoticeUpdate), h.NoticeMarkAllRead)

	g.GET("/follow", middleware.RequirePerm(perm.PoFollowList), h.Follow)
	g.GET("/follow/items", middleware.RequirePerm(perm.PoFollowList), h.FollowItems)
	g.GET("/version-follow", middleware.RequirePerm(perm.PoSchedule), h.VersionFollow)
	g.GET("/version-follow/items", middleware.RequirePerm(perm.PoSchedule), h.VersionFollowItems)
	g.GET("/version-follow/orphan-evidence", middleware.RequirePerm(perm.PoSchedule), h.VersionFollowOrphanEvidence)
	g.GET("/follow/demands/export", middleware.RequirePerm(perm.PoFollowList), h.FollowDemandExport)
	g.GET("/follow/project-weeklies", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklies)
	g.GET("/follow/project-weeklies/:id", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyDetail)
	g.GET("/follow/project-weeklies/:id/history", middleware.RequirePerm(perm.PoFollowList), h.ProjectWeeklyHistory)
	g.PUT("/follow/demand/:id", middleware.RequirePerm(perm.PoFollowUpdate), h.FollowSetDemand)
	g.PUT("/follow/project-report/:id", middleware.RequirePerm(perm.PoFollowUpdate), h.FollowRemoveProjectReport)

	// Keep the sidebar's public path aligned with the page capability name.
	// The singular path remains as a compatibility alias for existing links.
	for _, path := range []string{"/issues/risk", "/issue-risk"} {
		g.GET(path, middleware.RequirePerm(perm.KanbanStory), h.issueRiskAccess(), h.IssueRisk)
		g.GET(path+"/items", middleware.RequirePerm(perm.KanbanStory), h.issueRiskAccess(), h.IssueRiskItems)
	}

	NewBoardHandler(h.svc, h.logger).RegisterRoutes(g)
}

func (h *Handler) homePageAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := middleware.CurrentUser(c)
		if actor == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		canDemandView := currentUserHasPerm(c, perm.PoHomeList)
		requestedView := c.Query("view")
		if requestedView == "demand" && !canDemandView {
			h.denyHomeAccess(c)
			return
		}
		needsTeamCheck := requestedView == "team" || !canDemandView
		if needsTeamCheck {
			if h.teamViewAccess != nil {
				allowed, err := h.teamViewAccess(c.Request.Context(), actor)
				if err != nil {
					c.AbortWithStatus(http.StatusServiceUnavailable)
					return
				}
				if allowed {
					c.Set("teamHomeOnly", true)
					c.Next()
					return
				}
			}
			if requestedView == "team" || !canDemandView {
				h.denyHomeAccess(c)
				return
			}
		}
		c.Next()
	}
}

func (h *Handler) denyHomeAccess(c *gin.Context) {
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "无权限访问"})
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusForbidden)
	c.Abort()
	_, _ = c.Writer.WriteString("<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>无权限</title></head><body><h1>无权限访问</h1></body></html>")
}

// Home 渲染 PO 工作台首页。
func (h *Handler) Home(c *gin.Context) {
	if teamOnly, _ := c.Get("teamHomeOnly"); teamOnly == true {
		render.Page(c, http.StatusOK, "po/home_team", gin.H{
			"Title":             "团队视角",
			"PageTitle":         "团队视角",
			"PageDescription":   "按授权范围查看敏捷团队与小组",
			"CanViewDemandHome": currentUserHasPerm(c, perm.PoHomeList),
		})
		return
	}
	canViewTeamHome := false
	if h.teamViewAccess != nil {
		if allowed, err := h.teamViewAccess(c.Request.Context(), middleware.CurrentUser(c)); err == nil {
			canViewTeamHome = allowed
		}
	}
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
	canViewIssueRisk := currentUserHasPerm(c, perm.PoBoardDemandList)
	if canViewIssueRisk && pageError == "" {
		counts, countErr := h.svc.HomeIssueRiskCounts(c.Request.Context(), actor)
		if countErr != nil {
			if h.logger != nil {
				h.logger.Warn("po home issue/risk counts", zap.Error(countErr))
			}
		} else {
			resp.IssueRiskCounts = counts
		}
	}

	if h.logger != nil {
		h.logger.Info("po home version windows render",
			zap.String("account", account),
			zap.Int("render_count", len(resp.VersionWindows)),
		)
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":               "首页",
		"PageTitle":           "首页",
		"PageDescription":     "待办事项与需求进度",
		"AllCount":            resp.AllCount,
		"ValueStreamStages":   resp.Stages,
		"StagesValid":         resp.StagesValid,
		"VersionWindows":      resp.VersionWindows,
		"VersionWindowsError": versionWindowsError,
		"CanViewIssueRisk":    canViewIssueRisk,
		"CanViewTeamHome":     canViewTeamHome,
		"IssueRiskCounts":     resp.IssueRiskCounts,
		"KPI":                 resp.KPI,
		"PageError":           pageError,
	})
}

func currentUserHasPerm(c *gin.Context, p perm.Permission) bool {
	actor := middleware.CurrentUser(c)
	if actor == nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	value, ok := c.Get("userPerms")
	if !ok {
		return false
	}
	perms, ok := value.(map[string]bool)
	return ok && perms[p.String()]
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

// GetDemandPrimaryAction 返回指定需求的主行动推导
func (h *Handler) GetDemandPrimaryAction(c *gin.Context) {
	id, err := parseDeliverDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}
	actor := middleware.CurrentUser(c)
	actions, err := h.svc.DeriveDemandPrimaryActions(c.Request.Context(), actor, []uint{id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	pa, ok := actions[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到主行动"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": pa})
}

// ParticipateAction 处理参与人员主行动代理请求
func (h *Handler) ParticipateAction(c *gin.Context) {
	id, err := parseDeliverDemandID(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "需求 ID 无效"})
		return
	}
	actor := middleware.CurrentUser(c)
	storyIDsMap, err := h.svc.prepareParticipateStoryActions(c.Request.Context(), actor, []itemRef{{kind: "demand", id: int(id), stageStatus: "schedule"}}, DemandsReq{Relation: "participate"}, make(map[uint]primaryaction.PrimaryAction))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	storyID := storyIDsMap[int(id)]
	c.JSON(http.StatusOK, gin.H{"success": true, "storyId": storyID})
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
