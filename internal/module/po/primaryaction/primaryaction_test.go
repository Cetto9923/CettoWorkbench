// =============================================================================
// 文件: internal/module/po/primaryaction/primaryaction_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 Derive 在每一种价值流阶段下的 key / kind / enabled / reason 输出。
//       与 Stage 5 任务输入锁定的"阶段 → primaryAction 合同表"一一对应。
// 依赖: 仅 primaryaction + 标准库（无 DB）。
// =============================================================================

package primaryaction

import (
	"strings"
	"testing"
)

// TestDerive_AcceptStage 受理阶段：§4-1 阻塞，key 仍返回 + enabled=false + 中文 reason。
func TestDerive_AcceptStage(t *testing.T) {
	in := Input{
		Stage:               StageAccept,
		Kind:                ObjectBusinessDemand,
		ObjectID:            100,
		HasAcceptCapability: true,
	}
	pa := Derive(in)
	if pa.Key != string(KeyApprove) {
		t.Fatalf("StageAccept key = %q, want %q", pa.Key, KeyApprove)
	}
	if pa.Kind != string(KindDrawer) {
		t.Fatalf("StageAccept kind = %q, want %q", pa.Kind, KindDrawer)
	}
	if pa.Enabled {
		t.Fatal("StageAccept must be disabled (PLAN §4-1)")
	}
	if !strings.Contains(pa.Reason, "审批") {
		t.Fatalf("StageAccept reason should mention '审批', got %q", pa.Reason)
	}
}

// TestDerive_ClarifyStage 澄清：enabled 由 HasClarifyCapability 控制。
func TestDerive_ClarifyStage(t *testing.T) {
	cases := []struct {
		name           string
		has            bool
		wantEnabled    bool
		wantReasonHint string
	}{
		{"with_cap", true, true, ""},
		{"without_cap", false, false, "澄清权限"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pa := Derive(Input{
				Stage:                StageClarify,
				Kind:                 ObjectBusinessDemand,
				ObjectID:             101,
				HasClarifyCapability: c.has,
			})
			if pa.Key != string(KeyClarify) {
				t.Fatalf("key = %q, want %q", pa.Key, KeyClarify)
			}
			if pa.Enabled != c.wantEnabled {
				t.Fatalf("enabled = %v, want %v", pa.Enabled, c.wantEnabled)
			}
			if !c.wantEnabled && !strings.Contains(pa.Reason, c.wantReasonHint) {
				t.Fatalf("reason = %q, want contain %q", pa.Reason, c.wantReasonHint)
			}
			if !strings.Contains(pa.URL, "/demands/101/detail") {
				t.Fatalf("url should target demand detail, got %q", pa.URL)
			}
		})
	}
}

// TestDerive_ScheduleStage 排期：业务需求 → /schedule/demands/:id；独立研需 → /schedule/stories/:id。
func TestDerive_ScheduleStage(t *testing.T) {
	biz := Derive(Input{
		Stage: StageSchedule, Kind: ObjectBusinessDemand, ObjectID: 200,
		HasScheduleCapability: true,
	})
	if biz.Enabled != true {
		t.Fatal("biz schedule should be enabled")
	}
	if !strings.Contains(biz.URL, "/schedule/demands/200/scheduling") {
		t.Fatalf("biz schedule url = %q", biz.URL)
	}

	story := Derive(Input{
		Stage: StageSchedule, Kind: ObjectIndependentStory, ObjectID: 300,
		HasScheduleCapability: true,
	})
	if !strings.Contains(story.URL, "/schedule/stories/300/scheduling") {
		t.Fatalf("story schedule url = %q", story.URL)
	}
}

// TestDerive_SubmitTestStage 提测：§4-2 阻塞，enabled=false + reason 提测 endpoint 未配置。
func TestDerive_SubmitTestStage(t *testing.T) {
	pa := Derive(Input{
		Stage: StageDeveloping, Kind: ObjectBusinessDemand, ObjectID: 400,
		HasSubmitTestCapability: true,
	})
	if pa.Key != string(KeySubmitTest) {
		t.Fatalf("key = %q, want %q", pa.Key, KeySubmitTest)
	}
	if pa.Enabled {
		t.Fatal("submit_test must be disabled (PLAN §4-2)")
	}
	if !strings.Contains(pa.Reason, "提测") {
		t.Fatalf("reason = %q", pa.Reason)
	}
}

