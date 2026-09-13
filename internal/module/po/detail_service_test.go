// =============================================================================
// 文件: internal/module/po/detail_service_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 业务需求统一详情单元测试（入参解析、阶段映射、耗时、去重口径）。
// =============================================================================

package po

import (
	"testing"
	"time"

	"workbench/internal/module/po/primaryaction"
)

func TestExtractDemandID(t *testing.T) {
	cases := []struct {
		input string
		want  uint
	}{
		{"US12345", 12345},
		{"us67890", 67890},
		{"999", 999},
		{"  US42  ", 42},
		{"invalid", 0},
		{"", 0},
	}

	for _, tc := range cases {
		req := DemandDetailReq{ID: tc.input}
		got := req.ExtractDemandID()
		if got != tc.want {
			t.Errorf("ExtractDemandID(%q) = %d; want %d", tc.input, got, tc.want)
		}
	}
}

func TestBindPrimaryActionSpotlight(t *testing.T) {
	spotlight := &DetailSpotlight{ActionLabel: "排期"}
	bindPrimaryActionSpotlight(spotlight, primaryaction.PrimaryAction{
		Key:     string(primaryaction.KeySchedule),
		Label:   "排期",
		URL:     "/schedule/demands/42/scheduling",
		Enabled: true,
	})
	if spotlight.ActionURL != "/schedule/demands/42/scheduling" {
		t.Fatalf("schedule spotlight URL = %q", spotlight.ActionURL)
	}

	spotlight.ActionURL = ""
	bindPrimaryActionSpotlight(spotlight, primaryaction.PrimaryAction{
		Key:     string(primaryaction.KeySchedule),
		URL:     "/schedule/demands/42/scheduling",
		Enabled: false,
	})
	if spotlight.ActionURL != "" {
		t.Fatal("disabled schedule action must not create a clickable spotlight")
	}

	spotlight = &DetailSpotlight{ActionLabel: "去提测 →"}
	bindPrimaryActionSpotlight(spotlight, primaryaction.PrimaryAction{
		Key:     string(primaryaction.KeySubmitTest),
		Label:   "提测",
		URL:     "/demands/42/submit-test",
		Enabled: true,
	})
	if spotlight.ActionLabel != "提测" {
		t.Fatalf("submit_test spotlight label = %q", spotlight.ActionLabel)
	}
	if spotlight.ActionURL != "" {
		t.Fatalf("submit_test must not bind old /submit-test page URL, got %q", spotlight.ActionURL)
	}
}

func TestMapValueStage(t *testing.T) {
	cases := []struct {
		stage  string
		status string
		wantK  string
		wantL  string
	}{
		{"wait", "", "accept", "已受理"},
		{"inroadmap", "", "clarify", "澄清中"},
		{"incharter", "", "schedule", "排期中"},
		{"developing", "", "submittest", "提测"},
		{"testing", "", "testing", "测试中"},
		{"delivered", "", "publish", "发布"},
		{"closed", "", "closed", "已关闭"},
		{"", "clarify", "clarify", "澄清中"},
		{"", "waitdeliver", "publish", "发布"},
		{"", "released", "greyverify", "生产验证"},
		{"", "closed", "closed", "已关闭"},
		{"weird", "weird-status", "unknown", "未知"},
	}

	for _, tc := range cases {
		gotK, gotL := mapValueStage(tc.stage, tc.status)
		if gotK != tc.wantK || gotL != tc.wantL {
			t.Errorf("mapValueStage(%q, %q) = (%q, %q); want (%q, %q)",
				tc.stage, tc.status, gotK, gotL, tc.wantK, tc.wantL)
		}
	}
}

func TestFormatPriority(t *testing.T) {
	if formatPriority("1") != "P1" {
		t.Errorf("formatPriority(1) want P1")
	}
	if formatPriority("p3") != "P3" {
		t.Errorf("formatPriority(p3) want P3")
	}
	if formatPriority("") != "P2" {
		t.Errorf("formatPriority('') want default P2")
	}
}

func TestBuildValueStream(t *testing.T) {
	svc := &DetailService{}
	now := time.Now().AddDate(0, 0, -4)
	row := &DemandDetailRow{
		Stage:       "inroadmap",
		CreatedDate: &now,
	}

	vs := svc.buildValueStream(row)
	if vs == nil {
		t.Fatalf("buildValueStream returned nil")
	}
	if len(vs.Stages) != 10 {
		t.Errorf("expected 10 stages, got %d", len(vs.Stages))
	}
	if vs.TargetCycleDays != 28 {
		t.Errorf("TargetCycleDays want 28, got %d", vs.TargetCycleDays)
	}

	// 验证第二阶段（澄清）为 current
	if vs.Stages[1].Status != "current" {
		t.Errorf("stage 1 status want current, got %s", vs.Stages[1].Status)
	}
	// 验证第一阶段（受理）为 done，且不得伪造「实际 2 天」
	if vs.Stages[0].Status != "done" {
		t.Errorf("stage 0 status want done, got %s", vs.Stages[0].Status)
	}
	if vs.Stages[0].DurationText == "实际 2 天" {
		t.Errorf("done stage must not fabricate actual duration")
	}
}

func TestBuildValueStreamUnknownDoesNotFallbackClarify(t *testing.T) {
	svc := &DetailService{}
	vs := svc.buildValueStream(&DemandDetailRow{Status: "totally-unknown"})
	if vs == nil {
		t.Fatal("nil value stream")
	}
	foundUnknown := false
	for _, st := range vs.Stages {
		if st.Key == "unknown" && st.Status == "current" {
			foundUnknown = true
		}
		if st.Key == "clarify" && st.Status == "current" {
			t.Fatal("unknown status must not highlight clarify")
		}
	}
	if !foundUnknown {
		t.Fatal("expected unknown current stage")
	}
}

func TestBuildAppQualityTreeNoFabrication(t *testing.T) {
	tree := buildAppQualityTree([]DemandStoryRow{{ID: 1, Title: "x", ProductName: "app"}})
	if tree != nil {
		t.Fatalf("unconnected scanner must return nil tree, got %#v", tree)
	}
}

func TestBuildSpotlight(t *testing.T) {
	svc := &DetailService{}
	sp := svc.buildSpotlight("inroadmap", "")
	if sp == nil {
		t.Fatalf("buildSpotlight returned nil")
	}
	if sp.TargetTab != "requirement" {
		t.Errorf("spotlight for clarify want targetTab requirement, got %s", sp.TargetTab)
	}
	if sp.ActionLabel != "进入澄清办理区 →" {
		t.Errorf("spotlight action label want 进入澄清办理区 →, got %s", sp.ActionLabel)
	}

	sp = svc.buildSpotlight("developing", "developing")
	if sp == nil || sp.Badge != "待提测" {
		t.Fatalf("developing spotlight badge = %#v", sp)
	}
	if sp.ActionLabel != "去提测 →" {
		t.Errorf("developing spotlight action = %q", sp.ActionLabel)
	}
}
