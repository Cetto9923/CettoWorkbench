// =============================================================================
// 文件: internal/module/kanban/form.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板页展示用结构体。
// 依赖: 无
// =============================================================================

package kanban

// MemberRole 小组成员角色（展示排序与底色）。
const (
	MemberRolePO     = "po"
	MemberRoleCoach  = "coach"
	MemberRoleMember = "member"
)

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

// BizDemandItem 看板需求树单条业务需求（字段对齐首页价值流 WorkItemDetail）。
type BizDemandItem struct {
	ID           string `json:"id"`
	Pri          string `json:"pri"`
	Title        string `json:"title"`
	Owner        string `json:"owner"`
	ValueStream  string `json:"valueStream"`
	ZentaoUrl    string `json:"zentaoUrl"`
	ZentaoStatus string `json:"zentaoStatus"`
}

// ListBizDemandsResp 看板业务需求列表响应。
type ListBizDemandsResp struct {
	Items []BizDemandItem `json:"items"`
}
