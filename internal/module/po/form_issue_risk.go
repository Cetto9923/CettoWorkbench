package po

import "strings"

type IssueRiskListReq struct {
	Team        uint   `form:"team"`
	TeamgroupID uint   `form:"teamgroupId"`
	Kind        string `form:"kind"`
	Relation    string `form:"relation"`
	Status      string `form:"status"`
	Keyword     string `form:"keyword"`
	Loop        string `form:"loop"`
	Overdue     bool   `form:"overdue"`
	Project     uint   `form:"project"`
	Scope       string `form:"scope"` // 空值表示个人视角；team / dept 表示团队管理范围
	ScopeID     uint   `form:"scopeId"`
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`

	teamAccounts []string
}

func (r *IssueRiskListReq) Validate() []FieldError {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		r.Kind = "issue"
	}
	if r.Kind != "issue" && r.Kind != "risk" {
		return []FieldError{{Field: "kind", Message: "无效的问题风险类型"}}
	}
	if r.Team > 0 {
		r.Scope = "dept"
		r.ScopeID = r.Team
	}
	if r.TeamgroupID > 0 {
		r.Scope = "team"
		r.ScopeID = r.TeamgroupID
	}
	if r.Status == "open" {
		r.Status = ""
		r.Loop = "open"
	}
	if r.Relation == "mine" {
		r.Relation = "allRelated"
	}
	r.Relation = strings.TrimSpace(r.Relation)
	if r.Relation == "" {
		r.Relation = "allRelated"
	}
	if r.Relation != "allRelated" && r.Relation != "myAction" && r.Relation != "mySubmit" {
		return []FieldError{{Field: "relation", Message: "无效的关联范围"}}
	}
	r.Status = strings.TrimSpace(r.Status)
	r.Scope = strings.ToLower(strings.TrimSpace(r.Scope))
	if r.Scope != "" && r.Scope != "team" && r.Scope != "dept" {
		return []FieldError{{Field: "scope", Message: "无效的团队查看范围"}}
	}
	if r.ScopeID > 0 && r.Scope == "" {
		return []FieldError{{Field: "scopeId", Message: "选择具体团队时必须指定查看范围"}}
	}
	if r.Scope != "" && r.Relation != "allRelated" {
		return []FieldError{{Field: "relation", Message: "团队视角不支持个人关联范围"}}
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Loop = strings.TrimSpace(r.Loop)
	if r.Loop == "" {
		r.Loop = "open"
	}
	if r.Loop != "all" && r.Loop != "open" && r.Loop != "closed" {
		return []FieldError{{Field: "loop", Message: "无效的未关闭范围"}}
	}
	if r.Loop != "open" && r.Overdue {
		return []FieldError{{Field: "overdue", Message: "逾期筛选仅在未关闭为 open 时有效"}}
	}
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	return nil
}

type IssueRiskItem struct {
	ID          int64  `json:"id"`
	DisplayID   string `json:"displayId"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Project     string `json:"project"`
	Severity    string `json:"severity"`
	Priority    string `json:"priority"`
	Handler     string `json:"handler"`
	Submitter   string `json:"submitter"`
	Status      string `json:"status"`
	StatusCode  string `json:"statusCode"`
	CreatedDate string `json:"createdDate"`
	PlanDate    string `json:"planDate"`
	Days        int    `json:"days"`
	IsOverdue   bool   `json:"isOverdue"`
	OverdueDays int    `json:"overdueDays"`
	URL         string `json:"url"`
}
type IssueRiskListResp struct {
	Items        []IssueRiskItem    `json:"items"`
	Total        int64              `json:"total"`
	KindCounts   map[string]int64   `json:"kindCounts"`
	LoopCounts   map[string]int64   `json:"loopCounts"`
	OverdueCount int64              `json:"overdueCount"`
	Projects     []IssueRiskProject `json:"projects"`
}
type issueRiskRow struct {
	ID           int64  `gorm:"column:id"`
	Title        string `gorm:"column:title"`
	Pri          string `gorm:"column:pri"`
	Severity     string `gorm:"column:severity"`
	Status       string `gorm:"column:status"`
	CreatedDate  string `gorm:"column:created_date"`
	PlanDate     string `gorm:"column:plan_date"`
	CreatedBy    string `gorm:"column:created_by"`
	AssignedTo   string `gorm:"column:assigned_to"`
	CreatorName  string `gorm:"column:creator_name"`
	HandlerName  string `gorm:"column:handler_name"`
	ResolvedName string `gorm:"column:resolved_name"`
	ClosedName   string `gorm:"column:closed_name"`
	ProjectName  string `gorm:"column:project_name"`
}
