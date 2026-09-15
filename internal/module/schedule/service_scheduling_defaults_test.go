package schedule

import "testing"

func TestBuildDemandSchedulingStoryDefaultsMatchesZenTaoStoryDefaults(t *testing.T) {
	detail := &DemandSchedulingDetail{ID: 63425, Name: "测试新建", Pri: 3, WindowID: 26}
	got := buildDemandSchedulingStoryDefaults(
		detail,
		[]DemandSchedulingClarifyDefault{{ProductID: 7, ProductName: "项目管理系统2.0", Analyst: "analyst"}},
		[]UserStoryItem{
			{ProductID: 7, Role: "123", GV: "123", EffectivePoint: 8},
			{ProductID: 7, Role: "管理员", GV: "维护权限", EffectivePoint: 13},
		},
		[]SchedulingWindowProductPlan{{WindowID: 26, ProductID: 7, PlanID: 9001, PlanName: "26-0918窗口计划"}},
		map[string]string{"analyst": "需求分析员"},
	)
	if len(got) != 1 {
		t.Fatalf("expected one default, got %d", len(got))
	}
	wantTitle := "US63425-测试新建-项目管理系统2.0"
	if got[0].Title != wantTitle {
		t.Fatalf("title = %q, want %q", got[0].Title, wantTitle)
	}
	if got[0].Spec != "1、作为123，我希望123\n2、作为管理员，我希望维护权限\n" {
		t.Fatalf("spec = %q", got[0].Spec)
	}
	if got[0].AssignedTo != "analyst" || got[0].AssignedToName != "需求分析员" {
		t.Fatalf("assignee = %#v", got[0])
	}
	if got[0].PlanID != 9001 || got[0].PlanName != "26-0918窗口计划" {
		t.Fatalf("plan = %#v", got[0])
	}
	if got[0].Type != "story" || got[0].TypeLabel != "功能" || got[0].Pri != 3 || got[0].Estimate != 21 {
		t.Fatalf("type/pri/estimate = %#v", got[0])
	}
}
