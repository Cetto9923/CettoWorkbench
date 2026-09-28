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
	rg.GET("/home/team/scopes", h.DashboardScopes)
	rg.GET("/agileteam", h.RequireListAccess, h.AgileTeamView)
	rg.GET("/pmo", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/agileteam")
	})
	rg.GET("/workbench/views/pmo", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/agileteam")
	})

	g := rg.Group("/workbench/api/agile-teams")

	g.GET("", h.RequireListAccess, h.List)
	// 静态前缀路由必须在 /:id 之前，避免被参数路由吞掉。
	g.GET("/candidates", middleware.RequirePerm(perm.AgileTeamList), h.SearchCandidates)
	g.GET("/adjustments/:id", middleware.RequirePerm(perm.AgileTeamList), h.AdjustmentDetail)
	g.PUT("/adjustments/:id/confirm", middleware.RequirePerm(perm.AgileTeamConfirm), h.ConfirmAdjustment)
	g.PUT("/adjustments/:id/reject", middleware.RequirePerm(perm.AgileTeamConfirm), h.RejectAdjustment)
	g.GET("/organization-teams", middleware.RequirePerm(perm.AgileTeamList), h.OrgTeamOptions)

	g.GET("/:id", h.RequireListAccess, h.Detail)
	g.PUT("/:id/basic", middleware.RequirePerm(perm.AgileTeamUpdate), h.UpdateBasic)
	g.PUT("/:id/org-team", middleware.RequirePerm(perm.AgileTeamConfirm), h.UpdateOrgTeamMapping)
	g.POST("/:id/adjustments", middleware.RequirePerm(perm.AgileTeamUpdate), h.SubmitAdjustment)
}

// RequireListAccess 保留现有敏捷小组列表权限，同时允许按组织数据自动授权的负责人/教练进入只读视图。
func (h *Handler) RequireListAccess(c *gin.Context) {
	if hasPerm(c, perm.AgileTeamList) {
		c.Next()
		return
	}
	allowed, err := h.svc.CanEnterLeadView(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		writeErr(c, err)
		c.Abort()
		return
	}
	if !allowed {
		writeErr(c, errorx.New("forbidden", "无权查看敏捷团队管理视图"))
		c.Abort()
		return
	}
	c.Next()
}

// AgileTeamView 渲染敏捷小组治理页面。
func (h *Handler) AgileTeamView(c *gin.Context) {
	viewMode := resolveViewMode(c.Query("view"), hasPerm(c, perm.AgileTeamList))
	render.Page(c, http.StatusOK, "agileteam/index", gin.H{
		"Title":           "敏捷小组",
		"PageTitle":       "敏捷小组",
		"PageDescription": "敏捷团队编制、人员分工与组织架构治理",
		"ViewMode":        viewMode,
	})
}

func resolveViewMode(requested string, canListAll bool) string {
	if strings.ToLower(strings.TrimSpace(requested)) == "lead" {
		return "lead"
	}
	if canListAll {
		return "pmo"
	}
	return "lead"
}

func (h *Handler) List(c *gin.Context) {
	var req ListReq
	_ = c.ShouldBindQuery(&req)
	if !hasPerm(c, perm.AgileTeamList) {
		req.View = "lead"
	}
	resp, err := h.svc.List(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		writeErr(c, err)
		return
	}
	resp.CanMapOrgTeam = hasPerm(c, perm.AgileTeamConfirm)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *Handler) OrgTeamOptions(c *gin.Context) {
	items, err := h.svc.ListOrgTeamOptions(c.Request.Context())
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "小组 ID 无效"})
		return
	}
	leadView := strings.ToLower(c.Query("view")) == "lead"
	dynamicLeadView := !hasPerm(c, perm.AgileTeamList)
	leadView = leadView || dynamicLeadView
	if leadView {
		allowed, err := h.svc.CanViewTeamgroupLeadScope(c.Request.Context(), middleware.CurrentUser(c), id)
		if err != nil {
			writeErr(c, err)
			return
		}
		if !allowed {
			writeErr(c, errorx.New("forbidden", "无权查看该敏捷小组"))
			return
		}
	}
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
	resp.MemberDetailsAvailable = !leadView
	if leadView {
		resp.MemberDetailsAvailable, err = h.svc.CanViewTeamgroupMemberDetails(c.Request.Context(), middleware.CurrentUser(c), id)
		if err != nil {
			writeErr(c, err)
			return
		}
		if !resp.MemberDetailsAvailable {
			redactTeamgroupMemberDetails(resp)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func redactTeamgroupMemberDetails(resp *DetailResp) {
	if resp == nil {
		return
	}
	resp.Formal = []MemberItem{}
	resp.PendingJoin = []MemberItem{}
	resp.History = []HistoryItem{}
	if resp.Pending != nil {
		resp.Pending.SubmittedBy = ""
		resp.Pending.AddNames = []string{}
		resp.Pending.RemoveNames = []string{}
		resp.Pending.ChangeNames = []string{}
	}
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

// UpdateOrgTeamMapping 由 PMO 维护敏捷团队与组织团队的主挂靠。
func (h *Handler) UpdateOrgTeamMapping(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "小组 ID 无效"})
		return
	}
	var req OrgTeamMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	req.TeamgroupID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	if err := h.svc.SetOrgTeamMapping(c.Request.Context(), middleware.CurrentUser(c), req, hasPerm(c, perm.AgileTeamConfirm)); err != nil {
		writeErr(c, err)
		return
	}
	writeOK(c, "组织挂靠已更新")
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
