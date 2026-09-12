// =============================================================================
// 文件: internal/module/kanban/form.go
// 模块: 工作看板
// 类型: action
// 职责: 需求/任务看板页展示、查询与任务状态更新用结构体。
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
	Account string
	Display string
	Initial string // 头像首字（rune 安全）
	Role    string // po | coach | member
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
type ListDemandsReq struct {
	Account string `form:"account"`
}

// ListTasksReq 任务看板查询（按选中负责人账号过滤）。
type ListTasksReq struct {
	Account string `form:"account"`
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
