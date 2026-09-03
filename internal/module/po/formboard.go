// =============================================================================
// 文件: internal/module/po/formboard.go
// 模块: PO 工作台
// 类型: action
// 职责: 工作看板 Req/Resp 结构体。
// =============================================================================

package po

import "strings"

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

// BoardDemandItem 是需求树节点。
type BoardDemandItem struct {
	Kind           string             `json:"kind"`
	ID             int64              `json:"id"`
	DisplayID      string             `json:"displayId"`
	Title          string             `json:"title"`
	Stage          string             `json:"stage"`
	Status         string             `json:"status"`
	Priority       string             `json:"priority"`
	Owner          string             `json:"owner"`
	SubDemandCount int                `json:"subDemandCount"`
	StoryCount     int                `json:"storyCount"`
	TaskOpenCount  int                `json:"taskOpenCount"`
	Deadline       string             `json:"deadline"`
	Children       []*BoardDemandItem `json:"children,omitempty"`
	Collapse       bool               `json:"collapse"`
	ActionLabel    string             `json:"actionLabel"`
	URL            string             `json:"url"`
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
	ID         int64  `json:"id"`
	DisplayID  string `json:"displayId"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	Type       string `json:"type"`
	StoryID    int64  `json:"storyId"`
	StoryTitle string `json:"storyTitle"`
	StoryURL   string `json:"storyUrl"`
	Owner      string `json:"owner"`
	Deadline   string `json:"deadline"`
	Blocked    bool   `json:"blocked"`
	Overdue    bool   `json:"overdue"`
	URL        string `json:"url"`
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

// BoardOwnerOption 是任务负责人筛选项。
type BoardOwnerOption struct {
	Account string `json:"account"`
	Display string `json:"display"`
}

// BoardTaskSummary 是任务看板顶部快捷筛选数字。
type BoardTaskSummary struct {
	Total   int64 `json:"total"`
	Blocked int64 `json:"blocked"`
	Overdue int64 `json:"overdue"`
}
