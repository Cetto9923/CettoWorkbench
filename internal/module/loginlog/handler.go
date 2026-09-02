// =============================================================================
// 文件: internal/module/loginlog/handler.go
// 模块: 登录日志
// 类型: readonly
// 职责: 处理登录日志列表页面请求。
// 依赖: internal/middleware
//       internal/pkg/pagination
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package loginlog

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 登录日志 Handler。
type Handler struct {
	svc *Service
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/login-logs")
	g.Use(middleware.ActiveNav("/admin/login-logs"))
	g.GET("", middleware.RequirePerm(perm.LoginLogList), h.List)
}

// List 登录日志列表页。
func (h *Handler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.Normalize()
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_LOGINLOG_LIST, gin.H{
			"Title":     "登录日志",
			"PageTitle": "登录日志",
			"Form":      &req,
			"Errors":    fieldErrs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取登录日志失败", err)
		return
	}

	pager := pagination.New(resp.Total, req.Page, pagination.DefaultPageSize)
	render.Page(c, http.StatusOK, constants.TEMPLATE_LOGINLOG_LIST, gin.H{
		"Title":     "登录日志",
		"PageTitle": "登录日志",
		"Form":      &req,
		"Errors":    []FieldError{},
		"Logs":      resp.Items,
		"Pager":     pager,
	})
}
