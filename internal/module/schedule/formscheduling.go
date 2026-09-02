// =============================================================================
// 文件: internal/module/schedule/formscheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化(版本窗口排期)的 Request / Response / 嵌套数据契约。
//       拆分自 form.go:调度类表单 + SaveScheduling 集中。
// 依赖: internal/module/schedule/formshared.go (FieldError)
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

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
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate"`
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
	Revpoint       int    `json:"revpoint"`
	PointLabel     string `json:"pointLabel"`
	EffectivePoint int    `json:"effectivePoint"`
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

// Validate 校验排期保存请求。
func (r *SaveSchedulingReq) Validate() []FieldError {
	var errs []FieldError
	if r.WindowID == 0 {
		errs = append(errs, FieldError{Field: "windowId", Message: "版本窗口不能为空"})
	}
	for i, story := range r.Stories {
		prefix := "stories[" + strconv.Itoa(i) + "]"
		action := strings.TrimSpace(story.Action)
		switch action {
		case "new":
			if story.ProductID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".productId", Message: "系统不能为空"})
			}
			if strings.TrimSpace(story.Title) == "" {
				errs = append(errs, FieldError{Field: prefix + ".title", Message: "研发需求标题不能为空"})
			}
		case "edit", "delete":
			if story.ID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".id", Message: "研发需求 ID 无效"})
			}
		case "":
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "操作类型不能为空"})
		default:
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "不支持的操作类型"})
		}
		for j, task := range story.Tasks {
			taskPrefix := prefix + ".tasks[" + strconv.Itoa(j) + "]"
			taskAction := strings.TrimSpace(task.Action)
			switch taskAction {
			case "new":
				if task.ExecutionID == 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".executionId", Message: "执行不能为空"})
				}
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: taskPrefix + ".name", Message: "任务名称不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: taskPrefix + ".deadline", Message: "截止时间不能为空"})
				}
			case "edit", "delete":
				if task.ID == 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".id", Message: "任务 ID 无效"})
				}
				if taskAction == "edit" {
					if strings.TrimSpace(task.Name) == "" {
						errs = append(errs, FieldError{Field: taskPrefix + ".name", Message: "任务名称不能为空"})
					}
					if task.Estimate <= 0 {
						errs = append(errs, FieldError{Field: taskPrefix + ".estimate", Message: "预估不能为空"})
					}
					if strings.TrimSpace(task.Deadline) == "" {
						errs = append(errs, FieldError{Field: taskPrefix + ".deadline", Message: "截止时间不能为空"})
					}
				}
			case "":
				if action != "delete" {
					errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "任务操作类型不能为空"})
				}
			default:
				errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "不支持的任务操作类型"})
			}
		}
	}
	return errs
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

// StoryAttachmentItem 研发需求附件条目。
