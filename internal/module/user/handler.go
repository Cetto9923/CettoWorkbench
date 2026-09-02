// =============================================================================
// 文件: internal/module/user/handler.go
// 模块: 用户管理
// 类型: crud
// 职责: 处理用户管理相关 HTTP 请求并完成角色分配表单交互。
// 依赖: internal/middleware
//
//	internal/model
//	internal/pkg/flash
//	internal/pkg/pagination
//	internal/pkg/perm
//	internal/pkg/render
//
// =============================================================================
package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/fileexport"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理用户管理页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建用户模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册用户模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.ActiveNav("/admin/users"))
	users.GET("", middleware.RequirePerm(perm.UserList), h.List)
	users.GET("/new", middleware.RequirePerm(perm.UserCreate), h.NewForm)
	users.GET("/:id/edit", middleware.RequirePerm(perm.UserUpdate), h.EditForm)
	users.GET("/batch/create", middleware.RequirePerm(perm.UserCreate), h.BatchPage)

	users.POST("", middleware.RequirePerm(perm.UserCreate), h.Create)
	users.POST("/export", middleware.RequirePerm(perm.UserList), h.Export)
	users.POST("/batch/create", middleware.RequirePerm(perm.UserCreate), h.BatchCreate)
	users.POST("/:id", middleware.RequirePerm(perm.UserUpdate), h.Update)
	users.POST("/:id/toggle-status", middleware.RequirePerm(perm.UserUpdate), h.ToggleStatus)
	users.POST("/:id/reset-password", middleware.RequirePerm(perm.UserResetPassword), h.ResetPassword)

	users.DELETE("/:id", middleware.RequirePerm(perm.UserDelete), h.Delete)
}

// Export 导出用户列表并触发浏览器下载。
func (h *Handler) Export(c *gin.Context) {
	var req ExportReq
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "导出失败: 参数解析失败")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		flash.Error(c, "导出失败: 参数不合法")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	users, err := h.svc.Export(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("export users failed", zap.Error(err))
		flash.Error(c, "导出失败，请稍后重试")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	fileName := fileexport.BuildFileName(req.FileName, req.FileType, "用户列表")
	if err := fileexport.WriteTable(c.Writer, fileName, req.FileType, exportHeaders(), usersToStringRows(users)); err != nil {
		h.logger.Error("export users failed", zap.Error(err))
		c.String(http.StatusInternalServerError, "导出失败")
		return
	}
}

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

// Delete 删除用户
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		flash.Error(c, "删除用户失败: 用户ID不合法")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	err = h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id})
	if err != nil {
		h.logger.Error("delete user failed", zap.Int64("userID", id), zap.Error(err))
		flash.Error(c, "删除用户失败，请稍后重试")
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}

	flash.Success(c, "删除成功")
	c.Redirect(http.StatusSeeOther, "/admin/users")
}

// ToggleStatus 切换用户启用状态。
func (h *Handler) ToggleStatus(c *gin.Context) {
	var req ToggleStatusReq
	if err := c.ShouldBindUri(&req); err != nil {
		flash.Error(c, "切换用户状态失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "切换用户状态失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.ToggleStatus(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "切换用户状态失败: "+err.Error())
		render.Redirect(c, "/admin/users")
		return
	}
	flash.Success(c, "用户状态切换成功")
	render.Redirect(c, "/admin/users")
}

// ResetPassword 重置用户密码。
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordReq
	if err := c.ShouldBindUri(&req); err != nil {
		flash.Error(c, "重置密码失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "重置密码失败: 参数解析失败")
		render.Redirect(c, "/admin/users")
		return
	}

	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		var msgs []string
		for _, item := range fieldErrs {
			if item.Message != "" {
				msgs = append(msgs, item.Message)
			}
		}
		flash.Error(c, "重置密码失败: "+strings.Join(msgs, "；"))
		render.Redirect(c, "/admin/users")
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.ResetPassword(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "重置密码失败: "+err.Error())
		render.Redirect(c, "/admin/users")
		return
	}
	flash.Success(c, "密码重置成功")
	render.Redirect(c, "/admin/users")
}

func roleIDSet(roleIDs []int64) map[int64]bool {
	set := make(map[int64]bool, len(roleIDs))
	for _, id := range roleIDs {
		set[id] = true
	}
	return set
}

func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}

func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func expectsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	requestedWith := strings.ToLower(c.GetHeader("X-Requested-With"))
	return strings.Contains(accept, "application/json") || requestedWith == "xmlhttprequest"
}

func isBatchRowEmpty(req CreateReq) bool {
	return strings.TrimSpace(req.Account) == "" &&
		strings.TrimSpace(req.Email) == "" &&
		strings.TrimSpace(req.DisplayName) == "" &&
		strings.TrimSpace(req.Gender) == "" &&
		strings.TrimSpace(req.Password) == "" &&
		req.DeptID == 0 &&
		len(req.RoleIDs) == 0
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

func exportHeaders() []string {
	return []string{"ID", "账号", "邮箱", "姓名", "性别", "状态", "最后登录时间"}
}

func usersToStringRows(users []model.User) [][]string {
	rows := make([][]string, 0, len(users))
	for _, item := range users {
		rows = append(rows, []string{
			strconv.FormatInt(item.ID, 10),
			item.Account,
			item.Email,
			item.DisplayName,
			genderLabel(item.Gender),
			activeLabel(item.IsActive),
			formatTime(item.LastLoginDateDB),
		})
	}
	return rows
}
