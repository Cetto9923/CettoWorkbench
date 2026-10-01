// =============================================================================
// 文件: internal/module/schedule/form_filter.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义排期筛选常量、选项、解析及角标统计请求响应。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strings"
)

// 业需排期阶段（Service 层计算，列表仅二分展示）。
const (
	StageScheduleIncomplete = "排期未完成"
	StageScheduleDone       = "排期已完成"

	// 独立研发需求 Tab 排期阶段（4 级）。
	StageNoWindow                = "未关联窗口"
	StageNoStory                 = "未转研发"
	StageNoTask                  = "未建任务"
	StageTaskUnassigned          = "已建任务未指派"
	StageTaskAssigned            = "已建任务并指派"
	IndependentStageTaskAssigned = "已建任务已指派"
)

// 列表高级筛选排期阶段 URL 参数值。
const (
	StageFilterNoWindow       = "no_window"
	StageFilterNoStory        = "no_story"
	StageFilterNoTask         = "no_task"
	StageFilterTaskUnassigned = "task_unassigned"
	StageFilterTaskAssigned   = "task_assigned"
	StageFilterIncomplete     = "incomplete" // 业需：排期未完成（细分阶段合集）
	StageFilterSchedule       = "schedule"   // 兼容单值写法：等价于 incomplete / schedule
)

// StageFilterOption 排期阶段下拉选项。
type StageFilterOption struct {
	Value string
	Label string
}

// WindowFilterOption 版本窗口筛选下拉选项。
type WindowFilterOption struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	TeamName    string `json:"team_name"`
	TeamgroupID uint   `json:"team_group_id,omitempty"`
	ReleaseDate string `json:"release_date,omitempty"`
}

// ScheduleBizStageFilterOptions 业务需求列表筛选区排期阶段选项。
var ScheduleBizStageFilterOptions = []StageFilterOption{
	{Value: StageFilterIncomplete, Label: StageScheduleIncomplete},
	{Value: StageFilterTaskAssigned, Label: StageScheduleDone},
}

// ScheduleIndependentStageFilterOptions 独立研发需求列表筛选区排期阶段选项。
var ScheduleIndependentStageFilterOptions = []StageFilterOption{
	{Value: StageFilterNoWindow, Label: "未关联窗口"},
	{Value: StageFilterNoTask, Label: "未建任务"},
	{Value: StageFilterTaskUnassigned, Label: "已建任务未指派"},
	{Value: StageFilterTaskAssigned, Label: "已建任务已指派"},
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
		StageFilterIncomplete:     {},
		StageFilterSchedule:       {},
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
	allowed := make(map[string]bool, len(options)+1)
	for _, option := range options {
		allowed[option.Value] = true
	}
	allowed[StageFilterSchedule] = true
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

// 列表快捷筛选 filter 参数值。
const (
	FilterAllOpen          = "all_open"
	FilterUnscheduled      = "unscheduled"
	FilterPendingReview    = "pending_review"
	FilterManagerReviewing = "manager_reviewing"
	FilterClosed           = "closed"

	// FilterCountReuseSuspended 角标复用：挂起数量（非列表快捷筛选值）。
	FilterCountReuseSuspended = "suspended"

	// 角标统计 tab：业务需求 / 研发需求。
	FilterCountsTabDemand = "demand"
	FilterCountsTabStory  = "story"
)

// FilterCounts 各快捷筛选项数量。
type FilterCounts struct {
	AllOpen          int64 `json:"allOpen"`
	Unscheduled      int64 `json:"unscheduled"`
	PendingReview    int64 `json:"pendingReview"`
	ManagerReviewing int64 `json:"managerReviewing"`
	Closed           int64 `json:"closed"`
	Suspended        int64 `json:"suspended"` // 当前主筛选下 hang='1' 的数量
}

// FilterCountsReq 角标统计 ajax 入参。
type FilterCountsReq struct {
	Filter            string `form:"filter"`
	Tab               string `form:"tab"` // demand | story
	ReuseDemandFilter string `form:"reuseDemandFilter"`
	ReuseDemandTotal  int64  `form:"reuseDemandTotal"`
	ReuseStoryFilter  string `form:"reuseStoryFilter"`
	ReuseStoryTotal   int64  `form:"reuseStoryTotal"`
	Groups            string `form:"groups"`
	Products          string `form:"products"`
	Windows           string `form:"windows"`
	Stages            string `form:"stages"`
	Stage             string `form:"stage"`
}

// FilterCountsResp 角标统计 ajax 出参（仅填充当前 tab 对应侧）。
type FilterCountsResp struct {
	Success bool         `json:"success"`
	Demand  FilterCounts `json:"demand"`
	Story   FilterCounts `json:"story"`
}
