// =============================================================================
// 文件: internal/module/operationlog/handler.go
// 模块: 操作日志
// 类型: readonly
// 职责: 处理操作日志列表页面请求。
// 依赖: internal/middleware
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package operationlog

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 操作日志 Handler。
type Handler struct {
	svc *Service
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/operation-logs", middleware.ActiveNav("/admin/operation-logs"), middleware.RequirePerm(perm.OperationLogList), h.List)
}

// List 操作日志列表页。
func (h *Handler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.Account = strings.TrimSpace(req.Account)
	req.Method = strings.TrimSpace(req.Method)
	req.Path = strings.TrimSpace(req.Path)

	resp, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取操作日志失败", err)
		return
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_OPERATIONLOG_LIST, gin.H{
		"Title":     "操作日志",
		"PageTitle": "操作日志",
		"Form":      &req,
		"Logs":      resp.Items,
		"Pager":     resp.Pager,
		"Account":   req.Account,
		"Method":    req.Method,
		"Path":      req.Path,
	})
}
