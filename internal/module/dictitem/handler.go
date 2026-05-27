// =============================================================================
// 文件: internal/module/dictitem/handler.go
// 模块: 字典值
// 类型: crud
// 职责: 处理字典值管理页面请求和 JSON 查询接口。
// 依赖: internal/middleware
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package dictitem

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"goframework/internal/middleware"
	"goframework/internal/pkg/flash"
	"goframework/internal/pkg/perm"
	"goframework/internal/pkg/render"
)

// Handler 处理字典值页面请求。
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

// RegisterRoutes 注册字典值路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/dict-items")
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

	// 登录即可访问的字典值 JSON 查询接口，不绑定权限中间件。
	rg.GET("/dict/items", h.ListByTypeJSON)
}

// List 渲染字典值列表。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	var req ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	if strings.TrimSpace(req.TypeCode) == "" {
		req.TypeCode = strings.TrimSpace(c.Query("type"))
	}
	currentTypeName := strings.TrimSpace(c.Query("typeName"))
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("list dict items failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取字典值失败", err)
		return
	}
	render.Page(c, http.StatusOK, "dictitem/list", gin.H{
		"Title":           "字典值管理",
		"PageTitle":       "字典值管理",
		"TypeCode":        req.TypeCode,
		"CurrentTypeCode": req.TypeCode,
		"CurrentTypeName": currentTypeName,
		"Items":           resp.Items,
	})
}

// ListByTypeJSON 返回指定 typeCode 的字典值 JSON。
func (h *Handler) ListByTypeJSON(c *gin.Context) {
	var req ListReq
	_ = c.ShouldBindQuery(&req)
	if strings.TrimSpace(req.TypeCode) == "" {
		req.TypeCode = strings.TrimSpace(c.Query("type"))
	}
	req.Normalize()
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询字典值失败",
		})
		return
	}
	list := make([]gin.H, 0, len(resp.Items))
	for _, item := range resp.Items {
		list = append(list, gin.H{
			"id":       item.ID,
			"typeCode": item.TypeCode,
			"label":    item.Label,
			"value":    item.Value,
			"sort":     item.Sort,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"items":   list,
	})
}

// NewForm 渲染新建页。
func (h *Handler) NewForm(c *gin.Context) {
	h.bindRenderer(c)
	render.Page(c, http.StatusOK, "dictitem/create", gin.H{
		"Title":     "新增字典值",
		"PageTitle": "新增字典值",
		"Form": &CreateReq{
			TypeCode: strings.TrimSpace(c.Query("typeCode")),
		},
		"Errors": []FieldError{},
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
		flash.Error(c, "创建字典值失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-items/new?typeCode="+req.TypeCode)
		return
	}
	flash.Success(c, "字典值创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/dict-items?typeCode="+req.TypeCode)
}

// EditForm 渲染编辑页。
func (h *Handler) EditForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的字典值 ID", nil)
		return
	}
	actor := middleware.CurrentUser(c)
	resource, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "字典值不存在", err)
		return
	}
	render.Page(c, http.StatusOK, "dictitem/edit", gin.H{
		"Title":     "编辑字典值",
		"PageTitle": "编辑字典值",
		"Form":      NewUpdateReqFromModel(resource),
		"Resource":  resource,
		"Errors":    []FieldError{},
	})
}

// Update 处理编辑提交。
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的字典值 ID", nil)
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
		flash.Error(c, "更新字典值失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-items/"+strconv.FormatUint(id, 10)+"/edit")
		return
	}
	flash.Success(c, "字典值更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/dict-items?typeCode="+req.TypeCode)
}

// Delete 删除字典值。
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "删除失败: 字典值 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/dict-items")
		return
	}
	actor := middleware.CurrentUser(c)
	resource, _ := h.svc.GetByID(c.Request.Context(), actor, id)
	if err := h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id}); err != nil {
		flash.Error(c, "删除失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/dict-items")
		return
	}
	flash.Success(c, "字典值已删除")
	if resource != nil {
		c.Redirect(http.StatusSeeOther, "/admin/dict-items?typeCode="+resource.TypeCode)
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/dict-items")
}

func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, "dictitem/create", gin.H{
		"Title":     "新增字典值",
		"PageTitle": "新增字典值",
		"Form":      req,
		"Errors":    errs,
	})
}

func (h *Handler) renderEditForm(c *gin.Context, req *UpdateReq, resource any, errs []FieldError) {
	h.bindRenderer(c)
	render.Page(c, http.StatusUnprocessableEntity, "dictitem/edit", gin.H{
		"Title":     "编辑字典值",
		"PageTitle": "编辑字典值",
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
