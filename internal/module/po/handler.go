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
	"encoding/json"
	"net/http"

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

	g.GET("/home", h.Home)
	g.GET("/demands", h.Demands)
	g.GET("/demands/:id", middleware.RequirePerm(perm.PoDemandReview), h.DemandDetail)
	// 评审资格在 Service 里按 zt_demandreview 业务评审人校验（与指派给无关）。
	g.POST("/demands/:id/review", middleware.RequirePerm(perm.PoDemandReview), h.ReviewDemand)
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
		resp = &HomeResp{Stages: emptyValueStreamStages(), LaunchWindows: []LaunchWindowOption{}, Users: []UserOption{}}
	}
	if resp.LaunchWindows == nil {
		resp.LaunchWindows = []LaunchWindowOption{}
	}
	if resp.Users == nil {
		resp.Users = []UserOption{}
	}

	if h.logger != nil {
		h.logger.Info("po home version windows render",
			zap.String("account", account),
			zap.Int("render_count", len(resp.VersionWindows)),
			zap.Int("launch_window_count", len(resp.LaunchWindows)),
		)
	}

	launchWindowsJSON := "[]"
	if b, err := json.Marshal(resp.LaunchWindows); err == nil {
		launchWindowsJSON = string(b)
	}
	usersJSON := "[]"
	if b, err := json.Marshal(resp.Users); err == nil {
		usersJSON = string(b)
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":             "工作台首页",
		"PageTitle":         "工作台首页",
		"ValueStreamStages": resp.Stages,
		"VersionWindows":    resp.VersionWindows,
		"LaunchWindowsJSON": launchWindowsJSON,
		"UsersJSON":         usersJSON,
	})
}

// Demands 按价值流状态返回当前用户的需求/故事详情（JSON，后端分页）。
func (h *Handler) Demands(c *gin.Context) {
	var req DemandsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数解析失败",
		})
		return
	}
	req.Normalize()
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
		})
	}
	return stages
}

// DemandDetail 返回业需评审抽屉所需详情 JSON。
func (h *Handler) DemandDetail(c *gin.Context) {
	req := DemandDetailReq{ID: c.Param("id")}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "需求 ID 无效"})
		return
	}

	resp, err := h.svc.GetDemandDetail(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("po demand detail", zap.Error(err), zap.String("id", req.ID))
		}
		status, msg := demandDetailHTTPError(err)
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

func demandDetailHTTPError(err error) (int, string) {
	if biz, ok := errorx.IsBizError(err); ok {
		switch biz.Code {
		case errorx.ErrCodeForbidden:
			return http.StatusForbidden, biz.Msg
		case errorx.ErrCodeNotFound:
			return http.StatusNotFound, biz.Msg
		case errorx.ErrCodeInvalidParam:
			return http.StatusBadRequest, biz.Msg
		}
		return http.StatusBadRequest, biz.Msg
	}
	return http.StatusInternalServerError, "获取需求详情失败"
}
