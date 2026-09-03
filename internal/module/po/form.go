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
	TodoTabApproval TodoTab = "approval" // 审批决策（V10.1 默认第 1 Tab）
	TodoTabDemand   TodoTab = "demand"   // 需求治理（业务需求 + 研发需求 + 反馈）
	TodoTabAll      TodoTab = "all"      // 全部（跨 Tab 汇总）
)

// Relation 我的关系 V10.1 02 节。
type Relation string

const (
	RelationAll        Relation = "all"
	RelationInCharge   Relation = "in_charge"  // 我负责
	RelationCooperate  Relation = "cooperate"  // 我配合
	RelationFollow     Relation = "follow"     // 我关注
)

// Responsibility 办理责任 V10.1 02 节。
type Responsibility string

const (
	ResponsibilityAll         Responsibility = "all"
	ResponsibilityMyAction     Responsibility = "my_action"     // 待我处理
	ResponsibilityMyFollowUp   Responsibility = "my_follow_up"  // 待我跟进
)

// TodoListReq 我的待办列表请求。
// V10.1 02 节：7 维 AND = Tab ∩ 办理场景 ∩ 阶段 ∩ 对象 ∩ 我的关系 ∩ 办理责任 ∩ 关键词。
// 本期最小可用：Tab + 我的关系 + 办理责任 + 关键词。其它维度作为后续扩展点。
type TodoListReq struct {
	Tab           TodoTab        `form:"tab"`           // 对象域 Tab；默认 approval
	Relation      Relation       `form:"relation"`      // 我的关系；默认 all
	Responsibility Responsibility `form:"responsibility"` // 办理责任；默认 all
	Keyword       string         `form:"keyword"`       // 关键词
	Page          int            `form:"page"`          // 页码；1-based
	PageSize      int            `form:"pageSize"`      // 每页条数；默认 20
}

// Validate 校验 TodoListReq。
func (r *TodoListReq) Validate() []FieldError {
	r.Tab = TodoTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = TodoTabApproval
	}
	if r.Tab != TodoTabApproval && r.Tab != TodoTabDemand && r.Tab != TodoTabAll {
		return []FieldError{{Field: "tab", Message: "无效的对象域 Tab"}}
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
	Kind         string `json:"kind"`            // demand / story / task / bug / test
	ID           int64  `json:"id"`              // 业务需求 ID（业需/任务/...各自主键）
	DisplayID    string `json:"displayId"`       // 展示编号：业需 US{id}，研需 U{id}，任务/单据 TASK-{id} 等
	Title        string `json:"title"`           // 标题
	Type         string `json:"type"`            // 对象类型中文标签（业务需求/任务/Bug/测试单...）
	Stage        string `json:"stage"`           // 当前阶段（valueStream 标签或 zentao status 中文）
	Priority     string `json:"priority"`        // 优先级 P0..P4
	Relation     string `json:"relation"`        // 我负责/我配合/我关注
	Responsibility string `json:"responsibility"` // 待我处理/待我跟进
	Reason       string `json:"reason"`          // 形成原因（来源禅道 status 或业务场景）
	Deadline     string `json:"deadline"`        // 截止日期 YYYY-MM-DD（无日期空串）
	Owner        string `json:"owner"`           // 责任人展示名
	URL          string `json:"url"`             // 禅道详情 URL 或工作台任务详情 URL
}

// TodoListResp 我的待办列表响应。
type TodoListResp struct {
	Items     []TodoItem `json:"items"`
	Total     int64      `json:"total"`     // 过滤后总数（不含分页截断）
	Page      int        `json:"page"`      // 当前页
	PageSize  int        `json:"pageSize"`  // 每页条数
}

// DoneTab 已办对象域。V10.1 02 节：本期先打通需求治理 + 研发执行。
type DoneTab string

const (
	DoneTabAll    DoneTab = "all"
	DoneTabDemand DoneTab = "demand"
	DoneTabTask   DoneTab = "task"
	DoneTabBug    DoneTab = "bug"
	DoneTabTest   DoneTab = "test"
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
	Tab       DoneTab   `form:"tab"`       // 对象域；默认 all
	TimeRange TimeRange `form:"timeRange"` // 时间段；默认 all
	CustomFrom string   `form:"from"`      // 时间段=custom 时生效
	CustomTo   string   `form:"to"`        // 时间段=custom 时生效
	ObjectType string   `form:"objectType"` // 业务需求/任务/Bug/测试单等
	Result     string   `form:"result"`     // 操作结果
	Page       int      `form:"page"`
	PageSize   int      `form:"pageSize"`
}