// TestDerive_TestLinkStage 联调测试：0/1/多张测试单的三种 case。
func TestDerive_TestLinkStage(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageTesting, Kind: ObjectStory, ObjectID: 500,
			HasReadCapability: true,
		})
		if pa.Key != string(KeyViewTestOrder) {
			t.Fatalf("key = %q", pa.Key)
		}
		if pa.Enabled {
			t.Fatal("zero tests must be disabled")
		}
		if !strings.Contains(pa.Reason, "暂无") {
			t.Fatalf("reason = %q", pa.Reason)
		}
	})

	t.Run("one", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageTesting, Kind: ObjectStory, ObjectID: 501,
			HasReadCapability: true,
			TestsCount:        1,
			FirstTestURL:      "http://example.com/testtask-1",
		})
		if !pa.Enabled {
			t.Fatal("one test must be enabled")
		}
		if pa.URL != "http://example.com/testtask-1" {
			t.Fatalf("url = %q", pa.URL)
		}
		if pa.Kind != string(KindExternal) {
			t.Fatalf("kind = %q, want external", pa.Kind)
		}
	})

	t.Run("many", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageTesting, Kind: ObjectStory, ObjectID: 502,
			HasReadCapability: true,
			TestsCount:        3,
		})
		if pa.Enabled {
			t.Fatal("many tests must be disabled (前端选择列表)")
		}
		if !strings.Contains(pa.Reason, "选择") {
			t.Fatalf("reason = %q", pa.Reason)
		}
	})
}

// TestDerive_AcceptanceStage 验收：本人验收 vs 他人验收。
func TestDerive_AcceptanceStage(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageAcceptance, Kind: ObjectBusinessDemand, ObjectID: 600,
			HasAcceptCapability: true,
			IsAcceptanceOwner:   true,
		})
		if pa.Key != string(KeyAcceptDone) {
			t.Fatalf("key = %q, want %q", pa.Key, KeyAcceptDone)
		}
		if !pa.Enabled {
			t.Fatal("self acceptance must be enabled")
		}
		if !strings.Contains(pa.URL, "/demands/600/accept-done") {
			t.Fatalf("url = %q", pa.URL)
		}
	})

	t.Run("not_self_with_urge", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageAcceptance, Kind: ObjectBusinessDemand, ObjectID: 601,
			HasUrgeCapability: true,
		})
		if pa.Key != string(KeyRemindAccept) {
			t.Fatalf("key = %q, want %q", pa.Key, KeyRemindAccept)
		}
		if !pa.Enabled {
			t.Fatal("not-self with urge must be enabled")
		}
	})

	t.Run("not_self_without_urge", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageAcceptance, Kind: ObjectBusinessDemand, ObjectID: 602,
			HasUrgeCapability: false,
		})
		if pa.Key != string(KeyRemindAccept) {
			t.Fatalf("key = %q", pa.Key)
		}
		if pa.Enabled {
			t.Fatal("not-self without urge must be disabled")
		}
		if !strings.Contains(pa.Reason, "催办") {
			t.Fatalf("reason = %q", pa.Reason)
		}
	})
}

// TestDerive_DeliverStage 发起交付：能力缺失则禁用。
func TestDerive_DeliverStage(t *testing.T) {
	pa := Derive(Input{
		Stage: StageDeliver, Kind: ObjectBusinessDemand, ObjectID: 700,
		HasDeliverCapability: true,
	})
	if pa.Key != string(KeyDeliver) || !pa.Enabled {
		t.Fatalf("deliver enabled = %v key = %q", pa.Enabled, pa.Key)
	}
	if !strings.Contains(pa.URL, "/demands/700/deliver") {
		t.Fatalf("url = %q", pa.URL)
	}
}

