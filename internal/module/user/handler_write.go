// =============================================================================
// 文件: internal/module/user/handler_write.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户创建、批量创建、更新写操作及对应表单重渲染工具。
// 依赖: internal/middleware
//
//	internal/model
//	internal/pkg/flash
//	internal/pkg/render
//
// =============================================================================

package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/render"
)

// Create 处理新增用户提交。
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}

	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		h.renderCreateForm(c, &req, fieldErrs)
		return
	}

	actor := middleware.CurrentUser(c)
	if _, err := h.svc.Create(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "创建用户失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/users/new")
		return
	}
	flash.Success(c, "用户创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/users")
}

// BatchCreate 处理批量新增用户提交。
func (h *Handler) BatchCreate(c *gin.Context) {
	var req BatchCreateReq
	var err error
	if expectsJSON(c) {
		err = c.ShouldBindJSON(&req)
	} else {
		err = c.ShouldBind(&req)
	}
	if err != nil {
		if expectsJSON(c) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "参数解析失败",
			})
			return
		}
		flash.Error(c, "参数解析失败")
		c.Redirect(http.StatusSeeOther, "/admin/users/batch/create")
		return
	}

	validUsers := make([]CreateReq, 0, len(req.Users))
	rowErrs := make([]BatchFieldError, 0)
	seen := make(map[string]int)
	for idx := range req.Users {
		userReq := req.Users[idx]
		userReq.Account = strings.TrimSpace(userReq.Account)
		userReq.Email = strings.TrimSpace(userReq.Email)
		userReq.DisplayName = strings.TrimSpace(userReq.DisplayName)
		userReq.Gender = strings.TrimSpace(userReq.Gender)
		userReq.Password = strings.TrimSpace(userReq.Password)

		if isBatchRowEmpty(userReq) {
			continue
		}

		userReq.PasswordConfirm = userReq.Password
		userReq.IsActive = true
		for _, fe := range userReq.Validate() {
			rowErrs = append(rowErrs, BatchFieldError{
				Row:     idx + 1,
				Field:   fe.Field,
				Message: fe.Message,
			})
		}
		accountKey := strings.ToLower(userReq.Account)
		if prevRow, ok := seen[accountKey]; ok {
			rowErrs = append(rowErrs, BatchFieldError{
				Row:     idx + 1,
				Field:   "account",
				Message: "与第 " + strconv.Itoa(prevRow) + " 行用户名重复",
			})
		} else {
			seen[accountKey] = idx + 1
		}
		validUsers = append(validUsers, userReq)
	}

	if len(validUsers) == 0 {
		rowErrs = append(rowErrs, BatchFieldError{
			Row:     1,
			Field:   "_form",
			Message: "请至少填写一条用户数据",
		})
	}
	if len(rowErrs) > 0 {
		if expectsJSON(c) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "参数校验失败",
				"errors":  rowErrs,
			})
			return
		}
		flash.Error(c, "参数校验失败")
		c.Redirect(http.StatusSeeOther, "/admin/users/batch/create")
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.BatchCreate(c.Request.Context(), actor, BatchCreateReq{Users: validUsers})
	if err != nil {
		if expectsJSON(c) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "批量创建失败: " + err.Error(),
			})
			return
		}
		flash.Error(c, "批量创建失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/users/batch/create")
		return
	}

	if expectsJSON(c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "批量创建成功",
			"count":   resp.Count,
		})
		return
	}
	flash.Success(c, "批量创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/users")
}

// Update 处理编辑用户提交。
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的用户 ID", nil)
		return
	}

	var req UpdateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.ID = id

	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		actor := middleware.CurrentUser(c)
		// 重新加载原始用户对象，保证模板 .Resource.* 字段完整
		resource, _ := h.svc.GetByID(c.Request.Context(), actor, id)
		h.renderEditForm(c, &req, resource, fieldErrs)
		return
	}

	actor := middleware.CurrentUser(c)
	if _, err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "更新用户失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/users/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	flash.Success(c, "用户更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/users")
}

// renderCreateForm 在新建表单验证失败时重新渲染页面，保留用户已填写的内容。
func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	actor := middleware.CurrentUser(c)
	roles, _ := h.svc.GetRoles(c.Request.Context(), actor)
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_USER_CREATE, gin.H{
		"Title":           "新增用户",
		"PageTitle":       "新增用户",
		"Form":            req,
		"Roles":           roles,
		"SelectedRoleIDs": roleIDSet(req.RoleIDs),
		"Errors":          errs,
	})
}

// renderEditForm 在编辑表单验证失败时重新渲染页面。
// resource 必须传入从数据库加载的原始用户对象，
// 保证模板中所有 .Resource.* 字段（如 .Resource.ID）完整可用。
// 禁止用 *UpdateReq 替代 resource 参数——两者字段集合不同，
// 模板引用缺失字段时会静默渲染为零值，难以排查。
func (h *Handler) renderEditForm(c *gin.Context, req *UpdateReq, resource *model.User, errs []FieldError) {
	actor := middleware.CurrentUser(c)
	roles, _ := h.svc.GetRoles(c.Request.Context(), actor)
	depts, _ := h.svc.GetDepts(c.Request.Context(), actor)
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_USER_EDIT, gin.H{
		"Title":           "编辑用户",
		"PageTitle":       "编辑用户",
		"Form":            req,
		"Resource":        resource,
		"Roles":           roles,
		"Depts":           depts,
		"SelectedRoleIDs": roleIDSet(req.RoleIDs),
		"Errors":          errs,
	})
}

// roleIDSet 将角色 ID 切片转 map,便于模板快速判断。
func roleIDSet(roleIDs []int64) map[int64]bool {
	set := make(map[int64]bool, len(roleIDs))
	for _, id := range roleIDs {
		set[id] = true
	}
	return set
}

// expectsJSON 判定当前请求是否期望 JSON 响应（ajax）。
func expectsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	requestedWith := strings.ToLower(c.GetHeader("X-Requested-With"))
	return strings.Contains(accept, "application/json") || requestedWith == "xmlhttprequest"
}

// isBatchRowEmpty 判定批量创建行是否全部为空字段。
func isBatchRowEmpty(req CreateReq) bool {
	return strings.TrimSpace(req.Account) == "" &&
		strings.TrimSpace(req.Email) == "" &&
		strings.TrimSpace(req.DisplayName) == "" &&
		strings.TrimSpace(req.Gender) == "" &&
		strings.TrimSpace(req.Password) == "" &&
		req.DeptID == 0 &&
		len(req.RoleIDs) == 0
}
