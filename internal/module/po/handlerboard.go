// =============================================================================
// 文件: internal/module/po/handlerboard.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作看板 HTTP 请求。
// 依赖: internal/middleware
// =============================================================================

package po

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
	"workbench/internal/pkg/zentao"
)

// RegisterRoutes 注册看板路由（需求 + 任务 + 各自 items）。
// Rules §1: 路由必须挂 RequirePerm（拆分 boarddemand 与 boardtask 两个权限便于角色授权）。
func (h *BoardHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/board/demand", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardDemand)
	rg.GET("/board/demand/items", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardDemandItems)
	rg.GET("/board/task", middleware.RequirePerm(perm.PoBoardTaskList), h.BoardTask)
	rg.GET("/board/task/items", middleware.RequirePerm(perm.PoBoardTaskList), h.BoardTaskItems)
	rg.PUT("/board/tasks/:id/status", middleware.RequirePerm(perm.PoBoardTaskList), h.TransitionBoardTask)
	rg.GET("/board/issues", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardIssues)
	rg.GET("/board/issues/:id/actions", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardIssueActions)
	rg.POST("/board/issues/:id/transition", middleware.RequirePerm(perm.PoBoardDemandList), h.TransitionBoardIssue)
	rg.GET("/board/group/metrics", middleware.RequirePerm(perm.PoBoardDemandList), h.BoardGroupMetrics)
}

// TransitionBoardTask 处理任务看板拖拽状态变更。
func (h *BoardHandler) TransitionBoardTask(c *gin.Context) {
	taskID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || taskID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "任务 ID 无效"})
		return
	}
	var req BoardTaskTransitionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "任务状态参数无效"})
		return
	}
	err = h.svc.TransitionBoardTask(c.Request.Context(), middleware.CurrentUser(c), uint(taskID64), req)
	if bizErr, ok := errorx.IsBizError(err); ok {
		status := http.StatusUnprocessableEntity
		if bizErr.Code == errorx.ErrCodeForbidden {
			status = http.StatusForbidden
		}
		if bizErr.Code == errorx.ErrCodeNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"message": bizErr.Msg})
		return
	}
	if err != nil {
		h.logger.Error("po board task transition", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"message": "禅道任务状态更新失败，任务状态未变更"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TransitionBoardIssue 转发到禅道原生问题动作接口；不可用时明确返回，绝不修改本地展示状态。
func (h *BoardHandler) TransitionBoardIssue(c *gin.Context) {
	issueID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || issueID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "问题 ID 无效"})
		return
	}
	var req BoardIssueTransitionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "问题操作参数无效"})
		return
	}
	sessionID := ""
	if cookie, cookieErr := c.Request.Cookie(zentao.ZenTaoSessionCookieName); cookieErr == nil {
		sessionID = cookie.Value
	}
	err = h.svc.TransitionBoardIssue(c.Request.Context(), middleware.CurrentUser(c), issueID, req.Action, sessionID)
	if errors.Is(err, zentao.ErrIssueActionUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "code": "zentao_unavailable", "message": "禅道原生问题操作暂不可用，问题状态未变更"})
		return
	}
	if bizErr, ok := errorx.IsBizError(err); ok {
		status := http.StatusUnprocessableEntity
		if bizErr.Code == errorx.ErrCodeForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"message": bizErr.Msg})
		return
	}
	if err != nil {
		h.logger.Error("po board issue transition", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"message": "禅道问题操作失败，问题状态未变更"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// BoardIssueActions 返回当前用户可见问题从创建开始的禅道操作审计记录。
func (h *BoardHandler) BoardIssueActions(c *gin.Context) {
	issueID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || issueID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "问题 ID 无效"})
		return
	}
	afterID, err := strconv.ParseInt(c.DefaultQuery("afterId", "0"), 10, 64)
	if err != nil || afterID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "操作记录游标无效"})
		return
	}
	page, err := h.svc.BoardIssueActions(c.Request.Context(), middleware.CurrentUser(c), issueID, afterID)
	if err != nil {
		if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
			c.JSON(http.StatusForbidden, gin.H{"message": bizErr.Msg})
			return
		}
		h.logger.Error("po board issue actions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取问题流转记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": page.Items, "nextAfterId": page.NextAfterID})
}

// BoardGroupMetrics 返回选定敏捷小组的真实效能指标 JSON。
func (h *BoardHandler) BoardGroupMetrics(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req GroupMetricsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": errs})
		return
	}
	resp, err := h.svc.BoardGroupMetrics(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po board group metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取小组效能指标失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"groupId":   resp.GroupID,
		"hasGroup":  resp.HasGroup,
		"groupName": resp.GroupName,
		"metrics":   resp.Metrics,
	})
}

// BoardIssues 返回右侧问题栏真实问题 JSON。
func (h *BoardHandler) BoardIssues(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.BoardIssues(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("po board issues", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取问题失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   resp.Total,
		"open":    resp.Open,
		"closed":  resp.Closed,
		"items":   resp.Items,
	})
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
		"Title":           "需求看板",
		"PageTitle":       "需求看板",
		"PageDescription": "按需求/故事阶段流转与状态推进看板",
		"BaseUrl":         "/board/demand",
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
		"success":             true,
		"tree":                resp.Tree,
		"summary":             resp.Summary,
		"teamgroups":          resp.Teamgroups,
		"selectedTeamgroupId": resp.SelectedTeamgroupID,
	})
}

// BoardTask 渲染任务看板页。
func (h *BoardHandler) BoardTask(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_BOARD_TASK, gin.H{
		"Title":           "任务看板",
		"PageTitle":       "任务看板",
		"PageDescription": "按任务状态与敏捷小组执行跟进看板",
		"BaseUrl":         "/board/task",
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
