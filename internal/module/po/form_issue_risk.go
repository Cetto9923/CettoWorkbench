package po

import "strings"

type IssueRiskListReq struct {
	Kind     string `form:"kind"`
	Relation string `form:"relation"`
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	Loop     string `form:"loop"`
	Overdue  bool   `form:"overdue"`
	Project  uint   `form:"project"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

func (r *IssueRiskListReq) Validate() []FieldError {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		r.Kind = "issue"
	}
	if r.Kind != "issue" && r.Kind != "risk" {
		return []FieldError{{Field: "kind", Message: "无效的问题风险类型"}}
	}
	r.Relation = strings.TrimSpace(r.Relation)
	if r.Relation == "" {
		r.Relation = "allRelated"
	}
	if r.Relation != "allRelated" && r.Relation != "myAction" && r.Relation != "mySubmit" {
		return []FieldError{{Field: "relation", Message: "无效的关联范围"}}
	}
	r.Status = strings.TrimSpace(r.Status)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Loop = strings.TrimSpace(r.Loop)
	if r.Loop == "" {
		r.Loop = "all"
	}
	if r.Loop != "all" && r.Loop != "open" && r.Loop != "closed" {
		return []FieldError{{Field: "loop", Message: "无效的存续范围"}}
	}
	if r.Loop != "open" && r.Overdue {
		return []FieldError{{Field: "overdue", Message: "逾期筛选仅在存续为 open 时有效"}}
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
