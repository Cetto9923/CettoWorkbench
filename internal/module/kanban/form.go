// =============================================================================
// 文件: internal/module/kanban/form.go
// 模块: 工作看板
// 类型: action
// 职责: 需求/任务看板页展示、查询、问题栏与任务状态更新用结构体。
// 依赖: 无
// =============================================================================

package kanban

import "strings"

// MemberRole 小组成员角色（展示排序与底色）。
const (
	MemberRolePO     = "po"
	MemberRoleCoach  = "coach"
	MemberRoleMember = "member"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// UpdateTaskStatusReq 看板拖拽更新任务状态（仅 wait↔doing）。
type UpdateTaskStatusReq struct {
	ID     int64  `json:"-"`
	Status string `json:"status"`
}

// Validate 校验目标状态仅允许 wait / doing。
func (r *UpdateTaskStatusReq) Validate() []FieldError {
	var errs []FieldError
	st := strings.TrimSpace(r.Status)
	if st == "" {
		errs = append(errs, FieldError{Field: "status", Message: "状态不能为空"})
		return errs
	}
	if st != "wait" && st != "doing" {
		errs = append(errs, FieldError{Field: "status", Message: "仅支持未开始与进行中互转"})
		return errs
	}
	r.Status = st
	return errs
}

// MemberItem 敏捷小组成员（PO / 需求负责人行）。
type MemberItem struct {
	Account     string
	Display     string
	Initial     string // 头像首字（rune 安全）
	Role        string // po | coach | member
	DemandCount int64  // 需求看板：与首页价值流「全部」相同（业需+研需去重）
	TaskCount   int64  // 任务看板：指派给该账号的未开始 / 进行中任务
}

// TeamgroupItem 当前用户所属敏捷小组（页头 chips + 成员）。
type TeamgroupItem struct {
	ID      uint
	Name    string
	Members []MemberItem
}

// BizDemandItem 看板需求树单条（字段对齐首页价值流 WorkItemDetail；含业需与研需）。
type BizDemandItem struct {
	Kind         string `json:"kind"` // demand | story
	ID           string `json:"id"`
	Pri          string `json:"pri"`
	Title        string `json:"title"`
	Owner        string `json:"owner"`
	ValueStream  string `json:"valueStream"`
	ZentaoUrl    string `json:"zentaoUrl"`
	ZentaoStatus string `json:"zentaoStatus"`
}

// ListBizDemandsResp 看板需求树列表响应（业需 + 独立研需）。
type ListBizDemandsResp struct {
	Items []BizDemandItem `json:"items"`
}

// ListDemandsReq 需求树查询（按选中负责人账号过滤价值流）。
// Account=all 时按 TeamgroupID 对应小组全员聚合。
type ListDemandsReq struct {
	Account     string `form:"account"`
	TeamgroupID uint   `form:"teamgroupId"`
}

// ListTasksReq 任务看板查询（按选中负责人账号过滤）。
// Account=all 时按 TeamgroupID 对应小组全员聚合。
type ListTasksReq struct {
	Account     string `form:"account"`
	TeamgroupID uint   `form:"teamgroupId"`
}

// TaskItem 任务看板单卡。
type TaskItem struct {
	ID           int64  `json:"id"`
	DisplayID    string `json:"displayId"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	StoryID      int64  `json:"storyId"`
	StoryTitle   string `json:"storyTitle"`
	Owner        string `json:"owner"`
	OwnerAccount string `json:"ownerAccount"`
	Deadline     string `json:"deadline"`
	Blocked      bool   `json:"blocked"`
	Overdue      bool   `json:"overdue"`
	URL          string `json:"url"`
}

// TaskColumn 任务看板三列之一。
type TaskColumn struct {
	Key   string     `json:"key"`
	Name  string     `json:"name"`
	Items []TaskItem `json:"items"`
}

// TaskSummary 页头阻塞/超期计数。
type TaskSummary struct {
	Blocked int64 `json:"blocked"`
	Overdue int64 `json:"overdue"`
}

// ListTasksResp 任务看板三列响应。
type ListTasksResp struct {
	Columns []TaskColumn `json:"columns"`
	Summary TaskSummary  `json:"summary"`
}

// 问题栏 tab：未解决 / 已解决。
const (
	IssueTabUnresolved = "unresolved"
	IssueTabResolved   = "resolved"
)

// ListIssuesReq 看板右侧问题栏查询。
// Account=all 时按 TeamgroupID 对应小组全员聚合；Tab 决定状态筛选。
type ListIssuesReq struct {
	Account     string `form:"account"`
	TeamgroupID uint   `form:"teamgroupId"`
	Tab         string `form:"tab"` // unresolved | resolved
}

// Validate 规范化 tab，非法值回退未解决。
func (r *ListIssuesReq) Validate() []FieldError {
	tab := strings.TrimSpace(r.Tab)
	if tab == "" {
		tab = IssueTabUnresolved
	}
	if tab != IssueTabUnresolved && tab != IssueTabResolved {
		return []FieldError{{Field: "tab", Message: "仅支持 unresolved 或 resolved"}}
	}
	r.Tab = tab
	return nil
}

// IssueItem 问题栏单条。
type IssueItem struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Severity      string `json:"severity"`      // 1|2|3|4，供 .wb-severity[data-severity]
	SeverityLabel string `json:"severityLabel"` // 严重/较严重/较小/建议
	Status        string `json:"status"`
	StatusLabel   string `json:"statusLabel"`
	CreatedBy     string `json:"createdBy"`
	AssignedTo    string `json:"assignedTo"`
	Owner         string `json:"owner"`
	OwnerAccount  string `json:"ownerAccount"`
	URL           string `json:"url"`
}

// IssueTabCounts 未解决 / 已解决计数（不受当前 tab 限制）。
type IssueTabCounts struct {
	Unresolved int64 `json:"unresolved"`
	Resolved   int64 `json:"resolved"`
}

// ListIssuesResp 问题栏列表响应。
type ListIssuesResp struct {
	Items  []IssueItem    `json:"items"`
	Counts IssueTabCounts `json:"counts"`
}
