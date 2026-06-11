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
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

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
			Teamgroups: []TeamgroupOption{},
			Products:   []ZtProduct{},
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
		"Teamgroups":      formData.Teamgroups,
		"Products":        formData.Products,
		"Pager":           pagination.New(int64(len(bizRequirements)), 1, 10),
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
		"success":  true,
		"windowId": window.ID,
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
		"success":  true,
		"windowId": window.ID,
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
		"success": true,
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

func buildBizRequirements() []BizRequirement {
	return []BizRequirement{
		buildBizReq4001(),
		buildBizReq4002(),
		buildBizReq4004(),
		buildBizReq4005(),
		buildBizReq4007(),
		buildBizReq4008(),
		buildBizReq4009(),
		buildBizReq4010(),
		buildBizReq4012(),
		buildBizReq4030(),
	}
}

func buildBizReq4001() BizRequirement {
	biz := newBizBase("REQ-4001", "企业网银大额转账审批流程优化", "agile1", "current", "P0", "initial", true, false, "张明远")
	biz.SubBizRequirements = []SubBizRequirement{
		{
			ID: "SUB-4001-01", Title: "大额转账审批规则优化", Priority: "P0", PriClass: "p0",
			Owner: "张明远", OwnerInitial: ownerInitial("张明远"),
		},
	}
	biz.HasChildren = true
	return biz
}

func buildBizReq4002() BizRequirement {
	return newBizBase("REQ-4002", "移动端人脸识别登录功能", "agile1", "current", "P0", "final", false, false, "陈工")
}

func buildBizReq4004() BizRequirement {
	biz := newBizBase("REQ-4004", "对账文件格式升级改造", "agile2", "current", "P1", "initial", false, true, "王工")
	biz.DevRequirements = []DevRequirement{
		newDevReq("RD-4004-1", "对账文件格式升级改造-主流程改造", "P1", true, "王工", 0),
		newDevReq("RD-4004-2", "对账文件格式升级改造-联调适配", "P1", false, "李工", 0),
		newDevReq("RD-4004-3", "对账文件格式升级改造-批处理优化", "P1", true, "王工", 0),
	}
	biz.HasChildren = true
	return biz
}

func buildBizReq4005() BizRequirement {
	return newBizBase("REQ-4005", "统一认证微信扫码登录", "agile2", "next", "P1", "initial", false, false, "李工")
}

func buildBizReq4007() BizRequirement {
	return newBizBase("REQ-4007", "跨行转账手续费优化", "agile3", "current", "P1", "final", false, true, "孙工")
}

func buildBizReq4008() BizRequirement {
	return newBizBase("REQ-4008", "开放银行API接口升级", "agile2", "next", "P1", "final", false, false, "周工")
}

func buildBizReq4009() BizRequirement {
	biz := newBizBase("REQ-4009", "外汇兑换汇率实时展示", "agile1", "future", "P2", "initial", false, false, "待分配")
	biz.WindowStatus = "未排期"
	biz.WindowStatusClass = ""
	biz.VersionWindow = "—"
	return biz
}

func buildBizReq4010() BizRequirement {
	biz := newBizBase("REQ-4010", "历史遗留接口兼容适配", "agile3", "history", "P2", "initial", false, false, "张明远")
	biz.WindowStatus = "未排期"
	biz.WindowStatusClass = ""
	biz.VersionWindow = "历史窗口"
	return biz
}

func buildBizReq4012() BizRequirement {
	return newBizBase("REQ-4012", "贷款利率参数配置", "agile3", "next", "P2", "final", false, false, "刘工")
}

func buildBizReq4030() BizRequirement {
	biz := newBizBase("REQ-4030", "核心系统联机交易性能优化", "agile1", "current", "P0", "final", false, false, "李工")
	biz.DevRequirements = []DevRequirement{
		newDevReq("RD-4030-1", "核心系统联机交易性能优化-交易流水改造", "P0", true, "李工", 3),
	}
	biz.HasChildren = true
	return biz
}

func newBizBase(id, title, agile, plan, pri, stage string, blocked, overdue bool, owner string) BizRequirement {
	windowStatus, windowStatusClass := windowStatus(blocked, overdue, plan)
	stageTag, stageTagClass := stageTag(stage)
	rowClass := ""
	if overdue {
		rowClass = "overdue-row"
	}
	return BizRequirement{
		ID: id, Title: title, Priority: pri, PriClass: strings.ToLower(pri),
		WindowStatus: windowStatus, WindowStatusClass: windowStatusClass,
		AgileGroup: agileLabel(agile), StageTag: stageTag, StageTagClass: stageTagClass,
		VersionWindow: versionWindow(plan), Owner: owner, OwnerInitial: ownerInitial(owner),
		Blocked: blocked, Overdue: overdue, RowClass: rowClass,
	}
}

func newDevReq(id, title, pri string, isMain bool, owner string, taskCount int) DevRequirement {
	actionLabel := "拆任务"
	actionClass := "primary"
	if taskCount > 0 {
		actionLabel = "维护任务"
		actionClass = "success"
	}
	return DevRequirement{
		ID: id, Title: title, Priority: pri, PriClass: strings.ToLower(pri), IsMain: isMain,
		Owner: owner, OwnerInitial: ownerInitial(owner),
		TaskCount: taskCount, HasTasks: taskCount > 0, ActionLabel: actionLabel, ActionClass: actionClass,
	}
}

func windowStatus(blocked, overdue bool, plan string) (string, string) {
	if overdue {
		return "超期", "overdue"
	}
	if blocked {
		return "阻塞", "blocked"
	}
	if plan == "" || plan == "future" {
		return "未排期", ""
	}
	return "已排期", "planning"
}

func stageTag(stage string) (string, string) {
	if stage == "final" {
		return "终排", "stage-tag--final"
	}
	return "初排", "stage-tag--draft"
}

func ownerInitial(name string) string {
	n := strings.TrimSpace(name)
	if n == "" || n == "待分配" {
		return "?"
	}
	r, _ := utf8.DecodeRuneInString(n)
	return string(r)
}

func agileLabel(agile string) string {
	switch agile {
	case "agile1":
		return "移动银行组"
	case "agile2":
		return "基础平台组"
	case "agile3":
		return "客户服务组"
	default:
		if agile == "" {
			return "—"
		}
		return agile
	}
}

func versionWindow(plan string) string {
	switch plan {
	case "current":
		return "26-0524窗口"
	case "next":
		return "26-0607窗口"
	case "future", "next2":
		return "26-0621窗口"
	case "nextnext":
		return "26-0705窗口"
	case "recent", "recentRelease":
		return "26-0510已发"
	default:
		if strings.HasPrefix(plan, "history") {
			return "历史窗口"
		}
		if plan == "" {
			return "—"
		}
		return "未归属"
	}
}
