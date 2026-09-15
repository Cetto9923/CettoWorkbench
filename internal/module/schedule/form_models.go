// =============================================================================
// File: internal/module/schedule/form_models.go
// Module: schedule workbench
// Purpose: Data structures for schedule module.
// =============================================================================

package schedule

// StageFilterOption 排期阶段下拉选项。
type StageFilterOption struct {
	Value string
	Label string
}

// WindowFilterOption 版本窗口筛选下拉选项。
type WindowFilterOption struct {
	ID   uint
	Name string
}

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

// DemandSchedulingDetail 排期一体化弹窗业需详情。
type DemandSchedulingDetail struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Pri              int    `json:"pri"`
	BRA              string `json:"bra"`
	BRAName          string `json:"braName"`
	RD               string `json:"rd"`
	RDName           string `json:"rdName"`
	QD               string `json:"qd"`
	QDName           string `json:"qdName"`
	Accepter         string `json:"accepter"`
	AccepterName     string `json:"accepterName"`
	MainSystemID     uint   `json:"mainSystemId"`
	MainSystemName   string `json:"mainSystemName"`
	SchedulePlanDate string `json:"schedulePlanDate"`
	DevelopFinish    string `json:"developFinish"`
	TestFinish       string `json:"testFinish"`
	AcceptancedDate  string `json:"acceptancedDate"`
	WindowID         uint   `json:"windowId"`
	WindowName       string `json:"windowName"`
	WindowPhase      string `json:"windowPhase"`
	CanEditWindow    bool   `json:"canEditWindow"`
}

