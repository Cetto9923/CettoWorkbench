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

// ListGroupTasksReq 小组研发工作看板查询请求。
type ListGroupTasksReq struct {
	TeamID          uint   `form:"teamId"`
	GroupID         uint   `form:"groupId"`
	Account         string `form:"account"`
	Scope           string `form:"scope"` // all | confirmed | pending
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
	PendingPage     int    `form:"pendingPage"`
	PendingPageSize int    `form:"pendingPageSize"`
}

// Validate 校验小组任务查询参数。
func (r *ListGroupTasksReq) Validate() []FieldError {
	var errs []FieldError
	if r.TeamID == 0 {
		errs = append(errs, FieldError{Field: "teamId", Message: "团队ID不能为空"})
	}
	if r.GroupID == 0 {
		errs = append(errs, FieldError{Field: "groupId", Message: "小组ID不能为空"})
	}
	return errs
}

// TaskPaginationInfo 任务分页元信息。
type TaskPaginationInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

// GroupTaskItem 小组研发任务单卡。
type GroupTaskItem struct {
	ID              int64  `json:"id"`
	DisplayID       string `json:"displayId"`
	Title           string `json:"title"`
	Status          string `json:"status"` // wait | doing | done
	Type            string `json:"type"`
	StoryID         int64  `json:"storyId"`
	StoryTitle      string `json:"storyTitle"`
	Owner           string `json:"owner"`
	OwnerAccount    string `json:"ownerAccount"`
	Deadline        string `json:"deadline"`
	Overdue         bool   `json:"overdue"`
	URL             string `json:"url"`
	Attribution     string `json:"attribution"`     // confirmed | pending
	SourceGroupName string `json:"sourceGroupName"` // 跨组任务所属小组名称（若有）
	IsPendingReview bool   `json:"isPendingReview"` // 是否待核查归属
}

// GroupTaskColumn 小组任务看板单列。
type GroupTaskColumn struct {
	Key   string          `json:"key"`
	Name  string          `json:"name"`
	Count int             `json:"count"`
	Items []GroupTaskItem `json:"items"`
}

// GroupTaskSummary 小组任务指标概览（待核查任务严格与正式指标分离）。
type GroupTaskSummary struct {
	ConfirmedTotal     int `json:"confirmedTotal"`     // 正式任务总数 (wait+doing+近30天done)
	ConfirmedWait      int `json:"confirmedWait"`      // 正式未开始
	ConfirmedDoing     int `json:"confirmedDoing"`     // 正式进行中
	ConfirmedDone      int `json:"confirmedDone"`      // 正式已完成 (近30天口径)
	ConfirmedOverdue   int `json:"confirmedOverdue"`   // 正式逾期 (仅统计未完成任务)
	PendingReviewTotal int `json:"pendingReviewTotal"` // 待核查候选任务总数（不计入正式指标）
}

// GroupTasksResp 小组研发工作看板响应。
type GroupTasksResp struct {
	GroupID             uint               `json:"groupId"`
	GroupName           string             `json:"groupName"`
	TimeRangeLabel      string             `json:"timeRangeLabel"` // 统计口径说明：已完成任务统计近30天，逾期仅统计未完成
	Summary             GroupTaskSummary   `json:"summary"`
	Columns             []GroupTaskColumn  `json:"columns"`
	ConfirmedPagination TaskPaginationInfo `json:"confirmedPagination"`
	PendingReviewTasks  []GroupTaskItem    `json:"pendingReviewTasks"` // 独立分离的待核查候选任务
	PendingPagination   TaskPaginationInfo `json:"pendingPagination"`
}


