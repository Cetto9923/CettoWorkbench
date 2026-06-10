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
	"log"
	"net/http"
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
	ShortName      string
	Range          string
	Status         string
	ToneClass      string
	AgileGroup     string
	DemandCount    int
	CapacityHours  int
	UsedHours      int
	RemainingHours int
	BlockedCount   int
	UsedPercent    int
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

	windows := []WindowCard{
		{
			ShortName: "26-0524窗口", Range: "05-11 ~ 05-24", Status: "当前", ToneClass: "red",
			AgileGroup: "移动银行组", DemandCount: 10, CapacityHours: 55, UsedHours: 42,
			RemainingHours: 13, BlockedCount: 2, UsedPercent: 76,
		},
		{
			ShortName: "26-0607窗口", Range: "05-25 ~ 06-07", Status: "下一", ToneClass: "blue",
			AgileGroup: "基础平台组", DemandCount: 7, CapacityHours: 48, UsedHours: 28,
			RemainingHours: 20, BlockedCount: 1, UsedPercent: 58,
		},
		{
			ShortName: "26-0621窗口", Range: "06-08 ~ 06-21", Status: "规划中", ToneClass: "green",
			AgileGroup: "客户服务组", DemandCount: 4, CapacityHours: 48, UsedHours: 12,
			RemainingHours: 36, BlockedCount: 0, UsedPercent: 25,
		},
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

	// TODO: 临时调试日志，确认匹配参数后删除
	log.Printf("[DEBUG] matching-plans 请求: product_id=%d, end_date=%s", req.ProductID, req.EndDate)

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
