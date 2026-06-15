// =============================================================================
// 文件: internal/module/schedule/handler.go
// 模块: 排期工作台
// 类型: action
// 职责: 处理排期工作台页面请求并调用 Service。
// 依赖: internal/middleware
//       internal/pkg/pagination
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package schedule

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

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
	CapacityHours    int    `json:"capacityHours"`
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
		CapacityHours:    item.CapacityHours,
		CanEdit:          item.CanEdit,
		CanDelete:        item.CanDelete,
		HasLinkedDemands: item.HasLinkedDemands,
	}
}

// IndependentChildRequirement 独立研发需求子行（树形二级）。
type IndependentChildRequirement struct {
	ID              string
	Title           string
	Priority        string
	PriClass        string
	ProductName     string
	Stage           string
	StageClass      string
	WindowName      string
	TeamgroupName   string
	Owner           string
	TaskCount       int
	DetailURL       template.URL
	ScheduleURL     template.URL
	MaintainTaskURL template.URL
}

// IndependentRequirement 独立研发需求行（树形一级）。
type IndependentRequirement struct {
	ID              string
	Title           string
	Priority        string
	PriClass        string
	ProductName     string
	Stage           string
	StageClass      string
	WindowName      string
	TeamgroupName   string
	Owner           string
	TaskCount       int
	HasChildren     bool
	DetailURL       template.URL
	Children        []IndependentChildRequirement
}

// DevRequirement 研发需求行（树形三级）。
type DevRequirement struct {
	ID          string
	Title       string
	Priority    string
	PriClass    string
	IsMain      bool
	Owner       string
	TaskCount   int
	ActionLabel string
	ActionClass string
	DetailURL   template.URL
}

// SubBizRequirement 子业务需求行（树形二级）。
type SubBizRequirement struct {
	ID              string
	Title           string
	Priority        string
	PriClass        string
	Owner           string
	ActionLabel     string
	ActionClass     string
	DetailURL       template.URL
	DevRequirements []DevRequirement
}

// BizRequirement 业务需求行（树形一级）。
type BizRequirement struct {
	ID                 string
	Title              string
	Priority           string
	PriClass           string
	WindowStatus       string
	WindowStatusClass  string
	AgileGroup         string
	StageTag           string
	StageTagClass      string
	VersionWindow      string
	Owner              string
	ActionLabel        string
	ActionClass        string
	DetailURL          template.URL
	HasChildren        bool
	SubBizRequirements []SubBizRequirement
	DevRequirements    []DevRequirement
}

const scheduleRedirectURL = "/schedule"
const scheduleListPageSize = 10

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

// RegisterRoutes 注册排期工作台路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/schedule")
	g.Use(middleware.ActiveNav("/schedule"))
	{
		g.GET("", middleware.RequirePerm(perm.ScheduleList), h.Index)
		g.GET("/matching-plans", middleware.RequirePerm(perm.ScheduleList), h.GetMatchingPlans)
		g.POST("/windows", middleware.RequirePerm(perm.ScheduleCreate), h.CreateWindow)
		g.GET("/windows", middleware.RequirePerm(perm.ScheduleList), h.ListWindows)
		g.GET("/windows/:id", middleware.RequirePerm(perm.ScheduleList), h.GetWindow)
		g.PUT("/windows/:id", middleware.RequirePerm(perm.ScheduleUpdate), h.UpdateWindow)
		g.DELETE("/windows/:id", middleware.RequirePerm(perm.ScheduleDelete), h.DeleteWindow)
	}
}

// Index 渲染排期工作台页面。
func (h *Handler) Index(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)

	bizPage := parseSchedulePage(c.DefaultQuery("bizPage", "1"))
	indepPage := parseSchedulePage(c.DefaultQuery("indepPage", "1"))

	var listReq ListBizDemandsReq
	if err := c.ShouldBindQuery(&listReq); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return
	}
	listReq.Page = bizPage
	listReq.PageSize = scheduleListPageSize
	listReq.Normalize()

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

	bizResp, err := h.svc.ListBizDemands(c.Request.Context(), actor, listReq)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load biz demands failed", zap.Error(err))
		}
		bizResp = &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}
	}
	bizRequirements := toBizRequirementsView(bizResp.Items, h.zentaoURL)
	if h.logger != nil && len(bizRequirements) > 0 {
		h.logger.Debug("schedule biz demand detail url sample",
			zap.String("detailURL", string(bizRequirements[0].DetailURL)),
			zap.String("zentaoURL", h.zentaoURL),
		)
	}

	indepReq := ListIndependentReq{
		Page:     indepPage,
		PageSize: scheduleListPageSize,
	}
	indepReq.Normalize()

	if h.logger != nil {
		h.logger.Debug("independent stories query start",
			zap.String("account", actorAccount(actor)),
		)
	}

	indepResp, err := h.svc.ListIndependentStories(c.Request.Context(), actor, indepReq)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load independent stories failed", zap.Error(err))
		}
		indepResp = &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}
	}
	if indepResp == nil {
		indepResp = &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}
	}
	if h.logger != nil {
		h.logger.Debug("independent stories query done",
			zap.Int64("total", indepResp.Total),
			zap.Int("items", len(indepResp.Items)),
			zap.Error(err),
		)
	}
	independentRequirements := toIndependentRequirementsView(indepResp.Items, h.zentaoURL)

	bizPager := pagination.New(bizResp.Total, bizPage, scheduleListPageSize)
	bizPager.PageParam = "bizPage"
	bizPager.PreserveParams = map[string]string{"indepPage": strconv.Itoa(indepPage)}

	indepPager := pagination.New(indepResp.Total, indepPage, scheduleListPageSize)
	indepPager.PageParam = "indepPage"
	indepPager.PreserveParams = map[string]string{
		"bizPage": strconv.Itoa(bizPage),
		"tab":     "indep",
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_SCHEDULE_INDEX, gin.H{
		"Title":                   "排期工作台",
		"PageTitle":               "排期工作台",
		"Windows":                 windows,
		"BizRequirements":         bizRequirements,
		"BizTotal":                bizResp.Total,
		"IndependentRequirements": independentRequirements,
		"IndependentTotal":        indepResp.Total,
		"Teamgroups":              formData.Teamgroups,
		"Products":                formData.Products,
		"BizPager":                bizPager,
		"IndepPager":              indepPager,
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
