// =============================================================================
// 文件: internal/module/user/handler_form.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户新建/批量创建/编辑表单页 GET 渲染。
// 依赖: internal/middleware
//
//	internal/pkg/render
//
// =============================================================================

package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// NewForm 渲染新建用户表单页。
func (h *Handler) NewForm(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	roles, err := h.svc.GetRoles(c.Request.Context(), actor)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取角色列表失败", err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_USER_CREATE, gin.H{
		"Title":           "新增用户",
		"PageTitle":       "新增用户",
		"Form":            &CreateReq{IsActive: true, Gender: "m"},
		"Roles":           roles,
		"SelectedRoleIDs": map[int64]bool{},
		"Errors":          []FieldError{},
	})
}

// BatchCreate 渲染批量创建用户表单页。
func (h *Handler) BatchPage(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	roles, err := h.svc.GetRoles(c.Request.Context(), actor)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取角色列表失败", err)
		return
	}
	depts, err := h.svc.GetDepts(c.Request.Context(), actor)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取部门列表失败", err)
		return
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_USER_BATCHCREATE, gin.H{
		"Title":     "批量创建用户",
		"PageTitle": "批量创建用户",
		"Roles":     roles,
		"Depts":     depts,
	})
}

// EditForm 渲染编辑用户表单页。
func (h *Handler) EditForm(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的用户 ID", nil)
		return
	}

	actor := middleware.CurrentUser(c)
	user, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "用户不存在", err)
		return
	}
	roles, err := h.svc.GetRoles(c.Request.Context(), actor)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取角色列表失败", err)
		return
	}
	depts, err := h.svc.GetDepts(c.Request.Context(), actor)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取部门列表失败", err)
		return
	}
	roleIDs, err := h.svc.GetUserRoleIDs(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取用户角色失败", err)
		return
	}
	form := NewUpdateReqFromUser(user)
	form.RoleIDs = roleIDs

	render.Page(c, http.StatusOK, constants.TEMPLATE_USER_EDIT, gin.H{
		"Title":           "编辑用户",
		"PageTitle":       "编辑用户",
		"Form":            form,
		"Resource":        user,
		"Roles":           roles,
		"Depts":           depts,
		"SelectedRoleIDs": roleIDSet(roleIDs),
		"Errors":          []FieldError{},
	})
}
