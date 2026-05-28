// =============================================================================
// 文件: internal/module/dicttype/handler.go
// 模块: 字典类型
// 类型: crud
// 职责: 处理字典类型管理页面请求。
// 依赖: internal/middleware
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package dicttype

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理字典类型页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册字典类型路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/dict-types")
	g.Use(middleware.ActiveNav("/admin/dict-types"))
	{
		g.GET("", middleware.RequirePerm(perm.DictList), h.List)
		g.GET("/new", middleware.RequirePerm(perm.DictCreate), h.NewForm)
		g.POST("", middleware.RequirePerm(perm.DictCreate), h.Create)
		g.GET("/:id/edit", middleware.RequirePerm(perm.DictEdit), h.EditForm)
		g.POST("/:id", middleware.RequirePerm(perm.DictEdit), h.Update)
		g.PUT("/:id", middleware.RequirePerm(perm.DictEdit), h.Update)
		g.DELETE("/:id", middleware.RequirePerm(perm.DictDelete), h.Delete)
	}
}

// List 渲染字典类型列表。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("list dict types failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取字典类型失败", err)
		return
	}
	render.Page(c, http.StatusOK, "dicttype/list", gin.H{
		"Title":     "字典类型管理",
		"PageTitle": "字典类型管理",
		"Q":         req.Keyword,
		"Items":     resp.Items,
	})
}

// NewForm 渲染新建页。
func (h *Handler) NewForm(c *gin.Context) {
	h.bindRenderer(c)
	render.Page(c, http.StatusOK, "dicttype/create", gin.H{
		"Title":     "新增字典类型",
		"PageTitle": "新增字典类型",
		"Form":      &CreateReq{},
		"Errors":    []FieldError{},
	})
}

// Create 处理新建提交。
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		h.renderCreateForm(c, &req, errs)
		return
	}
	actor := middleware.CurrentUser(c)
	if _, err := h.svc.Create(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "创建字典类型失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-types/new")
		return
	}
	flash.Success(c, "字典类型创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/dict-types")
}

// EditForm 渲染编辑页。
func (h *Handler) EditForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的字典类型 ID", nil)
		return
	}
	actor := middleware.CurrentUser(c)
	resource, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "字典类型不存在", err)
		return
	}
	render.Page(c, http.StatusOK, "dicttype/edit", gin.H{
		"Title":     "编辑字典类型",
		"PageTitle": "编辑字典类型",
		"Form":      NewUpdateReqFromModel(resource),
		"Resource":  resource,
		"Errors":    []FieldError{},
	})
}

// Update 处理编辑提交。
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的字典类型 ID", nil)
		return
	}
	var req UpdateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		actor := middleware.CurrentUser(c)
		resource, _ := h.svc.GetByID(c.Request.Context(), actor, id)
		h.renderEditForm(c, &req, resource, errs)
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		flash.Error(c, "更新字典类型失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-types/"+strconv.FormatUint(id, 10)+"/edit")
		return
	}
	flash.Success(c, "字典类型更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/dict-types")
}

// Delete 删除字典类型。
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "删除失败: 字典类型 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/dict-types")
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id}); err != nil {
		flash.Error(c, "删除失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-types")
		return
	}
	flash.Success(c, "字典类型已删除")
	c.Redirect(http.StatusSeeOther, "/admin/dict-types")
}

func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, "dicttype/create", gin.H{
		"Title":     "新增字典类型",
		"PageTitle": "新增字典类型",
		"Form":      req,
		"Errors":    errs,
	})
}

func (h *Handler) renderEditForm(c *gin.Context, req *UpdateReq, resource any, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, "dicttype/edit", gin.H{
		"Title":     "编辑字典类型",
		"PageTitle": "编辑字典类型",
		"Form":      req,
		"Resource":  resource,
		"Errors":    errs,
	})
}

func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}

func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}
