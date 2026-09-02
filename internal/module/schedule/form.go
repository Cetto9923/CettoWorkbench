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

	WindowPhaseInitial = "初排"
	WindowPhaseFinal   = "终排"
)

// 列表高级筛选排期阶段 URL 参数值。
const (
	StageFilterNoWindow       = "no_window"
	StageFilterNoStory        = "no_story"
	StageFilterNoTask         = "no_task"
	StageFilterTaskUnassigned = "task_unassigned"
	StageFilterTaskAssigned   = "task_assigned"
)

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

// ScheduleBizStageFilterOptions 业务需求列表筛选区排期阶段选项。
var ScheduleBizStageFilterOptions = []StageFilterOption{
	{Value: StageFilterNoWindow, Label: "未关联窗口"},
	{Value: StageFilterNoStory, Label: "未转研发"},
	{Value: StageFilterNoTask, Label: "未建任务"},
	{Value: StageFilterTaskUnassigned, Label: "已建任务未指派"},
	{Value: StageFilterTaskAssigned, Label: "已建任务并指派"},
}

// ScheduleIndependentStageFilterOptions 独立研发需求列表筛选区排期阶段选项。
var ScheduleIndependentStageFilterOptions = []StageFilterOption{
	{Value: StageFilterNoWindow, Label: "未关联窗口"},
	{Value: StageFilterNoTask, Label: "未建任务"},
	{Value: StageFilterTaskUnassigned, Label: "已建任务未指派"},
	{Value: StageFilterTaskAssigned, Label: "已建任务已指派"},
}

// 版本窗口类型筛选值（对应 zt_versionwindow.status）。
const (
	WindowTypePlanning = "planning"
	WindowTypeCurrent  = "current"
	WindowTypeNext     = "next"
	WindowTypeReleased = "released"
)

// WindowTypeFilterOption 版本窗口类型筛选下拉项。
type WindowTypeFilterOption struct {
	Value string
	Label string
}

// ScheduleWindowTypeFilterOptions 列表筛选区版本窗口类型选项。
var ScheduleWindowTypeFilterOptions = []WindowTypeFilterOption{
	{Value: WindowTypePlanning, Label: "规划中"},
	{Value: WindowTypeCurrent, Label: "当前窗口"},
	{Value: WindowTypeNext, Label: "下一窗口"},
	{Value: WindowTypeReleased, Label: "已发布"},
}

// ParseCommaSeparatedUints 解析逗号分隔的无符号整型列表。
func ParseCommaSeparatedUints(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil || value == 0 {
			continue
		}
		id := uint(value)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseCommaSeparatedStages 解析逗号分隔的排期阶段筛选值。
func ParseCommaSeparatedStages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	allowed := map[string]struct{}{
		StageFilterNoWindow:       {},
		StageFilterNoStory:        {},
		StageFilterNoTask:         {},
		StageFilterTaskUnassigned: {},
		StageFilterTaskAssigned:   {},
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := allowed[part]; !ok {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeStageFilterForTab 保留当前列表 Tab 支持的排期阶段值。
func NormalizeStageFilterForTab(raw, tab string) string {
	stages := ParseCommaSeparatedStages(raw)
	if len(stages) == 0 {
		return ""
	}
	allowed := allowedStageFilterValues(tab)
	out := make([]string, 0, len(stages))
	for _, stage := range stages {
		if allowed[stage] {
			out = append(out, stage)
		}
	}
	return strings.Join(out, ",")
}

// ScheduleStageFilterOptionsForTab 返回当前列表 Tab 的排期阶段选项。
func ScheduleStageFilterOptionsForTab(tab string) []StageFilterOption {
	if tab == "indep" {
		return ScheduleIndependentStageFilterOptions
	}
	return ScheduleBizStageFilterOptions
}

func allowedStageFilterValues(tab string) map[string]bool {
	options := ScheduleBizStageFilterOptions
	if tab == "indep" {
		options = ScheduleIndependentStageFilterOptions
	}
	allowed := make(map[string]bool, len(options))
	for _, option := range options {
		allowed[option.Value] = true
	}
	return allowed
}

// NormalizePriorityFilter 规范化优先级筛选参数。
func NormalizePriorityFilter(value string) string {
	switch strings.TrimSpace(value) {
	case "0", "1", "2", "3", "4":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

// NormalizeWindowTypeFilter 规范化版本窗口类型筛选参数。
func NormalizeWindowTypeFilter(value string) string {
	switch strings.TrimSpace(value) {
	case WindowTypePlanning, WindowTypeCurrent, WindowTypeNext, WindowTypeReleased:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

// 列表快捷筛选 filter 参数值。
const (
	FilterAllOpen          = "all_open"
	FilterUnscheduled      = "unscheduled"
	FilterPendingReview    = "pending_review"
	FilterUnassigned       = "unassigned"
	FilterManagerReviewing = "manager_reviewing"
	FilterClosed           = "closed"
)

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

// NormalizeDemandFilter 规范化快捷筛选参数，默认全部未关闭。
func NormalizeDemandFilter(filter string) string {
	switch strings.TrimSpace(filter) {
	case FilterUnscheduled, FilterPendingReview, FilterUnassigned,
		FilterManagerReviewing, FilterClosed:
		return strings.TrimSpace(filter)
	case "suspended":
		return FilterAllOpen
	default:
		return FilterAllOpen
	}
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

// Validate 校验维护任务保存请求。
func (r *SaveStoryTasksReq) Validate() []FieldError {
	var errs []FieldError
	for i, task := range r.Tasks {
		prefix := "tasks[" + strconv.Itoa(i) + "]"
		action := strings.TrimSpace(task.Action)
		switch action {
		case "new":
			if task.Create {
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: prefix + ".name", Message: "任务名称不能为空"})
				}
				if task.ExecutionID == 0 {
					errs = append(errs, FieldError{Field: prefix + ".executionId", Message: "执行不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: prefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: prefix + ".deadline", Message: "截止时间不能为空"})
				}
			}
		case "edit", "delete":
			if task.ID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".id", Message: "任务 ID 无效"})
			}
			if action == "edit" && task.ExecutionID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".executionId", Message: "执行不能为空"})
			}
			if action == "edit" {
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: prefix + ".name", Message: "任务名称不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: prefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: prefix + ".deadline", Message: "截止时间不能为空"})
				}
			}
		case "":
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "操作类型不能为空"})
		default:
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "不支持的操作类型"})
		}
	}
	return errs
}

// SaveStoryTasksTask 维护任务弹窗保存任务条目。
// ProjectID/ExecutionID 由每条任务独立携带，projectId 仅用于前端联动加载执行，
// 后端通过 executionId 反查所属 project。
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

// storyPointLabel 将故事点数字映射为关键词，与禅道 config/changshu.php:152-156 一致。
func storyPointLabel(point int) string {
	switch point {
	case 2:
		return "微型"
	case 3:
		return "小型"
	case 5:
		return "中型"
	case 8:
		return "大型"
	default:
		return ""
	}
}