// SchedulingWindowOption 排期弹窗版本窗口下拉项。
type SchedulingWindowOption struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"releaseDate"`
	PlanTestDone string `json:"planTestDone,omitempty"`
	TestDone     string `json:"testDone,omitempty"`
	AcceptDone   string `json:"acceptDone,omitempty"`
}

// SchedulingUserOption 排期弹窗负责人下拉项。
type SchedulingUserOption struct {
	Account  string `json:"account"`
	Realname string `json:"realname"`
	Pinyin   string `json:"pinyin,omitempty"`
	Dept     string `json:"dept,omitempty"`
}

// ZtProductOption 禅道产品/系统下拉选项。
type ZtProductOption struct {
	ID   uint   `gorm:"column:id" json:"id"`
	Name string `gorm:"column:name" json:"name"`
}

// ZtTaskItem 禅道 zt_task 只读投影（排期弹窗）。
type ZtTaskItem struct {
	ID           uint    `gorm:"column:id" json:"id"`
	Name         string  `gorm:"column:name" json:"name"`
	Type         string  `gorm:"column:type" json:"type"`
	Pri          int     `gorm:"column:pri" json:"pri"`
	AssignedTo   string  `gorm:"column:assignedTo" json:"assignedTo"`
	Estimate     float64 `gorm:"column:estimate" json:"estimate"`
	Consumed     float64 `gorm:"column:consumed" json:"consumed"`
	Left         float64 `gorm:"column:left" json:"left"`
	EstStarted   string  `gorm:"column:estStarted" json:"estStarted"`
	Deadline     string  `gorm:"column:deadline" json:"deadline"`
	Status       string  `gorm:"column:status" json:"status"`
	FinishedBy   string  `gorm:"column:finishedBy" json:"finishedBy"`
	FinishedDate string  `gorm:"column:finishedDate" json:"finishedDate"`
	Project      uint    `gorm:"column:project" json:"project"`
	Execution    uint    `gorm:"column:execution" json:"execution"`
}

// ZtProjectOption 禅道项目下拉选项。
type ZtProjectOption struct {
	ID     uint   `gorm:"column:id" json:"id"`
	Name   string `gorm:"column:name" json:"name"`
	Status string `gorm:"column:status" json:"status"`
	Model  string `gorm:"column:model" json:"model"`
}

// ZtExecutionOption 禅道执行下拉选项。
type ZtExecutionOption struct {
	ID     uint   `gorm:"column:id" json:"id"`
	Name   string `gorm:"column:name" json:"name"`
	Type   string `gorm:"column:type" json:"type"`
	Status string `gorm:"column:status" json:"status"`
}

// DemandSchedulingTaskItem 排期弹窗研发任务条目。
type DemandSchedulingTaskItem struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	TypeLabel      string  `json:"typeLabel"`
	Pri            int     `json:"pri"`
	AssignedTo     string  `json:"assignedTo"`
	AssignedToName string  `json:"assignedToName"`
	Estimate       float64 `json:"estimate"`
	EstStarted     string  `json:"estStarted"`
	Deadline       string  `json:"deadline"`
	Project        uint    `json:"project"`
	ProjectName    string  `json:"projectName"`
	Execution      uint    `json:"execution"`
	ExecutionName  string  `json:"executionName"`
}

// DemandSchedulingProjectOption 排期弹窗项目下拉项。
type DemandSchedulingProjectOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// DemandSchedulingStoryItem 排期弹窗用户故事条目。
type DemandSchedulingStoryItem struct {
	ID             uint                            `json:"id"`
	Title          string                          `json:"title"`
	ProductID      uint                            `json:"productId"`
	ProductName    string                          `json:"productName"`
	ModuleID       uint                            `json:"moduleId"`
	ModuleName     string                          `json:"moduleName"`
	PlanID         uint                            `json:"planId"`
	PlanName       string                          `json:"planName"`
	Type           string                          `json:"type"`
	TypeLabel      string                          `json:"typeLabel"`
	Pri            int                             `json:"pri"`
	IsMain         bool                            `json:"isMain"`
	Estimate       float64                         `json:"estimate"`
	Spec           string                          `json:"spec"`
	AssignedTo     string                          `json:"assignedTo"`
	AssignedToName string                          `json:"assignedToName"`
	Tasks          []DemandSchedulingTaskItem      `json:"tasks"`
	Projects       []DemandSchedulingProjectOption `json:"projects"`
}

// UserStoryItem 排期弹窗用户故事条目（来自 zt_demanduserstory）。
type UserStoryItem struct {
	ID             uint   `json:"id"`
	Role           string `json:"role"`
	GV             string `json:"gv"`
	ProductID      uint   `json:"productId"`
	ProductName    string `json:"productName"`
	Revpoint       int    `json:"revpoint"`
	PointLabel     string `json:"pointLabel"`
	EffectivePoint int    `json:"effectivePoint"`
}

// ZtStory 禅道 zt_story 只读投影。
type ZtStory struct {
	ID                      uint    `gorm:"column:id"`
	Title                   string  `gorm:"column:title"`
	Pri                     int     `gorm:"column:pri"`
	Product                 uint    `gorm:"column:product"`
	Module                  uint    `gorm:"column:module"`
	Plan                    string  `gorm:"column:plan"`
	PlanName                string  `gorm:"column:planName"`
	Type                    string  `gorm:"column:type"`
	Stage                   string  `gorm:"column:stage"`
	Status                  string  `gorm:"column:status"`
	FromDemand              uint    `gorm:"column:fromDemand"`
	SourceType              string  `gorm:"column:sourceType"`
	Parent                  uint    `gorm:"column:parent"`
	IsMainSystemAssociation int     `gorm:"column:isMainSystemAssociation"`
	AssignedTo              string  `gorm:"column:assignedTo"`
	Estimate                float64 `gorm:"column:estimate"`
	Spec                    string  `gorm:"column:spec"`
}

// ZtDemandUserStory 禅道 zt_demanduserstory 只读投影。
type ZtDemandUserStory struct {
	ID         uint   `gorm:"column:id"`
	Demand     uint   `gorm:"column:demand"`
	Role       string `gorm:"column:role"`
	GV         string `gorm:"column:gv"`
	Product    uint   `gorm:"column:product"`
	Point      int    `gorm:"column:point"`
	Revpoint   int    `gorm:"column:revpoint"`
	SourceType string `gorm:"column:sourceType"`
}

// SaveSchedulingReq 排期一体化「确认并同步」保存请求。
type SaveSchedulingReq struct {
	WindowID        uint                  `json:"windowId"`
	RD              string                `json:"rd"`
	QD              string                `json:"qd"`
	Accepter        string                `json:"accepter"`
	DevelopFinish   string                `json:"developFinish"`
	TestFinish      string                `json:"testFinish"`
	AcceptancedDate string                `json:"acceptancedDate"`
	Stories         []SaveSchedulingStory `json:"stories"`
}

// SaveSchedulingStory 排期保存研发需求条目。
type SaveSchedulingStory struct {
	Action     string               `json:"action"`
	ID         uint                 `json:"id"`
	ProductID  uint                 `json:"productId"`
	ModuleID   uint                 `json:"moduleId"`
	PlanID     uint                 `json:"planId"`
	Title      string               `json:"title"`
	Type       string               `json:"type"`
	Pri        int                  `json:"pri"`
	AssignedTo string               `json:"assignedTo"`
	Estimate   float64              `json:"estimate"`
	Spec       string               `json:"spec"`
	Tasks      []SaveSchedulingTask `json:"tasks"`
}

// SaveSchedulingTask 排期保存任务条目。
type SaveSchedulingTask struct {
	Action      string  `json:"action"`
	ID          uint    `json:"id"`
	ExecutionID uint    `json:"executionId"`
	Type        string  `json:"type"`
	Pri         int     `json:"pri"`
	Name        string  `json:"name"`
	AssignedTo  string  `json:"assignedTo"`
	Estimate    float64 `json:"estimate"`
	EstStarted  string  `json:"estStarted"`
	Deadline    string  `json:"deadline"`
}

// StoryAttachmentItem 研发需求附件条目。
type StoryAttachmentItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// StoryTaskStoryItem 维护任务弹窗研发需求详情。
type StoryTaskStoryItem struct {
	ID             uint                  `json:"id"`
	Title          string                `json:"title"`
	ProductID      uint                  `json:"productId"`
	ProductName    string                `json:"productName"`
	AssignedToName string                `json:"assignedToName"`
	Spec           string                `json:"spec"`
	Verify         string                `json:"verify"`
	DemandID       uint                  `json:"demandId"`
	DemandName     string                `json:"demandName"`
	WindowName     string                `json:"windowName"`
	ReleaseDate    string                `json:"releaseDate"`
	Attachments    []StoryAttachmentItem `json:"attachments"`
}

// StoryTaskItem 维护任务弹窗任务条目。
type StoryTaskItem struct {
	ID             uint    `json:"id"`
	Type           string  `json:"type"`
	TypeLabel      string  `json:"typeLabel"`
	Name           string  `json:"name"`
	Pri            int     `json:"pri"`
	PriLabel       string  `json:"priLabel"`
	Status         string  `json:"status"`
	StatusLabel    string  `json:"statusLabel"`
	AssignedTo     string  `json:"assignedTo"`
	AssignedToName string  `json:"assignedToName"`
	FinishedBy     string  `json:"finishedBy"`
	FinishedByName string  `json:"finishedByName"`
	FinishedDate   string  `json:"finishedDate"`
	Estimate       float64 `json:"estimate"`
	Consumed       float64 `json:"consumed"`
	Left           float64 `json:"left"`
	Progress       int     `json:"progress"`
	EstStarted     string  `json:"estStarted"`
	Deadline       string  `json:"deadline"`
	ProjectID      uint    `json:"projectId"`
	ProjectName    string  `json:"projectName"`
	ExecutionID    uint    `json:"executionId"`
	ExecutionName  string  `json:"executionName"`
}

// StoryTaskSummary 只读任务列表汇总。
type StoryTaskSummary struct {
	Total         int     `json:"total"`
	WaitCount     int     `json:"waitCount"`
	DoingCount    int     `json:"doingCount"`
	EstimateTotal float64 `json:"estimateTotal"`
	ConsumedTotal float64 `json:"consumedTotal"`
	LeftTotal     float64 `json:"leftTotal"`
}

// StoryTasksResp 维护任务弹窗加载响应。
type StoryTasksResp struct {
	Story              StoryTaskStoryItem              `json:"story"`
	Tasks              []StoryTaskItem                 `json:"tasks"`
	Summary            StoryTaskSummary                `json:"summary"`
	Projects           []DemandSchedulingProjectOption `json:"projects"`
	Users              []SchedulingUserOption          `json:"users"`
	DefaultProjectID   uint                            `json:"defaultProjectId"`
	DefaultExecutionID uint                            `json:"defaultExecutionId"`
}

// SaveStoryTasksReq 维护任务弹窗保存请求。
// 每条任务独立携带 projectId/executionId，支持弹窗内逐行选择项目与执行。
type SaveStoryTasksReq struct {
	ProjectID   uint                 `json:"projectId"`
	ExecutionID uint                 `json:"executionId"`
	Tasks       []SaveStoryTasksTask `json:"tasks"`
}

// SaveStoryTasksTask 维护任务弹窗保存任务条目。
type SaveStoryTasksTask struct {
	Action      string  `json:"action"`
	ID          uint    `json:"id"`
	ProjectID   uint    `json:"projectId"`
	ExecutionID uint    `json:"executionId"`
	Type        string  `json:"type"`
	Pri         int     `json:"pri"`
	Name        string  `json:"name"`
	AssignedTo  string  `json:"assignedTo"`
	Estimate    float64 `json:"estimate"`
	EstStarted  string  `json:"estStarted"`
	Deadline    string  `json:"deadline"`
	Create      bool    `json:"create"`
}
