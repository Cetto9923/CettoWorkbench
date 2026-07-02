// =============================================================================
// 文件: internal/module/schedule/handler.go
// 模块: 排期工作台
// 类型: action
// 职责: 处理排期工作台页面请求并调用 Service。
// 依赖: internal/middleware
//       internal/pkg/render
//       internal/module/schedule/handler_demand.go
// =============================================================================

package schedule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

const scheduleRedirectURL = "/schedule"
const scheduleListPageSize = 10

// WindowCard 版本窗口概览卡片展示数据。
type WindowCard struct {
	ID               uint64
	ShortName        string
	Range            string
	ToneClass        string
	AgileGroup       string
	DemandCount      int
	CapacityHours    int
	UsedHours        int
	RemainingHours   int
	BlockedCount     int
	UsedPercent      int
	CanEdit          bool
	CanDelete        bool
	HasLinkedDemands bool
}

// WindowListItem 版本窗口维护列表项（JSON）。
type windowListItemJSON struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	ReleaseDate      string `json:"releaseDate"`
	Range            string `json:"range"`
	DemandCount      int    `json:"demandCount"`
	CapacityHours    int    `json:"capacityHours"`
	UsedHours        int    `json:"usedHours"`
	RemainingHours   int    `json:"remainingHours"`
	BlockedCount     int    `json:"blockedCount"`
	UsedPercent      int    `json:"usedPercent"`
	CanEdit          bool   `json:"canEdit"`
	CanDelete        bool   `json:"canDelete"`
	HasLinkedDemands bool   `json:"hasLinkedDemands"`
}

func toWindowListItemJSON(item WindowListItem) windowListItemJSON {
	return windowListItemJSON{
		ID:               item.ID,
		Name:             item.Name,
		ReleaseDate:      item.ReleaseDate,
		Range:            item.Range,
		DemandCount:      item.DemandCount,
		CapacityHours:    item.CapacityHours,
		UsedHours:        item.UsedHours,
		RemainingHours:   item.RemainingHours,
		BlockedCount:     item.BlockedCount,
		UsedPercent:      item.UsedPercent,
		CanEdit:          item.CanEdit,
		CanDelete:        item.CanDelete,
		HasLinkedDemands: item.HasLinkedDemands,
	}
}

// Handler 处理排期工作台页面请求。
type Handler struct {
	renderer  *render.Renderer
	logger    *zap.Logger
	svc       *Service
	zentaoURL string
}

// NewHandler 创建排期模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service, zentaoURL string) *Handler {
	return &Handler{
		renderer:  renderer,
		logger:    logger,
		svc:       svc,
		zentaoURL: strings.TrimRight(strings.TrimSpace(zentaoURL), "/"),
	}
}

// RegisterRoutes 注册排期工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/schedule")
	g.Use(middleware.ActiveNav("/schedule"))
	{
		g.GET("", h.Index)
		g.GET("/matching-plans", h.GetMatchingPlans)
		g.GET("/demands/:id/scheduling", h.GetDemandScheduling)
		g.POST("/demands/:id/save-scheduling", h.SaveScheduling)
		g.GET("/stories/:id/scheduling", h.GetStoryScheduling)
		g.POST("/stories/:id/save-scheduling", h.SaveStoryScheduling)
		g.GET("/products/:id/projects", h.GetProductProjects)
		g.GET("/projects/:id/executions", h.GetProjectExecutions)
		g.GET("/stories/:id/tasks", h.GetStoryTasks)
		g.POST("/stories/:id/save-tasks", h.SaveStoryTasks)
		g.POST("/windows", h.CreateWindow)
		g.GET("/windows", h.ListWindows)
		g.GET("/windows/:id", h.GetWindow)
		g.PUT("/windows/:id", h.UpdateWindow)
		g.DELETE("/windows/:id", h.DeleteWindow)
	}
}

