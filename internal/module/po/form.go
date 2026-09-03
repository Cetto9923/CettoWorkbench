// =============================================================================
// 文件: internal/module/po/form.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 Req/Resp 结构体。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"

	"workbench/internal/module/schedule"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValueStreamStage 价值流阶段卡片数据（对应 homeVsCompact 单个阶段）。
type ValueStreamStage struct {
	Label       string
	Status      string
	Count       int64
	DemandCount int64
	StoryCount  int64
}

// HomeResp PO 工作台首页数据。
type HomeResp struct {
	Stages         []ValueStreamStage
	VersionWindows []schedule.HomeVersionWindowCard
	KPI            KPICounts
}

// KPICounts 首页 5 个焦点摘要的真实计数。
// 4 个 KPI 由 repo CountKPI{...} 真实统计;MyPending 由 Service 计算。
// 字段为零时前端仍展示数字 0,不显示破折号。
type KPICounts struct {
	Today     int64 // 今日必推：今日到期 OR 已逾期 且未完成
	MyPending int64 // 待我处理：handlingResponsibility=currentUser（=价值流 all 计数）
	Blocked   int64 // 阻塞：主管部门审批存在拒绝 ∪ 验收阶段超期
	Overdue   int64 // 超期：today > deadline 且未完成（缺日期不算）
	Suspended int64 // 挂起：hang='1' 且未关闭
}

// DemandsReq 按价值流状态查询需求/故事详情。
type DemandsReq struct {
	Status string `form:"status"`
}

// Validate 校验查询参数。
func (r *DemandsReq) Validate() []FieldError {
	status := strings.TrimSpace(r.Status)
	if status == "" {
		return []FieldError{{Field: "status", Message: "状态不能为空"}}
	}
	if !isValidValueStreamStatus(status) {
		return []FieldError{{Field: "status", Message: "无效的价值流状态"}}
	}
	r.Status = status
	return nil
}

// WorkItemDetail 单条需求或故事详情。
type WorkItemDetail struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"` // 展示编号：业需 US{id}，研需 U{id}
	Pri          string `json:"pri"`
	Title        string `json:"title"`
	Stage        string `json:"stage"`
	Blocker      string `json:"blocker"`
	Next         string `json:"next"`
	Owner        string `json:"owner"`     // 与 NextOwner 同值，兼容旧字段
	NextOwner    string `json:"nextOwner"` // 下一责任人展示名（DeriveCurrentHandler）
	ZentaoUrl    string `json:"zentaoUrl"`
	ValueStream  string `json:"valueStream"`
	ZentaoStatus string `json:"zentaoStatus"` // 禅道 status 原文，前端按业需/研需分别映射中文
}

// DemandsResp 价值流状态下的需求详情列表。
type DemandsResp struct {
	Items []WorkItemDetail `json:"items"`
}

// TodoTab 我的待办对象域 Tab。V10.1 02 节：6 类对象域，本期先打通审批决策 + 需求治理。
// 其它 Tab（研发执行/测试质量/问题风险/个人事项）作为占位渲染，详细数据来源后续阶段接入。
type TodoTab string

const (
	TodoTabApproval  TodoTab = "approval" // 审批决策
	TodoTabDemand    TodoTab = "demand"   // 需求治理（业务需求 + 研发需求 + 反馈）
	TodoTabAll       TodoTab = "all"      // 全部（跨 Tab 汇总）
	TodoTabExecution TodoTab = "execution"
	TodoTabTesting   TodoTab = "testing"
	TodoTabRisk      TodoTab = "risk"
	TodoTabPersonal  TodoTab = "personal"
)

// Relation 我的关系 V10.1 02 节。
type Relation string

const (
	RelationAll       Relation = "all"
	RelationInCharge  Relation = "in_charge" // 我负责
	RelationCooperate Relation = "cooperate" // 我配合
	RelationFollow    Relation = "follow"    // 我关注
)

