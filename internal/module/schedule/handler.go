// =============================================================================
// 文件: internal/module/schedule/handler.go
// 模块: 排期工作台
// 类型: action
// 职责: 处理排期工作台页面请求并调用 Service。
// 依赖: internal/middleware
//       internal/pkg/pagination
//       internal/pkg/render
// =============================================================================

package schedule

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/render"
)

// WindowCard 版本窗口概览卡片展示数据。
type WindowCard struct {
	ID               uint64
	ShortName        string
	Range            string
	Status           string
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
type WindowListItem struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	ReleaseDate      string `json:"releaseDate"`
	Range            string `json:"range"`
	Status           string `json:"status"`
	CapacityHours    int    `json:"capacityHours"`
	CanEdit          bool   `json:"canEdit"`
	CanDelete        bool   `json:"canDelete"`
	HasLinkedDemands bool   `json:"hasLinkedDemands"`
}

// DevRequirement 研发需求行（树形三级）。
type DevRequirement struct {
	ID           string
	Title        string
	Priority     string
	PriClass     string
	IsMain       bool
	Owner        string
	OwnerInitial string
	TaskCount    int
	HasTasks     bool
	ActionLabel  string
	ActionClass  string
}

// SubBizRequirement 子业务需求行（树形二级）。
type SubBizRequirement struct {
	ID              string
	Title           string
	Priority        string
	PriClass        string
	Owner           string
	OwnerInitial    string
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
	OwnerInitial       string
	Blocked            bool
	Overdue            bool
	RowClass           string
	HasChildren        bool
	SubBizRequirements []SubBizRequirement
	DevRequirements    []DevRequirement
}

const scheduleRedirectURL = "/po/schedule"

// Handler 处理排期工作台页面请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建排期模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Index 渲染排期工作台页面。
func (h *Handler) Index(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}

	formData, err := h.svc.GetCreateWindowFormData(c.Request.Context(), account)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load create window form data failed", zap.Error(err), zap.String("account", account))
		}
		formData = &CreateWindowFormData{
			Teamgroups:      []TeamgroupOption{},
			Products:        []ZtProduct{},
			WindowTemplates: []WindowTemplateItem{},
		}
	}

	windowTemplatesJSON := template.JS("[]")
	if len(formData.WindowTemplates) > 0 {
		if encoded, marshalErr := json.Marshal(formData.WindowTemplates); marshalErr == nil {
			windowTemplatesJSON = template.JS(encoded)
		}
	}

	windows, err := h.svc.ListWindowCards(c.Request.Context(), account)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load version windows failed", zap.Error(err))
		}
		windows = []WindowCard{}
	}

	bizRequirements := buildBizRequirements()

	render.Page(c, http.StatusOK, constants.TEMPLATE_SCHEDULE_INDEX, gin.H{
		"Title":           "排期工作台",
		"PageTitle":       "排期工作台",
		"Windows":         windows,
		"BizRequirements": bizRequirements,
		"Teamgroups":          formData.Teamgroups,
		"Products":            formData.Products,
		"WindowTemplatesJSON": windowTemplatesJSON,
		"Pager":               pagination.New(int64(len(bizRequirements)), 1, 10),
	})
}

// ListWindows 返回版本窗口维护列表（JSON）。
func (h *Handler) ListWindows(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}

	items, err := h.svc.ListWindows(c.Request.Context(), account)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("list version windows failed", zap.Error(err), zap.String("account", account))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载版本窗口列表失败",
		})
		return
	}
	if items == nil {
		items = []WindowListItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"windows": items,
	})
}

// CreateWindow 保存新建版本窗口（JSON）。
func (h *Handler) CreateWindow(c *gin.Context) {
	var form CreateWindowForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	if errs := form.Validate(); len(errs) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}
	if strings.TrimSpace(account) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	window, err := BuildVersionWindowFromForm(form)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "窗口日期无效",
		})
		return
	}

	if err := h.svc.SaveWindowWithPlans(c.Request.Context(), window, form.Products, account); err != nil {
		if h.logger != nil {
			h.logger.Error("save version window failed",
				zap.Error(err),
				zap.String("account", account),
				zap.String("name", window.Name),
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

	detail, err := h.svc.GetWindowDetail(c.Request.Context(), id)
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

	var form CreateWindowForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	if errs := form.Validate(); len(errs) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	account := ""
	if actor != nil {
		account = actor.Account
	}
	if strings.TrimSpace(account) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	window, err := h.svc.GetVersionWindow(c.Request.Context(), id)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load version window for update failed", zap.Error(err), zap.Uint64("window_id", id))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载版本窗口失败",
		})
		return
	}
	if window == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "窗口不存在",
		})
		return
	}

	if err := ApplyFormToVersionWindow(window, form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "窗口日期无效",
		})
		return
	}

	if err := h.svc.UpdateWindowWithPlans(c.Request.Context(), window, form.Products, account); err != nil {
		if h.logger != nil {
			h.logger.Error("update version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
				zap.String("account", account),
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
	account := ""
	if actor != nil {
		account = actor.Account
	}
	if strings.TrimSpace(account) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	if err := h.svc.SoftDeleteWindow(c.Request.Context(), id, account); err != nil {
		if h.logger != nil {
			h.logger.Error("delete version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
				zap.String("account", account),
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

	resp, err := h.svc.GetMatchingPlans(c.Request.Context(), req)
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

