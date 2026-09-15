// =============================================================================
// File: internal/module/schedule/form_helpers.go
// Module: schedule workbench
// Purpose: Helper functions and static option slices for schedule module.
// =============================================================================

package schedule

import "strings"

// Stage filter option slices – exported for use by other packages.
var ScheduleBizStageFilterOptions = []StageFilterOption{{Value: StageFilterNoWindow, Label: "未关联窗口"}, {Value: StageFilterNoStory, Label: "未转研发"}, {Value: StageFilterNoTask, Label: "未建任务"}, {Value: StageFilterTaskUnassigned, Label: "已建任务未指派"}, {Value: StageFilterTaskAssigned, Label: "已建任务并指派"}}

var ScheduleIndependentStageFilterOptions = []StageFilterOption{{Value: StageFilterNoWindow, Label: "未关联窗口"}, {Value: StageFilterNoTask, Label: "未建任务"}, {Value: StageFilterTaskUnassigned, Label: "已建任务未指派"}, {Value: StageFilterTaskAssigned, Label: "已建任务已指派"}}

var ScheduleWindowTypeFilterOptions = []WindowTypeFilterOption{{Value: WindowTypePlanning, Label: "规划中"}, {Value: WindowTypeCurrent, Label: "当前窗口"}, {Value: WindowTypeNext, Label: "下一窗口"}, {Value: WindowTypeReleased, Label: "已发布"}}

// ScheduleStageFilterOptionsForTab returns the stage filter options for the given tab.
func ScheduleStageFilterOptionsForTab(tab string) []StageFilterOption {
	if tab == "indep" {
		return ScheduleIndependentStageFilterOptions
	}
	return ScheduleBizStageFilterOptions
}

// allowedStageFilterValues returns a map of allowed stage values for a tab.
func allowedStageFilterValues(tab string) map[string]bool {
	options := ScheduleBizStageFilterOptions
	if tab == "indep" {
		options = ScheduleIndependentStageFilterOptions
	}
	allowed := make(map[string]bool, len(options))
	for _, opt := range options {
		allowed[opt.Value] = true
	}
	return allowed
}

// NormalizePriorityFilter normalizes priority filter values (0‑4).
func NormalizePriorityFilter(value string) string {
	switch strings.TrimSpace(value) {
	case "0", "1", "2", "3", "4":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

// NormalizeWindowTypeFilter normalizes window type filter values.
func NormalizeWindowTypeFilter(value string) string {
	switch strings.TrimSpace(value) {
	case WindowTypePlanning, WindowTypeCurrent, WindowTypeNext, WindowTypeReleased:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

// NormalizeDemandFilter normalizes shortcut filter values.
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

// storyPointLabel maps a story point number to a label, matching ZenTao config.
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
