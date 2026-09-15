// =============================================================================
// File: internal/module/schedule/form_extras.go
// Module: schedule workbench
// Purpose: Additional structs that were originally in form.go but not included in
//          form_models.go. They are required for compilation of handlers and
//          services.
// =============================================================================

package schedule

// FilterCounts 各快捷筛选项数量。
type FilterCounts struct {
	AllOpen          int64
	Unscheduled      int64
	PendingReview    int64
	Unassigned       int64
	ManagerReviewing int64
	Closed           int64
	Suspended        int64 // 当前主筛选下 hang='1' 的数量
}

// WindowTypeFilterOption 版本窗口类型筛选下拉项。
type WindowTypeFilterOption struct {
	Value string
	Label string
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

// DemandWindowRef 业务需求关联的版本窗口。
type DemandWindowRef struct {
	DemandID   uint
	WindowID   uint
	WindowName string
}

// StoryWindowRef 研发需求关联的版本窗口。
type StoryWindowRef struct {
	StoryID     uint
	WindowID    uint
	WindowName  string
	TeamgroupID uint
}

// StoryTaskStat 研发任务统计。
type StoryTaskStat struct {
	StoryID    uint
	Total      int
	Unassigned int
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
	Keyword    string `form:"keyword"`    // 编号 / 标题 / 责任人 / 系统
	Pri        string `form:"pri"`        // 单值优先级：0-4
	WindowType string `form:"windowType"` // 版本窗口类型：planning/current/next/released
	DevOwner   string `form:"dev"`        // 独立研发需求当前按 assignedTo 过滤
	TestOwner  string `form:"test"`       // 独立研发需求当前按测试任务 assignedTo 过滤
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
