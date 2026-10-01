// =============================================================================
// 文件: internal/module/schedule/form_list.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义需求列表、版本窗口请求响应及展示类型。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strings"
)

// TeamgroupOption 敏捷小组下拉选项。
type TeamgroupOption struct {
	ID          uint
	DisplayName string
}

// CreateWindowFormData 新建版本窗口弹窗表单数据。
type CreateWindowFormData struct {
	Teamgroups []TeamgroupOption
	Products   []ZtProduct
}

// MatchingPlansReq 计划匹配查询请求。
type MatchingPlansReq struct {
	ProductID uint   `form:"product_id"`
	EndDate   string `form:"end_date"`
}

// MatchingPlanItem 计划匹配结果项。
type MatchingPlanItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// MatchingPlansResp 计划匹配查询响应。
type MatchingPlansResp struct {
	Plans    []MatchingPlanItem `json:"plans"`
	HasMatch bool               `json:"has_match"`
}

// CreateReq 新建版本窗口保存请求。
type CreateReq struct {
	ReleaseDate  string               `json:"releaseDate"`
	Name         string               `json:"name"`
	StartDate    string               `json:"startDate"`
	WindowType   string               `json:"windowType"`
	PlanTestDone string               `json:"planTestDone"`
	TestDone     string               `json:"testDone"`
	AcceptDone   string               `json:"acceptDone"`
	TeamgroupID  uint                 `json:"teamgroupId"`
	GroupSize    int                  `json:"groupSize"`
	Products     []WindowProductInput `json:"products"`
}

// UpdateReq 更新版本窗口请求。
type UpdateReq struct {
	ID           uint64               `json:"-"`
	ReleaseDate  string               `json:"releaseDate"`
	Name         string               `json:"name"`
	StartDate    string               `json:"startDate"`
	WindowType   string               `json:"windowType"`
	PlanTestDone string               `json:"planTestDone"`
	TestDone     string               `json:"testDone"`
	AcceptDone   string               `json:"acceptDone"`
	TeamgroupID  uint                 `json:"teamgroupId"`
	GroupSize    int                  `json:"groupSize"`
	Products     []WindowProductInput `json:"products"`
}

// DeleteReq 删除版本窗口请求。
type DeleteReq struct {
	ID uint64
}

// ListWindowsResp 版本窗口维护列表响应。
type ListWindowsResp struct {
	Windows []WindowListItem
}

// WindowListItem 版本窗口维护列表项。
type WindowListItem struct {
	ID               uint64
	Name             string
	ReleaseDate      string
	Range            string
	DemandCount      int
	CapacityHours    int
	UsedHours        int
	RemainingHours   int
	BlockedCount     int
	UsedPercent      int
	CanEdit          bool
	CanDelete        bool
	HasLinkedDemands bool
}

// WindowProductInput 版本窗口关联系统及计划同步选项。
type WindowProductInput struct {
	ProductID uint   `json:"productId"`
	SyncPlan  bool   `json:"syncPlan"`
	PlanTitle string `json:"planTitle"`
}

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// WindowProductDetail 版本窗口关联产品及计划详情。
type WindowProductDetail struct {
	ProductID   uint               `json:"productId"`
	ProductName string             `json:"productName"`
	SyncPlan    bool               `json:"syncPlan"`
	PlanTitle   string             `json:"planTitle"`
	PlanID      *uint              `json:"planId,omitempty"`
	HasMatch    bool               `json:"hasMatch"`
	Plans       []MatchingPlanItem `json:"plans,omitempty"`
}

// WindowDetailResp 版本窗口详情响应。
type WindowDetailResp struct {
	ID           uint64                `json:"id"`
	ReleaseDate  string                `json:"releaseDate"`
	Name         string                `json:"name"`
	StartDate    string                `json:"startDate"`
	WindowType   string                `json:"windowType"`
	PlanTestDone string                `json:"planTestDone"`
	TestDone     string                `json:"testDone"`
	AcceptDone   string                `json:"acceptDone"`
	TeamgroupID  uint                  `json:"teamgroupId"`
	GroupSize    uint                  `json:"groupSize"`
	Products     []WindowProductDetail `json:"products"`
}

// Normalize 规范化分页与筛选参数。
func (r *ListBizDemandsReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	if r.Stages == "" && strings.TrimSpace(r.Stage) != "" {
		r.Stages = strings.TrimSpace(r.Stage)
	}
	r.Windows = strings.TrimSpace(r.Windows)
	r.Filter = NormalizeDemandFilterWithWindows(r.Filter, r.Windows)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.TestOwner = strings.TrimSpace(r.TestOwner)
	r.AcceptOwner = strings.TrimSpace(r.AcceptOwner)
}

