// =============================================================================
// 文件: internal/module/agileteam/handler.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 列表 / 详情 JSON 接口。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/perm
// =============================================================================

package agileteam

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

const agileteamRedirect = "/agileteam"

// Handler HTTP 入口。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册 /workbench/api/agile-teams 与 /agileteam 路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/agileteam", middleware.RequirePerm(perm.AgileTeamList), h.AgileTeamView)
	rg.GET("/pmo", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/agileteam")
	})
	rg.GET("/workbench/views/pmo", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/agileteam")
	})

	g := rg.Group("/workbench/api/agile-teams")

	g.GET("", middleware.RequirePerm(perm.AgileTeamList), h.List)
	// 静态前缀路由必须在 /:id 之前，避免被参数路由吞掉。
	g.GET("/candidates", middleware.RequirePerm(perm.AgileTeamList), h.SearchCandidates)
	g.GET("/adjustments/:id", middleware.RequirePerm(perm.AgileTeamList), h.AdjustmentDetail)
	g.PUT("/adjustments/:id/confirm", middleware.RequirePerm(perm.AgileTeamConfirm), h.ConfirmAdjustment)
	g.PUT("/adjustments/:id/reject", middleware.RequirePerm(perm.AgileTeamConfirm), h.RejectAdjustment)

	g.GET("/:id", middleware.RequirePerm(perm.AgileTeamList), h.Detail)
	g.PUT("/:id/basic", middleware.RequirePerm(perm.AgileTeamUpdate), h.UpdateBasic)
	g.POST("/:id/adjustments", middleware.RequirePerm(perm.AgileTeamUpdate), h.SubmitAdjustment)
}

// AgileTeamView 渲染敏捷小组治理页面。
func (h *Handler) AgileTeamView(c *gin.Context) {
	render.Page(c, http.StatusOK, "agileteam/index", gin.H{
		"Title":           "敏捷小组",
		"PageTitle":       "敏捷小组",
		"PageDescription": "敏捷团队编制、人员分工与组织架构治理",
	})
}

func (h *Handler) List(c *gin.Context) {
	var req ListReq
	_ = c.ShouldBindQuery(&req)
	resp, err := h.svc.List(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "小组 ID 无效"})
		return
	}
	leadView := strings.ToLower(c.Query("view")) == "lead"
	canConfirm := hasPerm(c, perm.AgileTeamConfirm) && !leadView
	coarseCanEdit := hasPerm(c, perm.AgileTeamUpdate) && !leadView
	canEdit, err := h.svc.CanEditTeamgroupObject(
		c.Request.Context(), middleware.CurrentUser(c), id, coarseCanEdit, canConfirm,
	)
	if err != nil {
		writeErr(c, err)
		return
	}
	resp, err := h.svc.Detail(c.Request.Context(), middleware.CurrentUser(c), id, canConfirm, canEdit)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *Handler) SearchCandidates(c *gin.Context) {
	var req CandidateSearchReq
	_ = c.ShouldBindQuery(&req)
	resp, err := h.svc.SearchCandidates(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

func parseInt64Param(c *gin.Context, name string) (int64, bool) {
	raw := c.Param(name)
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func hasPerm(c *gin.Context, p perm.Permission) bool {
	raw, ok := c.Get("userPerms")
	if !ok {
		return false
	}
	m, ok := raw.(map[string]bool)
	if !ok {
		return false
	}
	return m[p.String()]
}

func writeOK(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": message, "redirectUrl": agileteamRedirect,
	})
}

func writeErr(c *gin.Context, err error) {
	if biz, ok := errorx.IsBizError(err); ok {
		status := http.StatusBadRequest
		switch biz.Code {
		case "not_found":
			status = http.StatusNotFound
		case "forbidden", "unauthorized":
			status = http.StatusForbidden
		case "conflict":
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "message": biz.Msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器错误"})
}
