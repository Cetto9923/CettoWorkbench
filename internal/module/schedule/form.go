// =============================================================================
// 文件: internal/module/schedule/form.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义排期模块请求/响应结构体。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
	"time"
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

// Validate 校验计划匹配查询参数。
func (r *MatchingPlansReq) Validate() []FieldError {
	var errs []FieldError
	if r.ProductID == 0 {
		errs = append(errs, FieldError{Field: "product_id", Message: "产品 ID 不能为空"})
	}
	endDate := strings.TrimSpace(r.EndDate)
	if endDate == "" {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期不能为空"})
	} else if len(endDate) != 10 || endDate[4] != '-' || endDate[7] != '-' {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期格式无效"})
	}
	return errs
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
	ReleaseDate string               `json:"releaseDate"`
	Name        string               `json:"name"`
	StartDate   string               `json:"startDate"`
	TeamgroupID uint                 `json:"teamgroupId"`
	GroupSize   int                  `json:"groupSize"`
	Products    []WindowProductInput `json:"products"`
}

// Validate 校验新建版本窗口请求。
func (r *CreateReq) Validate() []FieldError {
	return validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)
}

// UpdateReq 更新版本窗口请求。
type UpdateReq struct {
	ID          uint64               `json:"-"`
	ReleaseDate string               `json:"releaseDate"`
	Name        string               `json:"name"`
	StartDate   string               `json:"startDate"`
	TeamgroupID uint                 `json:"teamgroupId"`
	GroupSize   int                  `json:"groupSize"`
	Products    []WindowProductInput `json:"products"`
}

// Validate 校验更新版本窗口请求。
func (r *UpdateReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "窗口 ID 无效"})
	}
	errs = append(errs, validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)...)
	return errs
}

// DeleteReq 删除版本窗口请求。
type DeleteReq struct {
	ID uint64
}

// Validate 校验删除版本窗口请求。
func (r *DeleteReq) Validate() []FieldError {
	if r.ID == 0 {
		return []FieldError{{Field: "id", Message: "窗口 ID 无效"}}
	}
	return nil
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
	CapacityHours    int
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
	ID          uint64                `json:"id"`
	ReleaseDate string                `json:"releaseDate"`
	Name        string                `json:"name"`
	StartDate   string                `json:"startDate"`
	TeamgroupID uint                  `json:"teamgroupId"`
	GroupSize   uint                  `json:"groupSize"`
	Products    []WindowProductDetail `json:"products"`
}

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

// 业需排期阶段（Service 层计算）。
const (
	StageNoWindow       = "未关联窗口"
	StageNoStory        = "未转研发"
	StageNoTask         = "未建任务"
	StageTaskUnassigned = "已建任务未指派"
	StageTaskAssigned   = "已建任务并指派"

	// 独立研发需求 Tab 排期阶段（4 级，末级文案与业需不同）。
	IndependentStageTaskAssigned = "已建任务已指派"

	StoryStageNoWindow  = "未关联窗口"
	StoryStageHasWindow = "已关联窗口"
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
	MainSystem     string `gorm:"column:mainSystem"`
	TeamGroup      string `gorm:"column:teamGroup"`
	BRA            string `gorm:"column:BRA"`
	QD             string `gorm:"column:QD"`
	RD             string `gorm:"column:RD"`
	CreatedBy      string `gorm:"column:createdBy"`
	Pool           uint   `gorm:"column:pool"`
	Parent         uint   `gorm:"column:parent"`
	Hang           string `gorm:"column:hang"`
	Category       string `gorm:"column:category"`
	EstimateLaunch string `gorm:"column:estimateLaunch"`
}

// ListIndependentReq 独立研发需求 Tab 列表查询入参。
type ListIndependentReq struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
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
	ID         uint    `gorm:"column:id" json:"id"`
	Name       string  `gorm:"column:name" json:"name"`
	Type       string  `gorm:"column:type" json:"type"`
	AssignedTo string  `gorm:"column:assignedTo" json:"assignedTo"`
	Estimate   float64 `gorm:"column:estimate" json:"estimate"`
	Consumed   float64 `gorm:"column:consumed" json:"consumed"`
	Left       float64 `gorm:"column:left" json:"left"`
	EstStarted string  `gorm:"column:estStarted" json:"estStarted"`
	Deadline   string  `gorm:"column:deadline" json:"deadline"`
	Status     string  `gorm:"column:status" json:"status"`
	Project    uint    `gorm:"column:project" json:"project"`
	Execution  uint    `gorm:"column:execution" json:"execution"`
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

// DemandSchedulingResp 排期一体化弹窗加载数据。
type DemandSchedulingResp struct {
	*DemandSchedulingDetail
	InvolvedProducts  []ZtProductOption                            `json:"involvedProducts"`
	ProductProjects   map[string][]DemandSchedulingProjectOption   `json:"productProjects"`
	ProjectExecutions map[string][]ZtExecutionOption               `json:"projectExecutions"`
	Stories           []DemandSchedulingStoryItem                  `json:"stories"`
	Windows           []SchedulingWindowOption                     `json:"windows"`
	Users             []SchedulingUserOption                       `json:"users"`
}

// ZtStory 禅道 zt_story 只读投影。
type ZtStory struct {
	ID                      uint   `gorm:"column:id"`
	Title                   string `gorm:"column:title"`
	Pri                     int    `gorm:"column:pri"`
	Product                 uint   `gorm:"column:product"`
	Plan                    string `gorm:"column:plan"`
	Stage                   string `gorm:"column:stage"`
	Status                  string `gorm:"column:status"`
	FromDemand              uint   `gorm:"column:fromDemand"`
	SourceType              string `gorm:"column:sourceType"`
	Parent                  uint   `gorm:"column:parent"`
	IsMainSystemAssociation int     `gorm:"column:isMainSystemAssociation"`
	AssignedTo              string  `gorm:"column:assignedTo"`
	Estimate                float64 `gorm:"column:estimate"`
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
			case "edit", "delete":
				if task.ID == 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".id", Message: "任务 ID 无效"})
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
	Name        string  `json:"name"`
	AssignedTo  string  `json:"assignedTo"`
	Estimate    float64 `json:"estimate"`
	EstStarted  string  `json:"estStarted"`
	Deadline    string  `json:"deadline"`
}
