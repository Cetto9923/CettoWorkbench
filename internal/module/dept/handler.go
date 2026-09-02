// =============================================================================
// 文件: internal/module/dept/handler.go
// 模块: 部门管理
// 类型: crud
// 职责: 处理部门管理页面请求并调用 Service 执行 CRUD。
// 依赖: internal/middleware
//       internal/model
//       internal/pkg/flash
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package dept

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// FlatItem 列表页扁平渲染项。
type FlatItem struct {
	ID        uint64
	ParentID  uint64
	Name      string
	Leader    string
	Status    uint8
	Sort      int
	CreatedAt time.Time
	Level     int
	Indent    string
}

// ParentOption 父部门下拉项。
type ParentOption struct {
	ID     uint64
	Title  string
	Status uint8
}

// DeptFormView 编辑页回显数据。
type DeptFormView struct {
	DeptName string
	ParentID uint64
	Status   uint8
	Sort     int
	Leader   string
	Phone    string
	Email    string
}

// Handler 处理部门管理页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建部门模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册部门模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/depts")
	g.Use(middleware.ActiveNav("/admin/depts"))
	{
		g.GET("", middleware.RequirePerm(perm.DeptList), h.List)
		g.GET("/new", middleware.RequirePerm(perm.DeptCreate), h.NewForm)
		g.POST("", middleware.RequirePerm(perm.DeptCreate), h.Create)
		g.GET("/:id/edit", middleware.RequirePerm(perm.DeptEdit), h.EditForm)
		g.POST("/:id", middleware.RequirePerm(perm.DeptEdit), h.Update)
		g.PUT("/:id", middleware.RequirePerm(perm.DeptEdit), h.Update)
		g.DELETE("/:id", middleware.RequirePerm(perm.DeptDelete), h.Delete)
		g.PUT("/:id/status", middleware.RequirePerm(perm.DeptEdit), h.UpdateStatus)
	}
}

// List 渲染部门树形列表（扁平化展示）。
func (h *Handler) List(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		h.logger.Error("list dept failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "获取部门列表失败", err)
		return
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_DEPT_LIST, gin.H{
		"Title":     "部门管理",
		"PageTitle": "部门管理",
		"Items":     flatten(resp.Items, 0),
	})
}

// NewForm 渲染新增部门页。
func (h *Handler) NewForm(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	resp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取部门列表失败", err)
		return
	}
	form := &CreateReq{}
	if parentID, ok := parseID(c.Query("parentId")); ok {
		form.ParentID = parentID
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_DEPT_CREATE, gin.H{
		"Title":         "新增部门",
		"PageTitle":     "新增部门",
		"Form":          form,
		"ParentOptions": toParentOptions(flatten(resp.Items, 0), nil),
		"Errors":        []FieldError{},
	})
}

// Create 处理新增部门提交。
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBind(&req); err != nil {
		if expectsJSON(c) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "参数解析失败",
			})
			return
		}
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		if expectsJSON(c) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "参数校验失败",
				"errors":  errs,
			})
			return
		}
		h.renderCreateForm(c, &req, errs)
		return
	}
	actor := middleware.CurrentUser(c)
	createResp, err := h.svc.Create(c.Request.Context(), actor, req)
	if err != nil {
		if expectsJSON(c) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": fmt.Sprintf("创建部门失败: %s", err.Error()),
			})
			return
		}
		flash.Error(c, "创建部门失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/depts/new")
		return
	}
	if expectsJSON(c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "部门创建成功",
			"id":      createResp.ID,
		})
		return
	}
	flash.Success(c, "部门创建成功")
	c.Redirect(http.StatusSeeOther, "/admin/depts")
}

// EditForm 渲染编辑部门页。
func (h *Handler) EditForm(c *gin.Context) {
	h.bindRenderer(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的部门 ID", nil)
		return
	}
	actor := middleware.CurrentUser(c)
	resource, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		render.Error(c, http.StatusNotFound, "部门不存在", err)
		return
	}
	resp, err := h.svc.List(c.Request.Context(), actor, ListReq{})
	if err != nil {
		render.Error(c, http.StatusInternalServerError, "获取部门列表失败", err)
		return
	}
	excluded := descendantsSet(resp.Items, id)
	excluded[id] = true
	render.Page(c, http.StatusOK, constants.TEMPLATE_DEPT_EDIT, gin.H{
		"Title":         "编辑部门",
		"PageTitle":     "编辑部门",
		"Form":          NewUpdateReqFromDept(resource),
		"Dept":          buildDeptFormView(NewUpdateReqFromDept(resource), resource),
		"Resource":      resource,
		"ParentOptions": toParentOptions(flatten(resp.Items, 0), excluded),
		"Errors":        []FieldError{},
	})
}

// Update 处理编辑部门提交。
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		render.Error(c, http.StatusBadRequest, "无效的部门 ID", nil)
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
		flash.Error(c, "更新部门失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/depts/"+strconv.FormatUint(id, 10)+"/edit")
		return
	}
	flash.Success(c, "部门更新成功")
	c.Redirect(http.StatusSeeOther, "/admin/depts")
}