// ListBizDemandsResp 业务需求 Tab 列表响应。
type ListBizDemandsResp struct {
	Total int64           `json:"total"`
	Items []BizDemandItem `json:"items"`
}

// BizDemandItem 顶层业需（树形一级）。
type BizDemandItem struct {
	ID               uint            `json:"id"`
	Name             string          `json:"name"`
	Pri              int             `json:"pri"`
	Status           string          `json:"status"`
	MainSystemName   string          `json:"mainSystemName"`
	ExtraSystemCount int             `json:"extraSystemCount"`
	TeamgroupName    string          `json:"teamgroupName"`
	OwnerName        string          `json:"ownerName"`
	Stage            string          `json:"stage"`
	WindowName       string          `json:"windowName"`
	Suspended        bool            `json:"suspended"`
	Children         []SubDemandItem `json:"children"`
	Stories          []StoryItem     `json:"stories"`
}

// SubDemandItem 子业需（树形二级）。
type SubDemandItem struct {
	ID               uint        `json:"id"`
	Name             string      `json:"name"`
	Pri              int         `json:"pri"`
	Status           string      `json:"status"`
	MainSystemName   string      `json:"mainSystemName"`
	ExtraSystemCount int         `json:"extraSystemCount"`
	TeamgroupName    string      `json:"teamgroupName"`
	OwnerName        string      `json:"ownerName"`
	Stage            string      `json:"stage"`
	WindowName       string      `json:"windowName"`
	Stories          []StoryItem `json:"stories"`
}

// StoryItem 研发需求（树形三级）。
type StoryItem struct {
	ID                      uint   `json:"id"`
	Title                   string `json:"title"`
	Pri                     int    `json:"pri"`
	ProductName             string `json:"productName"`
	Stage                   string `json:"stage"`
	WindowName              string `json:"windowName"`
	TeamgroupName           string `json:"teamgroupName"`
	AssignedTo              string `json:"assignedTo"`
	AssignedToName          string `json:"assignedToName"`
	TaskCount               int    `json:"taskCount"`
	IsMainSystemAssociation int    `json:"isMainSystemAssociation"`
}

// StoryWindowRef 研发需求关联的版本窗口。
type StoryWindowRef struct {
	StoryID     uint
	WindowID    uint
	WindowName  string
	TeamgroupID uint
}

// DemandWindowRef 业务需求关联的版本窗口。
type DemandWindowRef struct {
	DemandID   uint
	WindowID   uint
	WindowName string
}

// StoryTaskStat 研发任务统计。
type StoryTaskStat struct {
	StoryID    uint
	Total      int
	Unassigned int
}

// ZtDemand 禅道 zt_demand 只读投影。
type ZtDemand struct {
	ID             uint   `gorm:"column:id"`
	Name           string `gorm:"column:name"`
	Pri            string `gorm:"column:pri"`
	Status         string `gorm:"column:status"`
	AssignedTo     string `gorm:"column:assignedTo"`
	MainSystem     string `gorm:"column:mainSystem"`
	TeamGroup      string `gorm:"column:teamGroup"`
	BRA            string `gorm:"column:BRA"`
	QD             string `gorm:"column:QD"`
	RD             string `gorm:"column:RD"`
	CreatedBy      string `gorm:"column:createdBy"`
	Pool           uint   `gorm:"column:pool"`
	Parent         int    `gorm:"column:parent"`
	Hang           string `gorm:"column:hang"`
	Category       string `gorm:"column:category"`
	EstimateLaunch string `gorm:"column:estimateLaunch"`
}

// Normalize 规范化分页参数。
func (r *ListIndependentReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	if r.Stages == "" && strings.TrimSpace(r.Stage) != "" {
		r.Stages = strings.TrimSpace(r.Stage)
	}
	r.Windows = strings.TrimSpace(r.Windows)
	r.Filter = NormalizeDemandFilterWithWindows(r.Filter, r.Windows)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.TestOwner = strings.TrimSpace(r.TestOwner)
}

// ListIndependentResp 独立研发需求 Tab 列表响应。
type ListIndependentResp struct {
	Total int64                  `json:"total"`
	Items []IndependentStoryItem `json:"items"`
}

// IndependentStoryItem 独立研发需求（树形一级/二级）。
type IndependentStoryItem struct {
	ID             uint                   `json:"id"`
	Title          string                 `json:"title"`
	Pri            int                    `json:"pri"`
	ProductName    string                 `json:"productName"`
	AssignedToName string                 `json:"assignedToName"`
	Stage          string                 `json:"stage"`
	WindowName     string                 `json:"windowName"`
	TaskCount      int                    `json:"taskCount"`
	TeamgroupName  string                 `json:"teamgroupName"`
	Children       []IndependentStoryItem `json:"children"`
}
