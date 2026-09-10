// =============================================================================
// 文件: internal/module/schedule/form_batch_cap_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 验证 SaveSchedulingReq / SaveStoryTasksReq 的批量上限（DB-R1）：
//       超过上限返回 FieldError，避免开事务；正常规模仍通过校验。
// =============================================================================

package schedule

import (
	"strings"
	"testing"
)

// TestSaveSchedulingReq_RejectsOverBatchLimit 验证 stories 总数超过 MaxSchedulingStories
// 直接被 Validate 拒绝（不开事务、零 DB 写入）。
func TestSaveSchedulingReq_RejectsOverBatchLimit(t *testing.T) {
	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories:  make([]SaveSchedulingStory, MaxSchedulingStories+1),
	}
	for i := range req.Stories {
		req.Stories[i] = SaveSchedulingStory{
			Action: "new",
			Tasks:  []SaveSchedulingTask{{Action: "new"}},
		}
	}
	errs := req.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected FieldError when stories > %d, got none", MaxSchedulingStories)
	}
	found := false
	for _, e := range errs {
		if e.Field == "stories" && strings.Contains(e.Message, "单次排期最多") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected FieldError on stories field with cap message, got %+v", errs)
	}
}

// TestSaveSchedulingReq_RejectsPerStoryTasksCap 验证单研需的 tasks 数量超限时
// 也被拒绝，且不影响 stories 字段其它校验。
func TestSaveSchedulingReq_RejectsPerStoryTasksCap(t *testing.T) {
	story := SaveSchedulingStory{
		Action: "new",
		Title:  "ok",
		Tasks:  make([]SaveSchedulingTask, MaxSchedulingTasksPerStory+1),
	}
	for i := range story.Tasks {
		story.Tasks[i] = SaveSchedulingTask{Action: "new"}
	}
	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories:  []SaveSchedulingStory{story},
	}
	errs := req.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected FieldError when per-story tasks > %d, got none", MaxSchedulingTasksPerStory)
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "stories[0].tasks") && strings.Contains(e.Message, "任务最多") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected FieldError on stories[0].tasks field with cap message, got %+v", errs)
	}
}

// TestSaveSchedulingReq_AtBoundaryIsAccepted 验证在边界值正好等于上限时
// 不触发上限错误（仅在超限时拒绝）。
func TestSaveSchedulingReq_AtBoundaryIsAccepted(t *testing.T) {
	req := &SaveSchedulingReq{
		WindowID: 1,
		Stories:  make([]SaveSchedulingStory, MaxSchedulingStories),
	}
	for i := range req.Stories {
		req.Stories[i] = SaveSchedulingStory{
			Action: "edit",
			ID:     uint(i + 1),
			Tasks:  []SaveSchedulingTask{},
		}
	}
	errs := req.Validate()
	for _, e := range errs {
		if e.Field == "stories" && strings.Contains(e.Message, "单次排期最多") {
			t.Fatalf("did not expect stories cap at exact boundary, got %+v", errs)
		}
	}
}

// TestSaveStoryTasksReq_RejectsOverBatchLimit 验证 SaveStoryTasksReq 在 tasks 超限时拒绝。
func TestSaveStoryTasksReq_RejectsOverBatchLimit(t *testing.T) {
	req := &SaveStoryTasksReq{
		Tasks: make([]SaveStoryTasksTask, MaxStoryTasks+1),
	}
	for i := range req.Tasks {
		req.Tasks[i] = SaveStoryTasksTask{Action: "delete", ID: uint(i + 1)}
	}
	errs := req.Validate()
	if len(errs) == 0 {
		t.Fatalf("expected FieldError when tasks > %d, got none", MaxStoryTasks)
	}
	found := false
	for _, e := range errs {
		if e.Field == "tasks" && strings.Contains(e.Message, "单次保存最多") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected FieldError on tasks field with cap message, got %+v", errs)
	}
}