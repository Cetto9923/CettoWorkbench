// =============================================================================
// 文件: internal/module/user/handler_list.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户列表页视图渲染及绑定渲染器。
// 依赖: internal/middleware
//
//	internal/pkg/pagination
//	internal/pkg/render
//
// =============================================================================

package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/render"
)

// List 渲染用户列表页。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.Normalize()

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("query users failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取用户列表失败", err)
		return
	}
	pager := pagination.New(resp.Total, req.Page, pagination.DefaultPageSize)
	render.Page(c, http.StatusOK, constants.TEMPLATE_USER_LIST, gin.H{
		"Title":      "用户管理",
		"PageTitle":  "用户管理",
		"Users":      resp.Items,
		"Pager":      pager,
		"Status":     req.Status,
		"SearchForm": req,
	})
}

// bindRenderer 将 renderer 注入 gin.Context,供模板自定义函数访问。
func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}
