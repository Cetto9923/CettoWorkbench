// =============================================================================
// 文件: internal/module/schedule/formfilter.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期列表筛选相关常量、Options 切片与 Normalize 工具函数。
//       拆分自 form.go:阶段筛选 + 窗口类型 + 优先级 + 需求快捷筛选集中。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

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