// Responsibility 办理责任 V10.1 02 节。
type Responsibility string

const (
	ResponsibilityAll        Responsibility = "all"
	ResponsibilityMyAction   Responsibility = "my_action"    // 待我处理
	ResponsibilityMyFollowUp Responsibility = "my_follow_up" // 待我跟进
)

// TodoListReq 我的待办列表请求。
// V10.1 02 节：7 维 AND = Tab ∩ 办理场景 ∩ 阶段 ∩ 对象 ∩ 我的关系 ∩ 办理责任 ∩ 关键词。
// 本期实现全部 7 维；具体对象 objectType 过滤只展示当前 Tab 实际有数据的对象类型。
type TodoListReq struct {
	Tab            TodoTab        `form:"tab"`            // 对象域 Tab；默认 all
	Action         TodoAction     `form:"action"`         // 办理场景；默认 all
	Stage          string         `form:"stage"`          // 阶段（仅 demand 生效）；默认 all
	ObjectType     string         `form:"objectType"`     // 对象类型 demand/story/task/bug/testtask；默认 all
	Relation       Relation       `form:"relation"`       // 我的关系；默认 all
	Responsibility Responsibility `form:"responsibility"` // 办理责任；默认 all
	Focus          string         `form:"focus"`          // 一级快捷筛选；默认 pending
	Keyword        string         `form:"keyword"`        // 关键词
	Page           int            `form:"page"`           // 页码；1-based
	PageSize       int            `form:"pageSize"`       // 每页条数；默认 20
}

// TodoAction 办理场景（V10.1 02 节 "为什么现在要办"）。
// 本期按需求 status 启发式映射；后续接入 zt_action action 字符串后精确化。
type TodoAction string

const (
	TodoActionAll      TodoAction = "all"
	TodoActionReview   TodoAction = "todo_review"   // 待受理/待评审: status IN (draft, wait, active, refuse)
	TodoActionSchedule TodoAction = "todo_schedule" // 待排期: status IN (clarified) + scheduleIncomplete
	TodoActionVerify   TodoAction = "todo_verify"   // 待验收: status IN (testing, waitacceptance)
	TodoActionDeliver  TodoAction = "todo_deliver"  // 待发起交付: status IN (acceptanced)，个人责任由基础集合约束
	TodoActionFollow   TodoAction = "todo_follow"   // 待跟进: 其它主动跟进
)

// Validate 校验 TodoListReq。
func (r *TodoListReq) Validate() []FieldError {
	r.Tab = TodoTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = TodoTabAll
	}
	switch r.Tab {
	case TodoTabApproval, TodoTabDemand, TodoTabAll, TodoTabExecution, TodoTabTesting, TodoTabRisk, TodoTabPersonal:
	default:
		return []FieldError{{Field: "tab", Message: "无效的对象域 Tab"}}
	}
	r.Action = TodoAction(strings.TrimSpace(string(r.Action)))
	if r.Action == "" {
		r.Action = TodoActionAll
	}
	switch r.Action {
	case TodoActionAll, TodoActionReview, TodoActionSchedule, TodoActionVerify, TodoActionDeliver, TodoActionFollow:
	default:
		return []FieldError{{Field: "action", Message: "无效的办理场景"}}
	}
	r.Stage = strings.TrimSpace(r.Stage)
	if r.Stage == "" {
		r.Stage = "all"
	}
	r.ObjectType = strings.TrimSpace(r.ObjectType)
	if r.ObjectType == "" {
		r.ObjectType = "all"
	}
	switch r.ObjectType {
	case "all", "approval", "demand", "story", "task", "bug", "testtask", "issue", "risk", "todo":
	default:
		return []FieldError{{Field: "objectType", Message: "无效的对象类型"}}
	}
	r.Relation = Relation(strings.TrimSpace(string(r.Relation)))
	if r.Relation == "" {
		r.Relation = RelationAll
	}
	switch r.Relation {
	case RelationAll, RelationInCharge, RelationCooperate, RelationFollow:
	default:
		return []FieldError{{Field: "relation", Message: "无效的我的关系"}}
	}
	r.Responsibility = Responsibility(strings.TrimSpace(string(r.Responsibility)))
	if r.Responsibility == "" {
		r.Responsibility = ResponsibilityAll
	}
	switch r.Responsibility {
	case ResponsibilityAll, ResponsibilityMyAction, ResponsibilityMyFollowUp:
	default:
		return []FieldError{{Field: "responsibility", Message: "无效的办理责任"}}
	}
	r.Focus = strings.TrimSpace(r.Focus)
	if r.Focus == "" {
		r.Focus = "pending"
	}
	switch r.Focus {
	case "pending", "today", "overdue", "blocked", "p1":
	default:
		return []FieldError{{Field: "focus", Message: "无效的一级筛选"}}
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	return nil
}