// TestDerive_ReleaseStage 发布：永远 None（plan §4 显式约定）。
func TestDerive_ReleaseStage(t *testing.T) {
	pa := Derive(Input{
		Stage: StageRelease, Kind: ObjectBusinessDemand, ObjectID: 800,
		HasDeliverCapability: true,
	})
	if pa.Key != "" || pa.Enabled || pa.Label != "—" {
		t.Fatalf("release must be None, got %+v", pa)
	}
}

// TestDerive_FeedbackStage 评价反馈：未评 / 已评 / 无评 三种。
func TestDerive_FeedbackStage(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageFeedback, Kind: ObjectBusinessDemand, ObjectID: 900,
			HasEvaluateCapability:  true,
			HasPendingEvaluateTask: true,
		})
		if pa.Key != string(KeyEvaluate) || !pa.Enabled {
			t.Fatalf("pending evaluate key/enabled = %q/%v", pa.Key, pa.Enabled)
		}
	})
	t.Run("history", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageFeedback, Kind: ObjectBusinessDemand, ObjectID: 901,
			HasReadCapability:     true,
			HasHistoricalEvaluate: true,
		})
		if pa.Key != string(KeyViewEvaluate) || !pa.Enabled {
			t.Fatalf("history evaluate key/enabled = %q/%v", pa.Key, pa.Enabled)
		}
		if !strings.Contains(pa.URL, "/demands/901/detail?tab=history") {
			t.Fatalf("url = %q", pa.URL)
		}
	})
	t.Run("none", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageFeedback, Kind: ObjectBusinessDemand, ObjectID: 902,
		})
		if pa.Key != "" || pa.Enabled {
			t.Fatalf("no pending/history must be None, got %+v", pa)
		}
	})
}

// TestDerive_ClosedAndDelivered 关闭 / 已上线 → None。
func TestDerive_ClosedAndDelivered(t *testing.T) {
	for _, s := range []StageKey{StageClosed, StageDelivered} {
		pa := Derive(Input{Stage: s, Kind: ObjectBusinessDemand, ObjectID: 1000})
		if pa.Key != "" {
			t.Fatalf("%s should be None, got key=%q", s, pa.Key)
		}
	}
}

// TestDerive_OtherStage 兜底 StageOther → None。
func TestDerive_OtherStage(t *testing.T) {
	pa := Derive(Input{Stage: StageOther, Kind: ObjectBusinessDemand, ObjectID: 1100})
	if pa.Key != "" {
		t.Fatalf("StageOther must be None, got key=%q", pa.Key)
	}
}

// TestDerive_ZeroObjectID 防御性：ObjectID=0 → None。
func TestDerive_ZeroObjectID(t *testing.T) {
	pa := Derive(Input{Stage: StageClarify, Kind: ObjectBusinessDemand, ObjectID: 0})
	if pa.Key != "" {
		t.Fatalf("zero id must be None")
	}
}

// TestEnabledAndDisabled_Builders Enabled / DisabledWithReason / None 工厂函数。
func TestEnabledAndDisabled_Builders(t *testing.T) {
	e := Enabled("k", "label", "internal", "/u")
	if !e.Enabled || e.Key != "k" || e.Label != "label" || e.URL != "/u" {
		t.Fatalf("Enabled wrong: %+v", e)
	}
	if e.Reason != "" {
		t.Fatal("Enabled must not have reason")
	}

	d := DisabledWithReason("k", "label", "modal", "/u", "原因")
	if d.Enabled {
		t.Fatal("DisabledWithReason must be disabled")
	}
	if d.Reason != "原因" {
		t.Fatalf("reason = %q", d.Reason)
	}

	n := None()
	if n.Key != "" || n.Label != "—" || n.Enabled {
		t.Fatalf("None shape wrong: %+v", n)
	}
}

// TestFormatNTestTasks 验证多张测试单 reason 文案。
func TestFormatNTestTasks(t *testing.T) {
	if got := fmtNTestTasks(0); !strings.Contains(got, "暂无") {
		t.Fatalf("0 tests: %q", got)
	}
	if got := fmtNTestTasks(2); !strings.Contains(got, "选择") {
		t.Fatalf("2 tests: %q", got)
	}
	if got := fmtNTestTasks(5); !strings.Contains(got, "选择") {
		t.Fatalf("5 tests: %q", got)
	}
}
