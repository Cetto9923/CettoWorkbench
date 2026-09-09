// =============================================================================
// 文件: internal/module/po/form_done.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办请求 / 响应 DTO 与参数校验。
// =============================================================================

package po

import (
	"strings"
)

// DoneTab 已办的一级场景分类（兼容字段）。
type DoneTab string

const (
	DoneTabAll       DoneTab = "all"
	DoneTabApproval  DoneTab = "approval"
	DoneTabDemand    DoneTab = "demand"
	DoneTabExecution DoneTab = "execution"
	DoneTabQuality   DoneTab = "quality"
	DoneTabRisks     DoneTab = "risks"
)

// TimeRange 已办时间段。
type TimeRange string

const (
	TimeRangeToday     TimeRange = "today"
	TimeRange7d        TimeRange = "7d"
	TimeRangeWeek      TimeRange = "week"
	TimeRangeThisWeek  TimeRange = "thisWeek"
	TimeRange30d       TimeRange = "30d"
	TimeRangeMonth     TimeRange = "month"
	TimeRangeThisMonth TimeRange = "thisMonth"
	TimeRangeLastMonth TimeRange = "lastMonth"
	TimeRangeQuarter   TimeRange = "quarter"
	TimeRangeCustom    TimeRange = "custom"
	TimeRangeAll       TimeRange = "all"
)

// DoneListReq 我的已办列表请求参数。
type DoneListReq struct {
	Mode       string    `form:"mode"`       // core | all
	Tab        DoneTab   `form:"tab"`        // 一级场景分类；默认 all
	TimeRange  TimeRange `form:"timeRange"`  // 时间段；默认 all
	CustomFrom string    `form:"from"`       // custom 时生效
	CustomTo   string    `form:"to"`         // custom 时生效
	ObjectType string    `form:"objectType"` // 对象类型（demand/story/task/bug/risk/issue/feedback/release/build/todo）
	Result     string    `form:"result"`     // 处理结果
	Action     string    `form:"action"`     // 处理动作
	ActionCode string    `form:"actionCode"` // 处理动作代码（别名）
	Project    string    `form:"project"`    // 所属项目
	Keyword    string    `form:"keyword"`    // 搜索 ID / 标题 / 操作内容
	Page       int       `form:"page"`
	PageSize   int       `form:"pageSize"`
}

// Validate 校验 DoneListReq。
func (r *DoneListReq) Validate() []FieldError {
	r.Mode = strings.TrimSpace(r.Mode)
	if r.Mode == "" {
		r.Mode = "core"
	}
	r.Tab = DoneTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = DoneTabAll
	}
	r.TimeRange = TimeRange(strings.TrimSpace(string(r.TimeRange)))
	if r.TimeRange == "" {
		r.TimeRange = TimeRangeAll
	}
	r.ObjectType = strings.TrimSpace(r.ObjectType)
	r.Result = strings.TrimSpace(r.Result)
	r.Action = strings.TrimSpace(r.Action)
	if r.Action == "" && r.ActionCode != "" {
		r.Action = strings.TrimSpace(r.ActionCode)
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

// DoneAction 我的已办单条：zt_action 投影与 9 列字段。
type DoneAction struct {
	ID              int64  `json:"id"`
	SourceActionId  int64  `json:"sourceActionId"`
	SourceSystem    string `json:"sourceSystem"`
	Actor           string `json:"actor"`
	ActorName       string `json:"actorName"`
	Action          string `json:"action"`
	ActionKey       string `json:"actionKey"`
	ActionName      string `json:"actionName"`
	IsCoreAction    bool   `json:"isCoreAction"`
	ObjectType      string `json:"objectType"`
	ObjectTypeLabel string `json:"objectTypeLabel"`
	ObjectID        int64  `json:"objectId"`
	ObjectCode      string `json:"objectCode"`
	ObjectName      string `json:"objectName"`
	ObjectTitle     string `json:"objectTitle"`
	Date            string `json:"date"`
	HandledAt       string `json:"handledAt"`
	Result          string `json:"result"`
	ResultCode      string `json:"resultCode"`
	ResultText      string `json:"resultText"`
	BeforeStatus    string `json:"beforeStatus"`
	AfterStatus     string `json:"afterStatus"`
	CurrentStatus   string `json:"currentStatus"`
	ProjectName     string `json:"projectName"`
	ExecutionName   string `json:"executionName"`
	ProductName     string `json:"productName"`
	PoolName        string `json:"poolName"`
	CommentSummary  string `json:"commentSummary"`
	NextOwnerName   string `json:"nextOwnerName"`
	CanOpenObject   bool   `json:"canOpenObject"`
	URL             string `json:"url"`
}

// DoneFacet 对象类型已办数量计数。
type DoneFacet struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// DoneSummary 已办时间段与指标汇总。
type DoneSummary struct {
	Today     int64 `json:"today"`
	Week      int64 `json:"week"`
	Month     int64 `json:"month"`
	Objects   int64 `json:"objects"`
	Estimated bool  `json:"estimated"`

	// 保持对旧前端与测试兼容
	All     int64 `json:"all"`
	Last7d  int64 `json:"last7d"`
	Last30d int64 `json:"last30d"`
	Quarter int64 `json:"quarter"`
}

// DoneListResp 我的已办列表响应。
type DoneListResp struct {
	Items    []DoneAction `json:"items"`
	Total    int64        `json:"total"`
	Summary  DoneSummary  `json:"summary"`
	Facets   []DoneFacet  `json:"facets"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// DoneMetaOption 下拉选项。
type DoneMetaOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DoneMetaAction 操作选项。
type DoneMetaAction struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	ObjectType string `json:"objectType"`
}

// DoneMetaResp 我的已办筛选项元数据。
type DoneMetaResp struct {
	Products        []DoneMetaOption `json:"products"`
	PageSizeOptions []int            `json:"pageSizeOptions"`
	ObjectTypes     []DoneMetaOption `json:"objectTypes"`
	ActionTypes     []DoneMetaAction `json:"actionTypes"`
	Results         []DoneMetaOption `json:"results"`
	TimeRanges      []DoneMetaOption `json:"timeRanges"`
	Projects        []DoneMetaOption `json:"projects"`
	Executions      []DoneMetaOption `json:"executions"`
}

// DoneDetailChange 变更明细。
type DoneDetailChange struct {
	FieldLabel string `json:"fieldLabel"`
	OldValue   string `json:"oldValue"`
	NewValue   string `json:"newValue"`
}

// DoneDetailTimeline 时间线节点。
type DoneDetailTimeline struct {
	ActionName string             `json:"actionName"`
	ActorName  string             `json:"actorName"`
	OccurredAt string             `json:"occurredAt"`
	IsCurrent  bool               `json:"isCurrent"`
	Changes    []DoneDetailChange `json:"changes"`
}

// DoneDetailContext 关联上下文。
type DoneDetailContext struct {
	ProductName   string `json:"productName"`
	ProjectName   string `json:"projectName"`
	ExecutionName string `json:"executionName"`
	PoolName      string `json:"poolName"`
	CurrentStatus string `json:"currentStatus"`
	CurrentOwner  string `json:"currentOwner"`
}

// DoneDetailResp 动作详情抽屉响应。
type DoneDetailResp struct {
	Item     DoneAction           `json:"item"`
	Context  DoneDetailContext    `json:"context"`
	Timeline []DoneDetailTimeline `json:"timeline"`
}
