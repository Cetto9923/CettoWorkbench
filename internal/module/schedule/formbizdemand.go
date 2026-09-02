// =============================================================================
// 文件: internal/module/schedule/formbizdemand.go
// 模块: 排期工作台
// 类型: action
// 职责: 业务需求 Tab 列表 + 独立研发需求 Tab 列表的 Request / Response / 数据契约。
//       拆分自 form.go:列表类表单集中,含嵌套 Item 结构。
// 依赖: internal/module/schedule/formshared.go (FieldError)
// =============================================================================

package schedule

import (
	"strings"
)

// ListBizDemandsReq 业务需求 Tab 列表查询入参。
type ListBizDemandsReq struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	TeamgroupID uint   `form:"teamgroupId"`
	ProductID   uint   `form:"productId"`
	Stage       string `form:"stage"`
	Status      string `form:"status"`
	Keyword     string `form:"keyword"`
	WindowID    uint   `form:"windowId"`
	Scope       string `form:"scope"`
	Filter      string `form:"filter"`     // all_open, unscheduled, pending_review, unassigned, manager_reviewing, closed
	Suspended   bool   `form:"suspended"`  // true 时叠加 AND hang = '1'
	Groups      string `form:"groups"`     // 逗号分隔的小组 ID
	Products    string `form:"products"`   // 逗号分隔的产品 ID
	Stages      string `form:"stages"`     // 逗号分隔的阶段值
	Windows     string `form:"windows"`    // 逗号分隔的版本窗口 ID
	Pri         string `form:"pri"`        // 单值优先级：0-4
	WindowType  string `form:"windowType"` // 版本窗口类型：planning/current/next/released
	DevOwner    string `form:"dev"`        // 开发负责人账号
	TestOwner   string `form:"test"`       // 测试负责人账号
	AcceptOwner string `form:"accept"`     // 验收负责人账号
}

// Validate 校验分页与基础参数。
func (r *ListBizDemandsReq) Validate() []FieldError {
	var errs []FieldError
	if r.Page < 1 {
		errs = append(errs, FieldError{Field: "page", Message: "页码必须大于等于 1"})
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		errs = append(errs, FieldError{Field: "pageSize", Message: "每页条数须在 1 到 100 之间"})
	}
	return errs
}

// Normalize 规范化分页参数。
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
	r.Filter = NormalizeDemandFilter(r.Filter)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	r.Windows = strings.TrimSpace(r.Windows)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.WindowType = NormalizeWindowTypeFilter(r.WindowType)
	r.DevOwner = strings.TrimSpace(r.DevOwner)
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
	WindowPhase      string          `json:"windowPhase"`
	WindowName       string          `json:"windowName"`
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
	WindowPhase      string      `json:"windowPhase"`
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

// ListIndependentReq 独立研发需求 Tab 列表查询入参。
type ListIndependentReq struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	Filter     string `form:"filter"`
	Suspended  bool   `form:"suspended"`  // story 无 hang 字段，查询时忽略
	Groups     string `form:"groups"`     // 逗号分隔的小组 ID
	Products   string `form:"products"`   // 逗号分隔的产品 ID
	Stages     string `form:"stages"`     // 逗号分隔的阶段值
	Windows    string `form:"windows"`    // 逗号分隔的版本窗口 ID
	Keyword    string `form:"keyword"`    // 编号/标题/负责人/系统
	Pri        string `form:"pri"`        // 单值优先级：0-4
	WindowType string `form:"windowType"` // 版本窗口类型：planning/current/next/released
	DevOwner   string `form:"dev"`        // 独立研发需求当前按 assignedTo 过滤
	TestOwner  string `form:"test"`       // 独立研发需求当前按测试任务 assignedTo 过滤
}

// Validate 校验分页参数。
func (r *ListIndependentReq) Validate() []FieldError {
	var errs []FieldError
	if r.Page < 1 {
		errs = append(errs, FieldError{Field: "page", Message: "页码必须大于等于 1"})
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		errs = append(errs, FieldError{Field: "pageSize", Message: "每页条数须在 1 到 100 之间"})
	}
	return errs
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
	r.Filter = NormalizeDemandFilter(r.Filter)
	r.Groups = strings.TrimSpace(r.Groups)
	r.Products = strings.TrimSpace(r.Products)
	r.Stages = strings.TrimSpace(r.Stages)
	r.Windows = strings.TrimSpace(r.Windows)
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Pri = NormalizePriorityFilter(r.Pri)
	r.WindowType = NormalizeWindowTypeFilter(r.WindowType)
	r.DevOwner = strings.TrimSpace(r.DevOwner)
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

// DemandSchedulingDetail 排期一体化弹窗业需详情。