// Validate 校验 DoneListReq。
func (r *DoneListReq) Validate() []FieldError {
	r.Tab = DoneTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = DoneTabAll
	}
	switch r.Tab {
	case DoneTabAll, DoneTabDemand, DoneTabTask, DoneTabBug, DoneTabTest:
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
	ID         int64  `json:"id"`         // zt_action.id
	Actor      string `json:"actor"`      // 操作人（应 = 当前账号）
	Action     string `json:"action"`     // 操作代码（中文化见 DoneActionLabel）
	ObjectType string `json:"objectType"` // 对象类型（demand/story/task/bug/testtask）
	ObjectID   int64  `json:"objectId"`   // 对象 ID
	ObjectName string `json:"objectName"` // 对象标题
	Date       string `json:"date"`       // 操作时间 YYYY-MM-DD HH:MM:SS
	Result     string `json:"result"`     // 操作结果/前后状态
}

// DoneListResp 我的已办列表响应。
type DoneListResp struct {
	Items    []DoneAction `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// NoticeListReq 通知中心列表请求。
// 本期实现: quick view 筛选 (全部/未读/需处理/异常提醒/今日新增) + 关键词。
// workbench 为主, 分类由 zt_action.action + zt_notify.objectType 推断, 不强套 V10.1 6 类。
type NoticeListReq struct {
	QuickView string `form:"quickView"` // all / unread / action / abnormal / today
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// Validate 校验 NoticeListReq。
func (r *NoticeListReq) Validate() []FieldError {
	r.QuickView = strings.TrimSpace(r.QuickView)
	if r.QuickView == "" {
		r.QuickView = "all"
	}
	switch r.QuickView {
	case "all", "unread", "action", "abnormal", "today":
	default:
		return []FieldError{{Field: "quickView", Message: "无效的快捷视图"}}
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

// NoticeItem 通知中心单条。
type NoticeItem struct {
	ID         int64  `json:"id"`
	ObjectType string `json:"objectType"` // 关联对象类型
	ObjectID   int64  `json:"objectId"`   // 关联对象 ID
	Subject    string `json:"subject"`    // 主题
	Data       string `json:"data"`       // 正文
	Actor      string `json:"actor"`      // 触发人
	Action     string `json:"action"`     // 触发的 action 标识
	Category   string `json:"category"`   // 推断分类 (workbench 实现)
	Read       bool   `json:"read"`       // 是否已读
	Date       string `json:"date"`       // 时间
	URL        string `json:"url"`        // 关联对象 URL (本期空, 后续接入 SSO)
}

// NoticeBucketResp 通知中心响应（含顶部 quick view 计数 + 列表）。
type NoticeBucketResp struct {
	Items    []NoticeItem `json:"items"`
	Total    int64        `json:"total"`
	Unread   int64        `json:"unread"`
	Action   int64        `json:"action"`
	Abnormal int64        `json:"abnormal"`
	Today    int64        `json:"today"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// FollowTab 我的关注对象视图。V10.1 04 节明确：只有 2 个对象视图（业务需求 / 项目报告），
// 不再有"全部"对象 Tab；切 Tab 时重置基础集合。
type FollowTab string

const (
	FollowTabDemand         FollowTab = "demand"          // 业务需求（默认）
	FollowTabProjectReport  FollowTab = "project_report"  // 项目报告
)

// FollowScope 我的关注二级筛选（V10.1 04 节：业务需求内部有 全部/重点关注/已关闭 等）。
type FollowScope string

const (
	FollowScopeAll      FollowScope = "all"
	FollowScopeKey      FollowScope = "key"      // 重点关注
	FollowScopeClosed   FollowScope = "closed"   // 已关闭
)

// FollowListReq 我的关注列表请求。
// V10.1 04 节：对象 Tab × 内部二级筛选 × 关键词；不再有跨对象的"全部"Tab。
type FollowListReq struct {
	Tab       FollowTab    `form:"tab"`       // 业务需求 / 项目报告
	Scope     FollowScope  `form:"scope"`     // 业务需求内部 全部/重点关注/已关闭
	Keyword   string       `form:"keyword"`
	Page      int          `form:"page"`
	PageSize  int          `form:"pageSize"`
}

// Validate 校验 FollowListReq。
func (r *FollowListReq) Validate() []FieldError {
	r.Tab = FollowTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = FollowTabDemand
	}
	switch r.Tab {
	case FollowTabDemand, FollowTabProjectReport:
	default:
		return []FieldError{{Field: "tab", Message: "无效的对象视图"}}
	}
	r.Scope = FollowScope(strings.TrimSpace(string(r.Scope)))
	if r.Scope == "" {
		r.Scope = FollowScopeAll
	}
	switch r.Scope {
	case FollowScopeAll, FollowScopeKey, FollowScopeClosed:
	default:
		return []FieldError{{Field: "scope", Message: "无效的二级筛选"}}
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

// FollowItem 我的关注单条。
type FollowItem struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	Owner      string `json:"owner"`
	LatestNote string `json:"latestNote"` // 最新动态
	Date       string `json:"date"`
	IsKey      bool   `json:"isKey"`     // 是否重点关注
	IsClosed   bool   `json:"isClosed"`  // 是否已关闭
	URL        string `json:"url"`
}

// FollowListResp 我的关注响应。
type FollowListResp struct {
	Items    []FollowItem `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}
