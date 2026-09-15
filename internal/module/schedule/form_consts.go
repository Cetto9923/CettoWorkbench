// =============================================================================
// File: internal/module/schedule/form_consts.go
// Module: schedule workbench
// Purpose: Constants and enums for schedule module.
// =============================================================================

package schedule

// Stage constants (service layer calculations).
const (
	StageNoWindow       = "未关联窗口"
	StageNoStory        = "未转研发"
	StageNoTask         = "未建任务"
	StageTaskUnassigned = "已建任务未指派"
	StageTaskAssigned   = "已建任务并指派"

	// Independent development stage for tab.
	IndependentStageTaskAssigned = "已建任务已指派"

	WindowPhaseInitial = "初排"
	WindowPhaseFinal   = "终排"
)

// Batch limits for scheduling save requests.
const (
	MaxSchedulingStories       = 50
	MaxSchedulingTasksPerStory = 50
	MaxStoryTasks              = 50
)

// Stage filter URL parameter values.
const (
	StageFilterNoWindow       = "no_window"
	StageFilterNoStory        = "no_story"
	StageFilterNoTask         = "no_task"
	StageFilterTaskUnassigned = "task_unassigned"
	StageFilterTaskAssigned   = "task_assigned"
)

// Window type filter values (matched to zt_versionwindow.status).
const (
	WindowTypePlanning = "planning"
	WindowTypeCurrent  = "current"
	WindowTypeNext     = "next"
	WindowTypeReleased = "released"
)

// Shortcut filter values.
const (
	FilterAllOpen          = "all_open"
	FilterUnscheduled      = "unscheduled"
	FilterPendingReview    = "pending_review"
	FilterUnassigned       = "unassigned"
	FilterManagerReviewing = "manager_reviewing"
	FilterClosed           = "closed"
)
