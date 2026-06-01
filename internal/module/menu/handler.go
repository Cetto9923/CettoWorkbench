// =============================================================================
// 文件: internal/module/menu/handler.go
// 模块: 菜单管理
// 类型: crud
// 职责: 处理菜单管理页面请求并调用 Service 完成增删改查。
// 依赖: internal/middleware
//       internal/model
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package menu

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
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// FlatItem 用于列表页扁平渲染树结构。
type FlatItem struct {
	ID       uint64
	ParentID uint64
	Title    string
	Icon     string
	Path     string
	Perm     string
	Sort     int
	Level    int
	Indent   string
	Type     string // 新增，值为 M / C / F，来自 model.Menu.Type
}

// ParentOption 表示父菜单下拉选项。
type ParentOption struct {
	ID    uint64
	Title string
}

// Handler 处理菜单管理页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建菜单模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册菜单模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/menus")
	g.Use(middleware.ActiveNav("/admin/menus"))
	{
		g.GET("", middleware.RequirePerm(perm.MenuList), h.List)
		g.GET("/new", middleware.RequirePerm(perm.MenuCreate), h.NewForm)
		g.POST("", middleware.RequirePerm(perm.MenuCreate), h.Create)
		g.GET("/:id/edit", middleware.RequirePerm(perm.MenuEdit), h.EditForm)
		g.POST("/:id", middleware.RequirePerm(perm.MenuEdit), h.Update)
		g.PUT("/:id", middleware.RequirePerm(perm.MenuEdit), h.Update)
		g.DELETE("/:id", middleware.RequirePerm(perm.MenuDelete), h.Delete)
	}
}

// List 渲染菜单树形列表（扁平化展示）。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		h.logger.Error("list menu tree failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取菜单列表失败", err)
		return
	}
	nodes := buildTree(resp.Items)
	render.Page(c, http.StatusOK, constants.TEMPLATE_MENU_LIST, gin.H{
		"Title":     "菜单管理",
		"PageTitle": "菜单管理",
		"Items":     flatten(nodes, 1),
	})
}

// NewForm 渲染新建菜单页。
func (h *Handler) NewForm(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	listResp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取菜单列表失败", err)
		return
	}
	nodes := buildTree(filterParentCandidates(listResp.Items))
	render.Page(c, http.StatusOK, constants.TEMPLATE_MENU_CREATE, gin.H{
		"Title":         "新增菜单",
		"PageTitle":     "新增菜单",
		"Form":          &CreateReq{},
		"ParentOptions": toParentOptions(flatten(nodes, 0), nil),
		"Errors":        []FieldError{},
	})
}

// Create 处理新增菜单提交。
func (h *Handler) Create(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req CreateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		h.renderCreateForm(c, &req, errs)
		return
	}
	_, err := h.svc.Create(c.Request.Context(), actor, req)
	if err != nil {
		flash.Error(c, "创建菜单失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/menus/new")
		return
	}
	flash.Success(c, "菜单创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/menus")
}

// EditForm 渲染编辑菜单页。
func (h *Handler) EditForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的菜单 ID", nil)
		return
	}
	actor := middleware.CurrentUser(c)
	m, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "菜单不存在", err)
		return
	}
	listResp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取菜单列表失败", err)
		return
	}
	nodes := buildTree(filterParentCandidates(listResp.Items))
	excluded := descendantsSet(nodes, id)
	excluded[id] = true
	render.Page(c, http.StatusOK, constants.TEMPLATE_MENU_EDIT, gin.H{
		"Title":         "编辑菜单",
		"PageTitle":     "编辑菜单",
		"Form":          NewUpdateReqFromModel(m),
		"Resource":      m,
		"ParentOptions": toParentOptions(flatten(nodes, 0), excluded),
		"Errors":        []FieldError{},
	})
}

// Update 处理编辑菜单提交。
func (h *Handler) Update(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的菜单 ID", nil)
		return
	}
	var req UpdateReq
	if err := c.ShouldBind(&req); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		resource, _ := h.svc.GetByID(c.Request.Context(), actor, id)
		h.renderEditForm(c, id, &req, resource, errs)
		return
	}
	if req.ParentID == id {
		flash.Error(c, "父菜单不能选择自己")
		c.Redirect(http.StatusSeeOther, "/admin/menus/"+strconv.FormatUint(id, 10)+"/edit")
		return
	}
	_, err := h.svc.Update(c.Request.Context(), actor, req)
	if err != nil {
		flash.Error(c, "更新菜单失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/menus/"+strconv.FormatUint(id, 10)+"/edit")
		return
	}
	flash.Success(c, "菜单更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/menus")
}

