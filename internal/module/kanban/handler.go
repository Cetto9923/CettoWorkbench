// =============================================================================
// 文件: internal/module/kanban/handler.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板静态页与需求树（业需+研需）JSON HTTP 请求。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/perm
//       internal/pkg/render
//       internal/pkg/zentao
// =============================================================================

package kanban

import (
	"net/http"

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
		"success": true,
		"items":   resp.Items,
	})
}