// Delete 删除部门。
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		if expectsJSON(c) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "删除失败: 部门 ID 不合法",
			})
			return
		}
		flash.Error(c, "删除失败: 部门 ID 不合法")
		c.Redirect(http.StatusSeeOther, "/admin/depts")
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.Delete(c.Request.Context(), actor, DeleteReq{ID: id}); err != nil {
		if expectsJSON(c) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": fmt.Sprintf("删除失败: %s", err.Error()),
			})
			return
		}
		flash.Error(c, "删除失败: "+err.Error())
		c.Redirect(http.StatusSeeOther, "/admin/depts")
		return
	}
	if expectsJSON(c) {
		c.JSON(http.StatusOK, gin.H{
			"message": "部门已删除",
		})
		return
	}
	flash.Success(c, "部门已删除")
	c.Redirect(http.StatusSeeOther, "/admin/depts")
}

// UpdateStatus 处理部门状态快速切换。
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "更新失败: 部门 ID 不合法",
		})
		return
	}
	var req UpdateStatusReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数解析失败",
		})
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.UpdateStatus(c.Request.Context(), actor, req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": fmt.Sprintf("更新失败: %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "修改成功",
	})
}

func (h *Handler) renderCreateForm(c *gin.Context, req *CreateReq, errs []FieldError) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	resp, _ := h.svc.List(c.Request.Context(), actor, ListReq{})
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_DEPT_CREATE, gin.H{
		"Title":         "新增部门",
		"PageTitle":     "新增部门",
		"Form":          req,
		"ParentOptions": toParentOptions(flatten(resp.Items, 0), nil),
		"Errors":        errs,
	})
}

func (h *Handler) renderEditForm(c *gin.Context, req *UpdateReq, resource *model.Dept, errs []FieldError) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)
	resp, _ := h.svc.List(c.Request.Context(), actor, ListReq{})
	excluded := descendantsSet(resp.Items, req.ID)
	excluded[req.ID] = true
	render.Page(c, http.StatusUnprocessableEntity, constants.TEMPLATE_DEPT_EDIT, gin.H{
		"Title":         "编辑部门",
		"PageTitle":     "编辑部门",
		"Form":          req,
		"Dept":          buildDeptFormView(req, resource),
		"Resource":      resource,
		"ParentOptions": toParentOptions(flatten(resp.Items, 0), excluded),
		"Errors":        errs,
	})
}

func buildDeptFormView(req *UpdateReq, resource *model.Dept) DeptFormView {
	view := DeptFormView{}
	if req != nil {
		view.DeptName = req.Name
		view.ParentID = req.ParentID
		view.Status = req.Status
		view.Sort = req.Sort
		view.Leader = req.Leader
		view.Phone = req.Phone
		view.Email = req.Email
	}
	if resource != nil && req == nil {
		view.DeptName = resource.Name
		view.ParentID = resource.ParentID
		view.Status = resource.Status
		view.Sort = resource.Sort
		view.Leader = resource.Leader
		view.Phone = resource.Phone
		view.Email = resource.Email
	}
	return view
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

func expectsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	requestedWith := strings.ToLower(c.GetHeader("X-Requested-With"))
	return strings.Contains(accept, "application/json") || requestedWith == "xmlhttprequest"
}

func flatten(nodes []*DeptNode, level int) []FlatItem {
	items := make([]FlatItem, 0)
	for _, node := range nodes {
		if node == nil || node.Dept == nil {
			continue
		}
		items = append(items, FlatItem{
			ID:        node.Dept.ID,
			ParentID:  node.Dept.ParentID,
			Name:      node.Dept.Name,
			Leader:    node.Dept.Leader,
			Status:    node.Dept.Status,
			Sort:      node.Dept.Sort,
			CreatedAt: node.Dept.CreatedAt,
			Level:     level,
			Indent:    strings.Repeat("— ", level),
		})
		items = append(items, flatten(node.Children, level+1)...)
	}
	return items
}

func toParentOptions(items []FlatItem, excluded map[uint64]bool) []ParentOption {
	out := make([]ParentOption, 0, len(items)+1)
	out = append(out, ParentOption{ID: 0, Title: "顶级部门"})
	for _, item := range items {
		if excluded != nil && excluded[item.ID] {
			continue
		}
		out = append(out, ParentOption{
			ID:     item.ID,
			Title:  strings.Repeat("— ", item.Level) + item.Name,
			Status: item.Status,
		})
	}
	return out
}

func descendantsSet(nodes []*DeptNode, id uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	var walk func(list []*DeptNode) bool
	var markAll func(node *DeptNode)
	markAll = func(node *DeptNode) {
		for _, child := range node.Children {
			if child == nil || child.Dept == nil {
				continue
			}
			out[child.Dept.ID] = true
			markAll(child)
		}
	}
	walk = func(list []*DeptNode) bool {
		for _, node := range list {
			if node == nil || node.Dept == nil {
				continue
			}
			if node.Dept.ID == id {
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