// Delete 删除菜单。
func (h *Handler) Delete(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		flash.Error(c, "删除失败: 菜单 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/menus")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id}); err != nil {
		flash.Error(c, "删除失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/menus")
		return
	}
	flash.Success(c, "菜单已删除")
	c.Redirect(http.StatusSeeOther, "/admin/menus")
}

func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	listResp, _ := h.svc.List(c.Request.Context(), actor, ListReq{})
	nodes := buildTree(filterParentCandidates(listResp.Items))
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_MENU_CREATE, gin.H{
		"Title":         "新增菜单",
		"PageTitle":     "新增菜单",
		"Form":          req,
		"ParentOptions": toParentOptions(flatten(nodes, 0), nil),
		"Errors":        errs,
	})
}

func (h *Handler) renderEditForm(c *gin.Context, id uint64, req *UpdateReq, resource *model.Menu, errs []FieldError) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	listResp, _ := h.svc.List(c.Request.Context(), actor, ListReq{})
	nodes := buildTree(filterParentCandidates(listResp.Items))
	excluded := descendantsSet(nodes, id)
	excluded[id] = true
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_MENU_EDIT, gin.H{
		"Title":         "编辑菜单",
		"PageTitle":     "编辑菜单",
		"Form":          req,
		"Resource":      resource,
		"ParentOptions": toParentOptions(flatten(nodes, 0), excluded),
		"Errors":        errs,
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

func flatten(nodes []*MenuNode, level int) []FlatItem {
	items := make([]FlatItem, 0)
	for _, node := range nodes {
		if node == nil || node.Menu == nil {
			continue
		}
		items = append(items, FlatItem{
			ID:       node.Menu.ID,
			ParentID: node.Menu.ParentID,
			Title:    node.Menu.Title,
			Icon:     node.Menu.Icon,
			Path:     node.Menu.Path,
			Perm:     node.Menu.Perm,
			Sort:     node.Menu.Sort,
			Level:    level,
			Indent:   strings.Repeat("— ", level),
			Type:     node.Menu.Type,
		})
		items = append(items, flatten(node.Children, level+1)...)
	}
	return items
}

func toParentOptions(items []FlatItem, excluded map[uint64]bool) []ParentOption {
	out := make([]ParentOption, 0, len(items)+1)
	out = append(out, ParentOption{ID: 0, Title: "顶级菜单"})
	seen := make(map[uint64]struct{}, len(items))
	for _, item := range items {
		if excluded != nil && excluded[item.ID] {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		prefix := strings.Repeat("— ", item.Level)
		out = append(out, ParentOption{
			ID:    item.ID,
			Title: prefix + item.Title,
		})
	}
	return out
}

func filterParentCandidates(rows []model.Menu) []model.Menu {
	out := make([]model.Menu, 0, len(rows))
	seen := make(map[uint64]struct{}, len(rows))
	for _, row := range rows {
		if isButtonMenuRow(row) {
			continue
		}
		if _, ok := seen[row.ID]; ok {
			continue
		}
		seen[row.ID] = struct{}{}
		out = append(out, row)
	}
	return out
}

func isButtonMenuRow(row model.Menu) bool {
	return strings.TrimSpace(row.Path) == "" && strings.TrimSpace(row.Perm) != ""
}

func descendantsSet(nodes []*MenuNode, id uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	var walk func(list []*MenuNode) bool
	var markAll func(node *MenuNode)
	markAll = func(node *MenuNode) {
		for _, child := range node.Children {
			if child == nil || child.Menu == nil {
				continue
			}
			out[child.Menu.ID] = true
			markAll(child)
		}
	}
	walk = func(list []*MenuNode) bool {
		for _, node := range list {
			if node == nil || node.Menu == nil {
				continue
			}
			if node.Menu.ID == id {
				markAll(node)
				return true
			}
			if walk(node.Children) {
				return true
			}
		}
		return false
	}
	_ = walk(nodes)
	return out
}
