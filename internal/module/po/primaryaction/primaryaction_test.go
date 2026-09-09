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

// TestDerive_AcceptStage 评审阶段：待我评审、创建人撤回/提交评审、他人只读。
func TestDerive_AcceptStage(t *testing.T) {
	// 1. 待评审 + 待我评审 -> 评审按钮
	t.Run("wait_and_can_review", func(t *testing.T) {
		in := Input{
			Stage:               StageAccept,
			Status:              "wait",
			Kind:                ObjectBusinessDemand,
			ObjectID:            100,
			CanReview:           true,
			HasAcceptCapability: true,
		}
		pa := Derive(in)
		if pa.Key != string(KeyApprove) {
			t.Fatalf("StageAccept key = %q, want %q", pa.Key, KeyApprove)
		}
		if pa.Label != "评审" {
			t.Fatalf("StageAccept label = %q, want 评审", pa.Label)
		}
		if pa.Kind != string(KindDrawer) {
			t.Fatalf("StageAccept kind = %q, want %q", pa.Kind, KindDrawer)
		}
		if !pa.Enabled {
			t.Fatal("StageAccept must be enabled")
		}
		if pa.URL != "/demands/100/review" {
			t.Fatalf("StageAccept URL = %q, want local review endpoint", pa.URL)
		}
	})

	// 2. 待评审 + 非待我评审 + 创建人 -> 撤回评审按钮
	t.Run("wait_and_is_creator", func(t *testing.T) {
		in := Input{
			Stage:     StageAccept,
			Status:    "wait",
			Kind:      ObjectBusinessDemand,
			ObjectID:  101,
			CanReview: false,
			IsCreator: true,
		}
		pa := Derive(in)
		if pa.Key != string(KeyWithdrawReview) {
			t.Fatalf("key = %q, want %q", pa.Key, KeyWithdrawReview)
		}
		if pa.Label != "撤回" {
			t.Fatalf("label = %q, want 撤回", pa.Label)
		}
		if !pa.Enabled {
			t.Fatal("withdraw_review must be enabled")
		}
		if pa.URL != "/demands/101/withdraw-review" {
			t.Fatalf("URL = %q, want /demands/101/withdraw-review", pa.URL)
		}
	})

	// 3. 草稿/驳回 + 创建人 -> 提交评审按钮
	t.Run("draft_or_refuse_and_is_creator", func(t *testing.T) {
		for _, st := range []string{"draft", "refuse"} {
			in := Input{
				Stage:     StageAccept,
				Status:    st,
				Kind:      ObjectBusinessDemand,
				ObjectID:  102,
				IsCreator: true,
			}
			pa := Derive(in)
			if pa.Key != string(KeySubmitReview) {
				t.Fatalf("status %s key = %q, want %q", st, pa.Key, KeySubmitReview)
			}
			if pa.Label != "提交评审" {
				t.Fatalf("status %s label = %q, want 提交评审", st, pa.Label)
			}
			if !pa.Enabled {
				t.Fatalf("status %s submit_review must be enabled", st)
			}
		}
	})

	// 4. 草稿/驳回/非待我评审的待评审 + 他人 -> None (—)
	t.Run("others_none", func(t *testing.T) {
		for _, st := range []string{"draft", "refuse", "wait"} {
			in := Input{
				Stage:     StageAccept,
				Status:    st,
				Kind:      ObjectBusinessDemand,
				ObjectID:  103,
				CanReview: false,
				IsCreator: false,
			}
			pa := Derive(in)
			if pa.Key != "" || pa.Enabled {
				t.Fatalf("status %s for others should be None, got %+v", st, pa)
			}
		}
	})
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
			if pa.URL != "/demands/101/clarify" {
				t.Fatalf("url = %q, want local clarify endpoint", pa.URL)
			}
		})
	}
}

// TestDerive_ScheduleStage 排期：按对象类型进入 Workbench 排期办理页。
func TestDerive_ScheduleStage(t *testing.T) {
	biz := Derive(Input{
		Stage: StageSchedule, Kind: ObjectBusinessDemand, ObjectID: 200,
		HasScheduleCapability: true,
	})
	if !biz.Enabled || biz.Kind != string(KindSchedule) {
		t.Fatalf("biz schedule should be enabled, got %+v", biz)
	}
	if !strings.Contains(biz.URL, "/schedule/demands/200/scheduling") {
		t.Fatalf("url = %q", biz.URL)
	}

	story := Derive(Input{
		Stage: StageSchedule, Kind: ObjectIndependentStory, ObjectID: 300,
		HasScheduleCapability: true,
	})
	if !story.Enabled || !strings.Contains(story.URL, "/schedule/stories/300/scheduling") {
		t.Fatalf("story schedule should be enabled, got %+v", story)
	}
}

// TestDerive_SubmitTestStage 提测：进入保留的 Workbench 办理页。
func TestDerive_SubmitTestStage(t *testing.T) {
	pa := Derive(Input{
		Stage: StageDeveloping, Kind: ObjectBusinessDemand, ObjectID: 400,
		HasSubmitTestCapability: true,
	})
	if pa.Key != string(KeySubmitTest) {
		t.Fatalf("key = %q, want %q", pa.Key, KeySubmitTest)
	}
	if !pa.Enabled || pa.Kind != string(KindInternal) || !strings.Contains(pa.URL, "/demands/400/submit-test") {
		t.Fatalf("submit_test should enter Workbench page, got %+v", pa)
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
		if !pa.Enabled || pa.Kind != string(KindDrawer) {
			t.Fatalf("acceptance should use native drawer entry, got %+v", pa)
		}
		if pa.URL != "/demands/600/acceptance" {
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
			t.Fatal("urge must be enabled for the responsible user")
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

// TestDerive_DeliverStage 发起交付：有权限进入交付抽屉。
func TestDerive_DeliverStage(t *testing.T) {
	pa := Derive(Input{
		Stage: StageDeliver, Kind: ObjectBusinessDemand, ObjectID: 700,
		HasDeliverCapability: true,
	})
	if pa.Key != string(KeyDeliver) || !pa.Enabled || pa.Kind != string(KindDrawer) {
		t.Fatalf("deliver must open drawer: %+v", pa)
	}
	if pa.URL != "/demands/700/deliver" {
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
		if pa.Key != string(KeyEvaluate) || !pa.Enabled || pa.Kind != string(KindExternal) {
			t.Fatalf("pending evaluate key/enabled = %q/%v", pa.Key, pa.Enabled)
		}
	})
	t.Run("history", func(t *testing.T) {
		pa := Derive(Input{
			Stage: StageFeedback, Kind: ObjectBusinessDemand, ObjectID: 901,
			HasReadCapability:     true,
			HasHistoricalEvaluate: true,
		})
		if pa.Key != "" || pa.Enabled {
			t.Fatalf("historical evaluate must have no PO action, got %+v", pa)
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
