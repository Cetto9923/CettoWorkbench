// =============================================================================
// 文件: internal/module/po/formboard.go
// 模块: PO 工作台
// 类型: action
// 职责: 工作看板 Req/Resp 结构体。
// =============================================================================

package po

import (
	"errors"
	"strings"
	"time"

	"workbench/internal/module/po/primaryaction"
)

// BoardDemandReq 是需求看板查询参数。
type BoardDemandReq struct {
	POAccount   string `form:"poAccount"`
	TeamgroupID uint   `form:"teamgroupId"`
	Stage       string `form:"stage"`
	ObjectType  string `form:"objectType"`
	Keyword     string `form:"keyword"`
	Collapse    string `form:"collapse"`
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
}

// GroupMetricsReq 是小组效能指标查询参数。
type GroupMetricsReq struct {
	TeamgroupID uint `form:"teamgroupId"`
}

// Validate 校验小组效能查询。
func (r *GroupMetricsReq) Validate() []FieldError {
	return nil
}

// BoardMetric 是小组效能快照中的一个指标。value/state 为真实聚合结果；
// 无法从真实表得出的指标 value="-"（前端展示为 "—"，禁止 mock）。
type BoardMetric struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Target string `json:"target"`
	Trend  string `json:"trend"`
	State  string `json:"state"` // good / warn / risk / flat

	// 内部用于阈值折算，不参与 JSON 序列化。
	higherIsBetter bool
	targetValue    float64
	measured       float64
	hasValue       bool
}

// GroupMetricsResp 是小组效能指标响应；HasGroup=false 时前端展示 Empty State。
type GroupMetricsResp struct {
	GroupID   uint           `json:"groupId"`
	HasGroup  bool           `json:"hasGroup"`
	GroupName string         `json:"groupName"`
	Metrics   []*BoardMetric `json:"metrics"`
}

// Validate 校验需求看板查询。
func (r *BoardDemandReq) Validate() []FieldError {
	r.POAccount = strings.TrimSpace(r.POAccount)
	r.Stage = strings.TrimSpace(r.Stage)
	r.ObjectType = strings.TrimSpace(r.ObjectType)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Collapse = strings.TrimSpace(r.Collapse)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 200 {
		r.PageSize = 30
	}
	if r.ObjectType != "" && r.ObjectType != "all" && r.ObjectType != "demand" && r.ObjectType != "sub_demand" && r.ObjectType != "story" {
		return []FieldError{{Field: "objectType", Message: "无效的对象类型"}}
	}
	return nil
}

// BoardDemandItem 是需求树节点。业务需求/子业务/研发需求共用一行结构。
// 研发需求(story)是"最细有效推进对象"：携带交付进展(Progress/TaskDone/TaskTotal)
// 与当前执行负责人(CurrentOwner)，且可通过兄弟接口按 storyId 下钻任务。
type BoardDemandItem struct {
	Kind           string                       `json:"kind"`
	ID             int64                        `json:"id"`
	DisplayID      string                       `json:"displayId"`
	Title          string                       `json:"title"`
	Stage          string                       `json:"stage"`
	Status         string                       `json:"status"`
	Priority       string                       `json:"priority"`
	Owner          string                       `json:"owner"`
	SubDemandCount int                          `json:"subDemandCount"`
	StoryCount     int                          `json:"storyCount"`
	TaskOpenCount  int                          `json:"taskOpenCount"`
	Deadline       string                       `json:"deadline"`
	CurrentOwner   string                       `json:"currentOwner"`
	Progress       int                          `json:"progress"`
	TaskTotal      int                          `json:"taskTotal"`
	TaskDone       int                          `json:"taskDone"`
	ProductName    string                       `json:"productName"`
	Independent    bool                         `json:"independent"`
	Children       []*BoardDemandItem           `json:"children,omitempty"`
	Collapse       bool                         `json:"collapse"`
	ActionLabel    string                       `json:"actionLabel"`
	URL            string                       `json:"url"`
	PrimaryAction  *primaryaction.PrimaryAction `json:"primaryAction,omitempty"` // 服务端主操作
}

// BoardDemandResp 是需求看板响应。
type BoardDemandResp struct {
	Tree       []*BoardDemandItem     `json:"tree"`
	Summary    BoardDemandSummary     `json:"summary"`
	Teamgroups []BoardTeamgroupOption `json:"teamgroups"`
}

// BoardDemandSummary 是需求看板顶部摘要。
type BoardDemandSummary struct {
	Total         int64 `json:"total"`
	PendingReview int64 `json:"pendingReview"`
	Blocked       int64 `json:"blocked"`
	Overdue       int64 `json:"overdue"`
}