// Index 渲染排期工作台页面。
func (h *Handler) Index(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)

	bizPage := parseSchedulePage(c.DefaultQuery("bizPage", "1"))
	indepPage := parseSchedulePage(c.DefaultQuery("indepPage", "1"))

	demandData, ok := h.loadScheduleIndexDemandData(c, actor, bizPage, indepPage)
	if !ok {
		return
	}

	formData, err := h.svc.GetCreateWindowFormData(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load create window form data failed", zap.Error(err))
		}
		formData = &CreateWindowFormData{
			Teamgroups: []TeamgroupOption{},
			Products:   []ZtProduct{},
		}
	}

	windows, err := h.svc.ListWindowCards(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load version windows failed", zap.Error(err))
		}
		windows = []WindowCard{}
	}

	filterProducts, err := h.svc.ListFilterProducts(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter products failed", zap.Error(err))
		}
		filterProducts = []ZtProduct{}
	}

	filterWindows, err := h.svc.ListFilterWindows(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter windows failed", zap.Error(err))
		}
		filterWindows = []WindowFilterOption{}
	}

	filterUsers, err := h.svc.ListScheduleUsers(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter users failed", zap.Error(err))
		}
		filterUsers = []SchedulingUserOption{}
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_SCHEDULE_INDEX, gin.H{
		"Title":                   "排期工作台",
		"PageTitle":               "排期工作台",
		"Windows":                 windows,
		"BizRequirements":         demandData.BizRequirements,
		"BizTotal":                demandData.BizTotal,
		"IndependentRequirements": demandData.IndependentRequirements,
		"IndependentTotal":        demandData.IndependentTotal,
		"Teamgroups":              formData.Teamgroups,
		"Products":                formData.Products,
		"FilterProducts":          filterProducts,
		"FilterWindows":           filterWindows,
		"FilterUsers":             filterUsers,
		"StageFilterOptions":      ScheduleStageFilterOptionsForTab(demandData.ActiveTab),
		"WindowTypeFilterOptions": ScheduleWindowTypeFilterOptions,
		"SelectedGroups":          demandData.SelectedGroups,
		"SelectedProducts":        demandData.SelectedProducts,
		"SelectedStages":          demandData.SelectedStages,
		"SelectedWindows":         demandData.SelectedWindows,
		"SelectedKeyword":         demandData.SelectedKeyword,
		"SelectedPri":             demandData.SelectedPri,
		"SelectedWindowType":      demandData.SelectedWindowType,
		"SelectedDevOwner":        demandData.SelectedDevOwner,
		"SelectedTestOwner":       demandData.SelectedTestOwner,
		"SelectedAcceptOwner":     demandData.SelectedAcceptOwner,
		"SelectedGroupMap":        demandData.SelectedGroupMap,
		"SelectedProductMap":      demandData.SelectedProductMap,
		"SelectedStageMap":        demandData.SelectedStageMap,
		"SelectedWindowMap":       demandData.SelectedWindowMap,
		"BizPager":                demandData.BizPager,
		"IndepPager":              demandData.IndepPager,
		"ActiveFilter":            demandData.ActiveFilter,
		"SuspendedActive":         demandData.SuspendedActive,
		"SuspendedCount":          demandData.SuspendedCount,
		"BizFilterCounts":         demandData.BizFilterCounts,
		"IndepFilterCounts":       demandData.IndepFilterCounts,
	})
}

// ListWindows 返回版本窗口维护列表（JSON）。
func (h *Handler) ListWindows(c *gin.Context) {
	actor := middleware.CurrentUser(c)

	resp, err := h.svc.ListWindows(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("list version windows failed", zap.Error(err))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载版本窗口列表失败",
		})
		return
	}

	windows := make([]windowListItemJSON, 0, len(resp.Windows))
	for _, item := range resp.Windows {
		windows = append(windows, toWindowListItemJSON(item))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"windows": windows,
	})
}

// CreateWindow 保存新建版本窗口（JSON）。
func (h *Handler) CreateWindow(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  errs,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	if err := h.svc.Create(c.Request.Context(), actor, req); err != nil {
		if h.logger != nil {
			h.logger.Error("save version window failed",
				zap.Error(err),
				zap.String("name", strings.TrimSpace(req.Name)),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "保存版本窗口失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口保存成功",
		"redirectUrl": scheduleRedirectURL,
	})
}

// GetWindow 获取版本窗口详情（JSON）。
func (h *Handler) GetWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	detail, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get version window detail failed", zap.Error(err), zap.Uint64("window_id", id))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	resp := gin.H{"success": true}
	if detail != nil {
		resp["id"] = detail.ID
		resp["releaseDate"] = detail.ReleaseDate
		resp["name"] = detail.Name
		resp["startDate"] = detail.StartDate
		resp["teamgroupId"] = detail.TeamgroupID
		resp["groupSize"] = detail.GroupSize
		resp["products"] = detail.Products
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateWindow 更新版本窗口（JSON）。
func (h *Handler) UpdateWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  errs,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	if err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		if h.logger != nil {
			h.logger.Error("update version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "更新版本窗口失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口更新成功",
		"redirectUrl": scheduleRedirectURL,
	})
}

// DeleteWindow 软删除版本窗口（JSON）。
func (h *Handler) DeleteWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	deleteReq := DeleteReq{ID: id}
	if errs := deleteReq.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   formatFieldErrors(errs),
		})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, deleteReq); err != nil {
		if h.logger != nil {
			h.logger.Error("delete version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口已删除",
		"redirectUrl": scheduleRedirectURL,
	})
}

// GetMatchingPlans 根据产品 ID 和结束日期返回匹配计划（JSON）。
func (h *Handler) GetMatchingPlans(c *gin.Context) {
	var req MatchingPlansReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数解析失败",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.GetMatchingPlans(c.Request.Context(), actor, req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get matching plans failed",
				zap.Error(err),
				zap.Uint("product_id", req.ProductID),
				zap.String("end_date", req.EndDate),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "查询匹配计划失败",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func parseSchedulePage(raw string) int {
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}

func parseWindowID(c *gin.Context) (uint64, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func formatFieldErrors(errs []FieldError) string {
	if len(errs) == 0 {
		return "参数校验失败"
	}
	messages := make([]string, 0, len(errs))
	for _, item := range errs {
		msg := strings.TrimSpace(item.Message)
		if msg != "" {
			messages = append(messages, msg)
		}
	}
	if len(messages) == 0 {
		return "参数校验失败"
	}
	return strings.Join(messages, "；")
}
