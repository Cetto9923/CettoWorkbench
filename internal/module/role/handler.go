// =============================================================================
// 文件: internal/module/role/handler.go
// 模块: 角色管理
// 类型: crud
// 职责: 处理角色 CRUD 与权限分配相关 HTTP 请求。
// 依赖: internal/middleware
//       internal/model
//       internal/pkg/flash
//       internal/pkg/pagination
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package role

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理角色管理页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建角色模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册角色模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/roles")
	g.Use(middleware.ActiveNav("/admin/roles"))
	{
		g.GET("", middleware.RequirePerm(perm.RoleList), h.List)
		g.GET("/new", middleware.RequirePerm(perm.RoleCreate), h.NewForm)
		g.POST("", middleware.RequirePerm(perm.RoleCreate), h.Create)
		g.GET("/:id/permissions", middleware.RequirePerm(perm.RoleEdit), h.AssignPermsForm)
		g.POST("/:id/permissions", middleware.RequirePerm(perm.RoleEdit), h.AssignPerms)
		g.GET("/:id/edit", middleware.RequirePerm(perm.RoleEdit), h.EditForm)
		g.POST("/:id", middleware.RequirePerm(perm.RoleEdit), h.Update)
		g.PUT("/:id", middleware.RequirePerm(perm.RoleEdit), h.Update)
		g.DELETE("/:id", middleware.RequirePerm(perm.RoleDelete), h.Delete)
	}
}

// List 渲染角色列表页。
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
		h.logger.Error("query roles failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取角色列表失败", err)
		return
	}

	pager := pagination.New(resp.Total, req.Page, req.PageSize)
	render.Page(c, http.StatusOK, constants.TEMPLATE_ROLE_LIST, gin.H{
		"Title":     "角色管理",
		"PageTitle": "角色管理",
		"Roles":     resp.Items,
		"Pager":     pager,
		"Q":         req.Keyword,
	})
}

// NewForm 渲染新建角色页（示例文案，不调用 Service）。
func (h *Handler) NewForm(c *gin.Context) {
	h.bindRenderer(c)
	render.Page(c, http.StatusOK, constants.TEMPLATE_ROLE_CREATE, gin.H{
		"Title":     "新增角色",
		"PageTitle": "新增角色",
		"Form": &CreateReq{
			Name:   "（示例）运营专员",
			Remark: "示例说明：提交前请改为真实角色名称与备注。",
		},
		"Errors": []FieldError{},
	})
}

// Create 处理新增角色提交。
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
		flash.Error(c, "创建角色失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/roles/new")
		return
	}
	flash.Success(c, "角色创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/roles")
}

// EditForm 渲染编辑角色页。
func (h *Handler) EditForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的角色 ID", nil)
		return
	}

	actor := middleware.CurrentUser(c)
	role, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "角色不存在", err)
		return
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_ROLE_EDIT, gin.H{
		"Title":     "编辑角色",
		"PageTitle": "编辑角色",
		"Form":      NewUpdateReqFromRole(role),
		"Resource":  role,
		"Errors":    []FieldError{},
	})
}

// Update 处理编辑角色提交。
func (h *Handler) Update(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的角色 ID", nil)
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
		resource, _ := h.svc.GetByID(c.Request.Context(), actor, id)
		h.renderEditForm(c, &req, resource, fieldErrs)
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "更新角色失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/roles/"+strconv.FormatInt(id, 10)+"/edit")
		return
	}
	flash.Success(c, "角色更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/roles")
}

// Delete 删除角色。
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		flash.Error(c, "删除失败: 角色 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/roles")
		return
	}

	actor := middleware.CurrentUser(c)
	err = h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id})
	if err != nil {
		flash.Error(c, err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/roles")
		return
	}
	flash.Success(c, "角色已删除")
	c.Redirect(http.StatusSeeOther, "/admin/roles")
}

// AssignPermsForm 渲染权限分配页。
func (h *Handler) AssignPermsForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的角色 ID", nil)
		return
	}

	actor := middleware.CurrentUser(c)
	role, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "角色不存在", err)
		return
	}

	pageData, err := h.svc.AssignPermsFormData(c.Request.Context(), actor, AssignPermsFormDataReq{RoleID: id})
	if err != nil {
		h.logger.Error("load assign perms page failed", zap.Int64("roleID", id), zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "加载权限数据失败", err)
		return
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_ROLE_ASSIGNPERMS, gin.H{
		"Title":     "分配权限",
		"PageTitle": "分配权限 — " + role.Name,
		"Resource":  role,
		"PermTree":  pageData.PermTree,
	})
}

// AssignPerms 提交角色权限分配。
func (h *Handler) AssignPerms(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "保存失败: 角色 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/roles")
		return
	}

	var req AssignPermsReq
	if err := c.ShouldBind(&req); err != nil {
		flash.Error(c, "保存失败: 参数解析失败")
		c.Redirect(http.StatusSeeOther, "/admin/roles/"+strconv.FormatInt(id, 10)+"/permissions")
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.ReplacePermissions(c.Request.Context(), actor, id, req.PermCodes); err != nil {
		flash.Error(c, "保存权限失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/roles/"+strconv.FormatInt(id, 10)+"/permissions")
		return
	}
	flash.Success(c, "权限已保存")
	c.Redirect(http.StatusSeeOther, "/admin/roles")
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

func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_ROLE_CREATE, gin.H{
		"Title":     "新增角色",
		"PageTitle": "新增角色",
		"Form":      req,
		"Errors":    errs,
	})
}

func (h *Handler) renderEditForm(c *gin.Context, req *UpdateReq, resource *model.Role, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_ROLE_EDIT, gin.H{
		"Title":     "编辑角色",
		"PageTitle": "编辑角色",
		"Form":      req,
		"Resource":  resource,
		"Errors":    errs,
	})
}
