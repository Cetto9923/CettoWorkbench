package schedule

import (
	"strings"
	"testing"
)

func TestNormalizeStageFilterForTab(t *testing.T) {
	t.Parallel()

	if got := NormalizeStageFilterForTab("no_story,task_unassigned,task_assigned", "biz"); got != "no_story,task_unassigned,task_assigned" {
		t.Fatalf("biz normalize = %q", got)
	}
	if got := NormalizeStageFilterForTab("no_story,task_unassigned,task_assigned", "indep"); got != "task_unassigned,task_assigned" {
		t.Fatalf("indep normalize = %q", got)
	}
}

func TestBuildIndepStoryStageOrClause(t *testing.T) {
	t.Parallel()

	if got := buildIndepStoryStageOrClause([]string{StageFilterNoStory}); got.sql != "" {
		t.Fatalf("unexpected sql for no_story: %q", got.sql)
	}
	if got := buildIndepStoryStageOrClause([]string{StageFilterTaskUnassigned}); got.sql == "" {
		t.Fatalf("expected sql for task_unassigned")
	}
}

func TestBuildIndepStoryStageOrClauseUsesAggregateSQL(t *testing.T) {
	t.Parallel()

	got := buildIndepStoryStageOrClause([]string{StageFilterNoWindow, StageFilterTaskAssigned})
	if !strings.Contains(got.sql, "SELECT agg.top_id") {
		t.Fatalf("expected aggregate top story query, got: %q", got.sql)
	}
	if strings.Contains(got.sql, "ch.parent = s.id") {
		t.Fatalf("stage filter should avoid correlated child story lookups, got: %q", got.sql)
	}
}

func TestCalcBizDemandStage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		demandIDs       []uint
		allStories      []ZtStory
		mainStories     []ZtStory
		windowByDemand  map[uint]DemandWindowRef
		taskStatByStory map[uint]StoryTaskStat
		want            string
	}{
		{
			name:           "no window first",
			demandIDs:      []uint{1},
			windowByDemand: map[uint]DemandWindowRef{},
			want:           StageNoWindow,
		},
		{
			name:           "no story after window",
			demandIDs:      []uint{1},
			windowByDemand: map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			want:           StageNoStory,
		},
		{
			name:            "no task",
			demandIDs:       []uint{1},
			allStories:      []ZtStory{{ID: 11}},
			mainStories:     []ZtStory{{ID: 11}},
			windowByDemand:  map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 0, Unassigned: 0}},
			want:            StageNoTask,
		},
		{
			name:            "task unassigned",
			demandIDs:       []uint{1},
			allStories:      []ZtStory{{ID: 11}},
			mainStories:     []ZtStory{{ID: 11}},
			windowByDemand:  map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 1, Unassigned: 1}},
			want:            StageTaskUnassigned,
		},
		{
			name:            "task assigned",
			demandIDs:       []uint{1},
			allStories:      []ZtStory{{ID: 11}},
			mainStories:     []ZtStory{{ID: 11}},
			windowByDemand:  map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 1, Unassigned: 0}},
			want:            StageTaskAssigned,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calcBizDemandStage(tc.demandIDs, tc.allStories, tc.mainStories, tc.windowByDemand, tc.taskStatByStory); got != tc.want {
				t.Fatalf("calcBizDemandStage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCalcDemandWindowPhase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		demandIDs      []uint
		stories        []ZtStory
		windowByDemand map[uint]DemandWindowRef
		windowByStory  map[uint]StoryWindowRef
		want           string
	}{
		{
			name:      "no window has no phase",
			demandIDs: []uint{1},
			want:      "",
		},
		{
			name:           "demand window without story is initial",
			demandIDs:      []uint{1},
			windowByDemand: map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			want:           WindowPhaseInitial,
		},
		{
			name:           "demand window with story is final",
			demandIDs:      []uint{1},
			stories:        []ZtStory{{ID: 11}},
			windowByDemand: map[uint]DemandWindowRef{1: {DemandID: 1, WindowID: 10, WindowName: "w"}},
			want:           WindowPhaseFinal,
		},
		{
			name:          "story window with story is final",
			demandIDs:     []uint{1},
			stories:       []ZtStory{{ID: 11}},
			windowByStory: map[uint]StoryWindowRef{11: {StoryID: 11, WindowID: 10, WindowName: "w"}},
			want:          WindowPhaseFinal,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calcDemandWindowPhase(tc.demandIDs, tc.stories, tc.windowByDemand, tc.windowByStory); got != tc.want {
				t.Fatalf("calcDemandWindowPhase() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCanEditSchedulingWindow(t *testing.T) {
	t.Parallel()

	if !canEditSchedulingWindow(0, 1) {
		t.Fatalf("no window should remain editable")
	}
	if !canEditSchedulingWindow(10, 0) {
		t.Fatalf("initial window should remain editable")
	}
	if canEditSchedulingWindow(10, 1) {
		t.Fatalf("final window should be locked")
	}
}

func TestCalcIndependentStoryStage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		stories         []ZtStory
		windowByStory   map[uint]StoryWindowRef
		taskStatByStory map[uint]StoryTaskStat
		want            string
	}{
		{
			name:          "no window",
			stories:       []ZtStory{{ID: 11}},
			windowByStory: map[uint]StoryWindowRef{},
			want:          StageNoWindow,
		},
		{
			name:            "no task",
			stories:         []ZtStory{{ID: 11}},
			windowByStory:   map[uint]StoryWindowRef{11: {StoryID: 11, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 0, Unassigned: 0}},
			want:            StageNoTask,
		},
		{
			name:            "unassigned",
			stories:         []ZtStory{{ID: 11}},
			windowByStory:   map[uint]StoryWindowRef{11: {StoryID: 11, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 1, Unassigned: 1}},
			want:            StageTaskUnassigned,
		},
		{
			name:            "assigned",
			stories:         []ZtStory{{ID: 11}},
			windowByStory:   map[uint]StoryWindowRef{11: {StoryID: 11, WindowID: 10, WindowName: "w"}},
			taskStatByStory: map[uint]StoryTaskStat{11: {StoryID: 11, Total: 1, Unassigned: 0}},
			want:            IndependentStageTaskAssigned,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calcIndependentStoryStage(tc.stories, tc.windowByStory, tc.taskStatByStory); got != tc.want {
				t.Fatalf("calcIndependentStoryStage() = %q, want %q", got, tc.want)
			}
		})
	}
}
