// =============================================================================
// 文件: internal/module/schedule/form_scheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义排期展示类型及版本窗口基础字段校验。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
	"time"
)

func validateWindowSaveFields(
	releaseDate, name, startDate string,
	teamgroupID uint,
	groupSize int,
	products []WindowProductInput,
) []FieldError {
	var errs []FieldError
	releaseDate = strings.TrimSpace(releaseDate)
	if releaseDate == "" {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期不能为空"})
	} else if _, err := time.Parse("2006-01-02", releaseDate); err != nil {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期格式无效"})
	}
	if strings.TrimSpace(name) == "" {
		errs = append(errs, FieldError{Field: "name", Message: "窗口名称不能为空"})
	}
	startDate = strings.TrimSpace(startDate)
	if startDate == "" {
		errs = append(errs, FieldError{Field: "startDate", Message: "窗口开始日期不能为空"})
	} else if _, err := time.Parse("2006-01-02", startDate); err != nil {
		errs = append(errs, FieldError{Field: "startDate", Message: "窗口开始日期格式无效"})
	}
	if teamgroupID == 0 {
		errs = append(errs, FieldError{Field: "teamgroupId", Message: "敏捷小组不能为空"})
	}
	if groupSize < 0 {
		errs = append(errs, FieldError{Field: "groupSize", Message: "小组人数不能为负数"})
	}
	for i, product := range products {
		if product.ProductID == 0 {
			errs = append(errs, FieldError{
				Field:   "products",
				Message: "第 " + strconv.Itoa(i+1) + " 个关联系统 ID 不能为空",
			})
		}
		if product.SyncPlan && strings.TrimSpace(product.PlanTitle) == "" {
			errs = append(errs, FieldError{
				Field:   "products",
				Message: "第 " + strconv.Itoa(i+1) + " 个系统勾选同步创建计划时，计划名称不能为空",
			})
		}
	}
	return errs
}

// DemandSchedulingDetail 排期一体化弹窗业需详情。
type DemandSchedulingDetail struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Pri              int    `json:"pri"`
	BRA              string `json:"bra"`
	BRAName          string `json:"braName"`
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
}

// ZtProductOption 禅道产品/系统下拉选项。
type ZtProductOption struct {
	ID   uint   `gorm:"column:id" json:"id"`
	Name string `gorm:"column:name" json:"name"`
	RD   string `gorm:"column:RD" json:"rd"` // 发布负责人账号
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
	PlanTitle      string                          `json:"planTitle"`
	IsMain         bool                            `json:"isMain"`
	Estimate       float64                         `json:"estimate"`
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
	Dept           uint   `json:"dept"`
	DeptName       string `json:"deptName"`
	Revpoint       int    `json:"revpoint"`
	PointLabel     string `json:"pointLabel"`
	EffectivePoint int    `json:"effectivePoint"`
}

// TaskTypeOption 禅道任务类型下拉项（zt_lang typeList）。
type TaskTypeOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DemandSchedulingResp 排期一体化弹窗加载数据。
type DemandSchedulingResp struct {
	*DemandSchedulingDetail
	InvolvedProducts  []ZtProductOption                          `json:"involvedProducts"`
	ProductProjects   map[string][]DemandSchedulingProjectOption `json:"productProjects"`
	ProjectExecutions map[string][]ZtExecutionOption             `json:"projectExecutions"`
	Stories           []DemandSchedulingStoryItem                `json:"stories"`
	UserStories       []UserStoryItem                            `json:"userStories"`
	Windows           []SchedulingWindowOption                   `json:"windows"`
	Users             []SchedulingUserOption                     `json:"users"`
	TaskTypes         []TaskTypeOption                           `json:"taskTypes"`
}

// ZtStory 禅道 zt_story 只读投影。
type ZtStory struct {
	ID                      uint    `gorm:"column:id"`
	Title                   string  `gorm:"column:title"`
	Pri                     int     `gorm:"column:pri"`
	Product                 uint    `gorm:"column:product"`
	Plan                    string  `gorm:"column:plan"`
	Stage                   string  `gorm:"column:stage"`
	Status                  string  `gorm:"column:status"`
	FromDemand              uint    `gorm:"column:fromDemand"`
	SourceType              string  `gorm:"column:sourceType"`
	Parent                  uint    `gorm:"column:parent"`
	IsMainSystemAssociation int     `gorm:"column:isMainSystemAssociation"`
	AssignedTo              string  `gorm:"column:assignedTo"`
	Estimate                float64 `gorm:"column:estimate"`
}

// ZtDemandUserStory 禅道 zt_demanduserstory 只读投影。
type ZtDemandUserStory struct {
	ID         uint   `gorm:"column:id"`
	Demand     uint   `gorm:"column:demand"`
	Role       string `gorm:"column:role"`
	GV         string `gorm:"column:gv"`
	Product    uint   `gorm:"column:product"`
	Dept       uint   `gorm:"column:dept"`
	Point      int    `gorm:"column:point"`
	Revpoint   int    `gorm:"column:revpoint"`
	SourceType string `gorm:"column:sourceType"`
}

// SaveSchedulingStory 排期保存研发需求条目。
type SaveSchedulingStory struct {
	Action     string               `json:"action"`
	ID         uint                 `json:"id"`
	ProductID  uint                 `json:"productId"`
	Title      string               `json:"title"`
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