// TodoItem 我的待办单条（横跨业务需求/研发需求/任务/Bug/测试单等多种对象）。
// kind 决定展示与跳转链接生成。
type TodoItem struct {
	Kind           string `json:"kind"`           // demand / story / task / bug / test
	ID             int64  `json:"id"`             // 业务需求 ID（业需/任务/...各自主键）
	DisplayID      string `json:"displayId"`      // 展示编号：业需 US{id}，研需 U{id}，任务/单据 TASK-{id} 等
	Title          string `json:"title"`          // 标题
	Type           string `json:"type"`           // 对象类型中文标签（业务需求/任务/Bug/测试单...）
	Stage          string `json:"stage"`          // 当前阶段（valueStream 标签或 zentao status 中文）
	Priority       string `json:"priority"`       // 优先级 P0..P4
	Relation       string `json:"relation"`       // 我负责/我配合/我关注
	Responsibility string `json:"responsibility"` // 待我处理/待我跟进
	Reason         string `json:"reason"`         // 形成原因（来源禅道 status 或业务场景）
	Deadline       string `json:"deadline"`       // 截止日期 YYYY-MM-DD（无日期空串）
	Owner          string `json:"owner"`          // 责任人展示名
	URL            string `json:"url"`            // 禅道详情 URL 或工作台任务详情 URL
	Action         string `json:"action"`         // 当前可执行或跟进动作
	Blocked        bool   `json:"blocked"`        // 是否存在明确阻塞事实
}

// TodoSummary 我的待办一级快捷指标，与返回列表同源计算。
type TodoSummary struct {
	Pending int `json:"pending"`
	Today   int `json:"today"`
	Overdue int `json:"overdue"`
	Blocked int `json:"blocked"`
	P1      int `json:"p1"`
}

// TodoGroupCounts 我的待办对象域计数。
type TodoGroupCounts struct {
	All       int `json:"all"`
	Approval  int `json:"approval"`
	Demand    int `json:"demand"`
	Execution int `json:"execution"`
	Testing   int `json:"testing"`
	Risk      int `json:"risk"`
	Personal  int `json:"personal"`
}

// TodoListResp 我的待办列表响应。
type TodoListResp struct {
	Items    []TodoItem      `json:"items"`
	Total    int64           `json:"total"`    // 过滤后总数（不含分页截断）
	Page     int             `json:"page"`     // 当前页
	PageSize int             `json:"pageSize"` // 每页条数
	Summary  TodoSummary     `json:"summary"`
	Groups   TodoGroupCounts `json:"groups"`
}

// DoneTab 已办的动作分类。对象类型由 ObjectType 二级筛选；审批决策是跨需求对象的正式动作集合。
type DoneTab string

const (
	DoneTabAll      DoneTab = "all"
	DoneTabApproval DoneTab = "approval"
	DoneTabDemand   DoneTab = "demand"
	DoneTabTask     DoneTab = "task"
	DoneTabBug      DoneTab = "bug"
	DoneTabTest     DoneTab = "test"
)

// TimeRange 已办时间段。
type TimeRange string

