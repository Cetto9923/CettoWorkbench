// =============================================================================
// 文件: internal/module/teamleader/form.go
// 模块: 团队长工作台
// 类型: form
// 职责: 定义团队、小组、成员层级查询的请求和响应结构体及参数校验。
// 依赖: 无
// =============================================================================

package teamleader

// FieldError 表单字段校验错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// TeamHierarchyReq 团队层级拓扑查询请求。
type TeamHierarchyReq struct {
	TeamID uint `form:"teamId"` // 指定三级团队ID；若为0则自动识别当前登录人所属团队
}

// Validate 校验查询参数。
func (r *TeamHierarchyReq) Validate() []FieldError {
	return nil
}

// PersonDTO 基础人员简要信息。
type PersonDTO struct {
	Account string `json:"account"`
	Name    string `json:"name"`
}

// TeamMemberDTO 小组成员信息。
type TeamMemberDTO struct {
	Account         string   `json:"account"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	IsMultiGroup    bool     `json:"isMultiGroup"`
	OtherGroupNames []string `json:"otherGroupNames"`
}

// SubGroupDTO 子敏捷小组展示结构体。
type SubGroupDTO struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	GroupLeader PersonDTO       `json:"groupLeader"` // 小组长（明确非SM）
	PO          PersonDTO       `json:"po"`          // 产品经理 / PO
	ScrumMaster PersonDTO       `json:"scrumMaster"` // 敏捷教练（未配置时明确标识）
	MemberCount int             `json:"memberCount"` // 本组有效成员数（已按组内去重）
	Members     []TeamMemberDTO `json:"members"`
}

// TeamSummaryDTO 三级团队大盘信息。
type TeamSummaryDTO struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	Leader             PersonDTO `json:"leader"`             // 团队长
	PO                 PersonDTO `json:"po"`                 // 产品经理 / PO
	TotalUniqueMembers int       `json:"totalUniqueMembers"` // 团队净去重总人数
}

// TeamOptionDTO 授权可切换团队选项。
type TeamOptionDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// TeamHierarchyResp 团队层级完整响应。
type TeamHierarchyResp struct {
	CurrentTeam     *TeamSummaryDTO `json:"currentTeam"`     // 当前查看团队，无关联团队时为 nil
	SubGroups       []SubGroupDTO   `json:"subGroups"`       // 当前团队下属子小组列表
	AuthorizedTeams []TeamOptionDTO `json:"authorizedTeams"` // 当前账号被授权的所有三级团队列表
	MyRole          string          `json:"myRole"`          // 当前账号身份: team_leader / group_leader / po / member / admin / guest
}