// BoardTeamgroupOption 是当前用户可切换的小组。
type BoardTeamgroupOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// BoardTaskReq 是任务看板查询参数。
type BoardTaskReq struct {
	OwnerAccount string `form:"ownerAccount"`
	TeamgroupID  uint   `form:"teamgroupId"`
	StoryID      int64  `form:"storyId"`
	StatusFilter string `form:"status"`
	Focus        string `form:"focus"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

// Validate 校验任务看板查询。
func (r *BoardTaskReq) Validate() []FieldError {
	r.OwnerAccount = strings.TrimSpace(r.OwnerAccount)
	r.StatusFilter = strings.TrimSpace(r.StatusFilter)
	r.Focus = strings.TrimSpace(r.Focus)
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 200 {
		r.PageSize = 50
	}
	if r.StoryID < 0 {
		return []FieldError{{Field: "storyId", Message: "无效的研发需求"}}
	}
	if r.StatusFilter != "" && r.StatusFilter != "wait" && r.StatusFilter != "doing" && r.StatusFilter != "done" && r.StatusFilter != "pause" {
		return []FieldError{{Field: "status", Message: "无效的状态过滤"}}
	}
	if r.Focus != "" && r.Focus != "blocked" && r.Focus != "overdue" {
		return []FieldError{{Field: "focus", Message: "无效的关注筛选"}}
	}
	return nil
}

// BoardTaskItem 是一张真实任务卡。
type BoardTaskItem struct {
	ID           int64  `json:"id"`
	DisplayID    string `json:"displayId"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Priority     string `json:"priority"`
	Type         string `json:"type"`
	StoryID      int64  `json:"storyId"`
	StoryTitle   string `json:"storyTitle"`
	StoryURL     string `json:"storyUrl"`
	Owner        string `json:"owner"`
	OwnerAccount string `json:"ownerAccount"`
	Deadline     string `json:"deadline"`
	Blocked      bool   `json:"blocked"`
	Overdue      bool   `json:"overdue"`
	URL          string `json:"url"`
}

// BoardTaskTransitionReq 是任务看板拖拽状态变更请求。
type BoardTaskTransitionReq struct {
	Status       string `json:"status"`
	TeamgroupID  uint   `json:"teamgroupId"`
	FinishedBy   string `json:"finishedBy"`
	FinishedDate string `json:"finishedDate"`
}

// Validate 校验任务拖拽状态变更。
func (r *BoardTaskTransitionReq) Validate() []FieldError {
	r.Status = normalizeBoardTaskStatus(r.Status)
	r.FinishedBy = strings.TrimSpace(r.FinishedBy)
	r.FinishedDate = strings.TrimSpace(r.FinishedDate)
	if r.Status != "wait" && r.Status != "doing" && r.Status != "done" {
		return []FieldError{{Field: "status", Message: "无效的目标状态"}}
	}
	if r.Status == "done" && r.FinishedDate != "" {
		if _, err := parseBoardTaskFinishedDate(r.FinishedDate); err != nil {
			return []FieldError{{Field: "finishedDate", Message: "完成时间格式无效"}}
		}
	}
	return nil
}

func normalizeBoardTaskStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "wait", "未开始":
		return "wait"
	case "doing", "进行中":
		return "doing"
	case "done", "已完成", "完成":
		return "done"
	default:
		return strings.TrimSpace(status)
	}
}

func parseBoardTaskFinishedDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid time format")
}

// BoardTaskColumn 是任务看板三列之一。
type BoardTaskColumn struct {
	Key   string           `json:"key"`
	Name  string           `json:"name"`
	Items []*BoardTaskItem `json:"items"`
}

// BoardTaskResp 是任务看板响应。
type BoardTaskResp struct {
	Columns             []*BoardTaskColumn     `json:"columns"`
	Owners              []BoardOwnerOption     `json:"owners"`
	Teamgroups          []BoardTeamgroupOption `json:"teamgroups"`
	SelectedTeamgroupID uint                   `json:"selectedTeamgroupId"`
	Summary             BoardTaskSummary       `json:"summary"`
}

// BoardOwnerOption 是任务负责人筛选项（Count 为该负责人的任务数，供 chips 显示）。
type BoardOwnerOption struct {
	Account string `json:"account"`
	Display string `json:"display"`
	Count   int64  `json:"count"`
}

// BoardTaskSummary 是任务看板顶部快捷筛选数字。
type BoardTaskSummary struct {
	Total   int64 `json:"total"`
	Blocked int64 `json:"blocked"`
	Overdue int64 `json:"overdue"`
}

// BoardIssueItem 右侧问题栏一条真实问题。
type BoardIssueItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Priority  string `json:"priority"`
	Severity  string `json:"severity"`
	Status    string `json:"status"`
	CreatedBy string `json:"createdBy"`
}

// BoardIssueResp 右侧问题栏响应；open/closed 为按状态聚类的真实计数。
type BoardIssueResp struct {
	Total  int64            `json:"total"`
	Open   int64            `json:"open"`
	Closed int64            `json:"closed"`
	Items  []BoardIssueItem `json:"items"`
}

// BoardIssueAction 是问题在禅道中的一条审计记录。字段均直接来自 zt_action。
type BoardIssueAction struct {
	ID        int64  `json:"id"`
	Date      string `json:"date"`
	Actor     string `json:"actor"`
	ActorName string `json:"actorName"`
	Action    string `json:"action"`
	Extra     string `json:"extra"`
	Comment   string `json:"comment"`
}

// BoardIssueActionPage 是有界加载的问题审计记录页，前端按 nextAfterID 继续加载至创建记录。
type BoardIssueActionPage struct {
	Items       []BoardIssueAction `json:"items"`
	NextAfterID int64              `json:"nextAfterId"`
}

// BoardIssueTransitionReq 是工作台问题状态操作请求；实际状态写入由禅道原生接口完成。
type BoardIssueTransitionReq struct {
	Action string `json:"action" binding:"required"`
}