const (
	TimeRangeToday     TimeRange = "today"
	TimeRange7d        TimeRange = "7d"
	TimeRangeWeek      TimeRange = "week"
	TimeRange30d       TimeRange = "30d"
	TimeRangeMonth     TimeRange = "month"
	TimeRangeLastMonth TimeRange = "last_month"
	TimeRangeQuarter   TimeRange = "quarter"
	TimeRangeCustom    TimeRange = "custom"
	TimeRangeAll       TimeRange = "all"
)

// DoneListReq 我的已办列表请求。
// V10.1 02 节：已办形成条件 = 本人真实执行的正式业务动作。来源 zt_action + Workbench 审计。
// 严格定义：待办消失不能自动变成已办。
type DoneListReq struct {
	Tab        DoneTab   `form:"tab"`        // 动作分类；默认 all
	TimeRange  TimeRange `form:"timeRange"`  // 时间段；默认 all
	CustomFrom string    `form:"from"`       // 时间段=custom 时生效
	CustomTo   string    `form:"to"`         // 时间段=custom 时生效
	ObjectType string    `form:"objectType"` // 业务需求/任务/Bug/测试单等
	Result     string    `form:"result"`     // 操作结果
	Page       int       `form:"page"`
	PageSize   int       `form:"pageSize"`
}

// Validate 校验 DoneListReq。
func (r *DoneListReq) Validate() []FieldError {
	r.Tab = DoneTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = DoneTabAll
	}
	switch r.Tab {
	case DoneTabAll, DoneTabApproval, DoneTabDemand, DoneTabTask, DoneTabBug, DoneTabTest:
	default:
		return []FieldError{{Field: "tab", Message: "无效的对象域 Tab"}}
	}
	r.TimeRange = TimeRange(strings.TrimSpace(string(r.TimeRange)))
	if r.TimeRange == "" {
		r.TimeRange = TimeRangeAll
	}
	switch r.TimeRange {
	case TimeRangeToday, TimeRange7d, TimeRangeWeek, TimeRange30d, TimeRangeMonth,
		TimeRangeLastMonth, TimeRangeQuarter, TimeRangeCustom, TimeRangeAll:
	default:
		return []FieldError{{Field: "timeRange", Message: "无效的时间段"}}
	}
	r.ObjectType = strings.TrimSpace(r.ObjectType)
	r.Result = strings.TrimSpace(r.Result)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	return nil
}

// DoneAction 我的已办单条：zt_action 投影。
type DoneAction struct {
	ID              int64  `json:"id"`              // zt_action.id
	Actor           string `json:"actor"`           // 操作人（应 = 当前账号）
	Action          string `json:"action"`          // 操作代码（中文化见 DoneActionLabel）
	ObjectType      string `json:"objectType"`      // 对象类型（demand/story/task/bug/testtask...）
	ObjectTypeLabel string `json:"objectTypeLabel"` // 对象类型中文标签
	ObjectID        int64  `json:"objectId"`        // 对象 ID
	ObjectName      string `json:"objectName"`      // 对象标题
	Date            string `json:"date"`            // 操作时间 YYYY-MM-DD HH:MM:SS
	Result          string `json:"result"`          // 操作结果/前后状态
	URL             string `json:"url"`             // 禅道详情 URL
}

// DoneListResp 我的已办列表响应。
type DoneListResp struct {
	Items    []DoneAction `json:"items"`
	Total    int64        `json:"total"`
	Summary  DoneSummary  `json:"summary"` // 时间段概览计数（与待办 focus 卡同构）
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// DoneSummary 已办时间段概览计数。
type DoneSummary struct {
	All     int64 `json:"all"`
	Today   int64 `json:"today"`
	Last7d  int64 `json:"last7d"`
	Week    int64 `json:"week"`
	Last30d int64 `json:"last30d"`
	Month   int64 `json:"month"`
	Quarter int64 `json:"quarter"`
}
