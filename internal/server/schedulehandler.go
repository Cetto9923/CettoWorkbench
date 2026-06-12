// =============================================================================
// 文件: internal/server/schedulehandler.go
// 模块: 基础设施
// 类型: infra
// 职责: 排期工作台页面 Handler（本轮硬编码假数据）。
// 依赖: internal/pkg/pagination, internal/pkg/render
// =============================================================================

package server

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"workbench/internal/constants"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/render"
)

// ScheduleWindowCard 版本窗口概览卡片展示数据。
type ScheduleWindowCard struct {
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

// ScheduleDevRequirement 研发需求行（树形三级）。
type ScheduleDevRequirement struct {
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

// ScheduleSubBizRequirement 子业务需求行（树形二级）。
type ScheduleSubBizRequirement struct {
	ID              string
	Title           string
	Priority        string
	PriClass        string
	Owner           string
	OwnerInitial    string
	DevRequirements []ScheduleDevRequirement
}

// ScheduleBizRequirement 业务需求行（树形一级）。
type ScheduleBizRequirement struct {
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
	SubBizRequirements []ScheduleSubBizRequirement
	DevRequirements    []ScheduleDevRequirement
}

// ScheduleHandler 渲染排期工作台页面。
func ScheduleHandler(c *gin.Context) {
	windows := []ScheduleWindowCard{
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

	bizRequirements := buildScheduleBizRequirements()

	render.Page(c, http.StatusOK, constants.TEMPLATE_SCHEDULE_INDEX, gin.H{
		"Title":           "排期工作台",
		"PageTitle":       "排期工作台",
		"Windows":         windows,
		"BizRequirements": bizRequirements,
		"Pager":           pagination.New(int64(len(bizRequirements)), 1, 10),
	})
}

func buildScheduleBizRequirements() []ScheduleBizRequirement {
	return []ScheduleBizRequirement{
		buildScheduleBizReq4001(),
		buildScheduleBizReq4002(),
		buildScheduleBizReq4004(),
		buildScheduleBizReq4005(),
		buildScheduleBizReq4007(),
		buildScheduleBizReq4008(),
		buildScheduleBizReq4009(),
		buildScheduleBizReq4010(),
		buildScheduleBizReq4012(),
		buildScheduleBizReq4030(),
	}
}

func buildScheduleBizReq4001() ScheduleBizRequirement {
	biz := newScheduleBizBase("REQ-4001", "企业网银大额转账审批流程优化", "agile1", "current", "P0", "initial", true, false, "张明远")
	biz.SubBizRequirements = []ScheduleSubBizRequirement{
		{
			ID: "SUB-4001-01", Title: "大额转账审批规则优化", Priority: "P0", PriClass: "p0",
			Owner: "张明远", OwnerInitial: ownerInitial("张明远"),
		},
	}
	biz.HasChildren = true
	return biz
}

func buildScheduleBizReq4002() ScheduleBizRequirement {
	return newScheduleBizBase("REQ-4002", "移动端人脸识别登录功能", "agile1", "current", "P0", "final", false, false, "陈工")
}

func buildScheduleBizReq4004() ScheduleBizRequirement {
	biz := newScheduleBizBase("REQ-4004", "对账文件格式升级改造", "agile2", "current", "P1", "initial", false, true, "王工")
	biz.DevRequirements = []ScheduleDevRequirement{
		newScheduleDevReq("RD-4004-1", "对账文件格式升级改造-主流程改造", "P1", true, "王工", 0),
		newScheduleDevReq("RD-4004-2", "对账文件格式升级改造-联调适配", "P1", false, "李工", 0),
		newScheduleDevReq("RD-4004-3", "对账文件格式升级改造-批处理优化", "P1", true, "王工", 0),
	}
	biz.HasChildren = true
	return biz
}

func buildScheduleBizReq4005() ScheduleBizRequirement {
	return newScheduleBizBase("REQ-4005", "统一认证微信扫码登录", "agile2", "next", "P1", "initial", false, false, "李工")
}

func buildScheduleBizReq4007() ScheduleBizRequirement {
	return newScheduleBizBase("REQ-4007", "跨行转账手续费优化", "agile3", "current", "P1", "final", false, true, "孙工")
}

func buildScheduleBizReq4008() ScheduleBizRequirement {
	return newScheduleBizBase("REQ-4008", "开放银行API接口升级", "agile2", "next", "P1", "final", false, false, "周工")
}

func buildScheduleBizReq4009() ScheduleBizRequirement {
	biz := newScheduleBizBase("REQ-4009", "外汇兑换汇率实时展示", "agile1", "future", "P2", "initial", false, false, "待分配")
	biz.WindowStatus = "未排期"
	biz.WindowStatusClass = ""
	biz.VersionWindow = "—"
	return biz
}

func buildScheduleBizReq4010() ScheduleBizRequirement {
	biz := newScheduleBizBase("REQ-4010", "历史遗留接口兼容适配", "agile3", "history", "P2", "initial", false, false, "张明远")
	biz.WindowStatus = "未排期"
	biz.WindowStatusClass = ""
	biz.VersionWindow = "历史窗口"
	return biz
}

func buildScheduleBizReq4012() ScheduleBizRequirement {
	return newScheduleBizBase("REQ-4012", "贷款利率参数配置", "agile3", "next", "P2", "final", false, false, "刘工")
}

func buildScheduleBizReq4030() ScheduleBizRequirement {
	biz := newScheduleBizBase("REQ-4030", "核心系统联机交易性能优化", "agile1", "current", "P0", "final", false, false, "李工")
	biz.DevRequirements = []ScheduleDevRequirement{
		newScheduleDevReq("RD-4030-1", "核心系统联机交易性能优化-交易流水改造", "P0", true, "李工", 3),
	}
	biz.HasChildren = true
	return biz
}

func newScheduleBizBase(id, title, agile, plan, pri, stage string, blocked, overdue bool, owner string) ScheduleBizRequirement {
	windowStatus, windowStatusClass := scheduleWindowStatus(blocked, overdue, plan)
	stageTag, stageTagClass := scheduleStageTag(stage)
	rowClass := ""
	if overdue {
		rowClass = "overdue-row"
	}
	return ScheduleBizRequirement{
		ID: id, Title: title, Priority: pri, PriClass: strings.ToLower(pri),
		WindowStatus: windowStatus, WindowStatusClass: windowStatusClass,
		AgileGroup: scheduleAgileLabel(agile), StageTag: stageTag, StageTagClass: stageTagClass,
		VersionWindow: scheduleVersionWindow(plan), Owner: owner, OwnerInitial: ownerInitial(owner),
		Blocked: blocked, Overdue: overdue, RowClass: rowClass,
	}
}

func newScheduleDevReq(id, title, pri string, isMain bool, owner string, taskCount int) ScheduleDevRequirement {
	actionLabel := "拆任务"
	actionClass := "primary"
	if taskCount > 0 {
		actionLabel = "维护任务"
		actionClass = "success"
	}
	return ScheduleDevRequirement{
		ID: id, Title: title, Priority: pri, PriClass: strings.ToLower(pri), IsMain: isMain,
		Owner: owner, OwnerInitial: ownerInitial(owner),
		TaskCount: taskCount, HasTasks: taskCount > 0, ActionLabel: actionLabel, ActionClass: actionClass,
	}
}

func scheduleWindowStatus(blocked, overdue bool, plan string) (string, string) {
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

func scheduleStageTag(stage string) (string, string) {
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

func scheduleAgileLabel(agile string) string {
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

func scheduleVersionWindow(plan string) string {
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
