// =============================================================================
// 文件: internal/module/po/blocker_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 「卡点快速响应」纯函数单测：等级优先级、限长、去重与契约。
// 依赖: 无
// =============================================================================

package po

import (
	"reflect"
	"testing"
)

func TestLevelOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   BlockerLevel
		want int
	}{
		{in: BlockerLevelBlocked, want: 0},
		{in: BlockerLevelOverdue, want: 1},
		{in: BlockerLevelRisk, want: 2},
		{in: BlockerLevelCoord, want: 3},
		{in: BlockerLevel("unknown"), want: 99},
	}
	for _, tc := range cases {
		if got := levelOrder(tc.in); got != tc.want {
			t.Errorf("levelOrder(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestLevelLabel(t *testing.T) {
	t.Parallel()

	cases := map[BlockerLevel]string{
		BlockerLevelBlocked: "阻塞",
		BlockerLevelOverdue: "超期",
		BlockerLevelRisk:    "风险",
		BlockerLevelCoord:   "协调",
		BlockerLevel("foo"): "foo",
	}
	for in, want := range cases {
		if got := levelLabel(in); got != want {
			t.Errorf("levelLabel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestClassifyDemand(t *testing.T) {
	t.Parallel()

	cases := map[string]BlockerLevel{
		"developing":     BlockerLevelBlocked,
		"testing":        BlockerLevelOverdue,
		"waitacceptance": BlockerLevelOverdue,
		"schedule":       BlockerLevelRisk,
		"clarify":        BlockerLevelCoord,
		"__unknown__":    BlockerLevelRisk,
	}
	for in, want := range cases {
		if got := classifyDemand(in); got != want {
			t.Errorf("classifyDemand(%q)=%q want %q", in, got, want)
		}
	}
}

func TestClassifyStory(t *testing.T) {
	t.Parallel()

	if got := classifyStory("schedule"); got != BlockerLevelRisk {
		t.Errorf("schedule 应当 risk，got %q", got)
	}
	if got := classifyStory("acceptanced"); got != BlockerLevelOverdue {
		t.Errorf("acceptanced 应当 overdue，got %q", got)
	}
	if got := classifyStory("__any__"); got != BlockerLevelRisk {
		t.Errorf("unknown 应当 risk，got %q", got)
	}
}

func TestIsOwnActionStage(t *testing.T) {
	t.Parallel()

	if !isOwnActionStage("waitacceptance") {
		t.Errorf("waitacceptance 应该是 own action")
	}
	if !isOwnActionStage("acceptanced") {
		t.Errorf("acceptanced 应该是 own action")
	}
	if isOwnActionStage("developing") {
		t.Errorf("developing 不应该是 own action")
	}
}

func TestSortBlockers(t *testing.T) {
	t.Parallel()

	in := []BlockerDetail{
		{ID: "1", Kind: "demand", Level: BlockerLevelRisk, Stage: "排期"},
		{ID: "2", Kind: "demand", Level: BlockerLevelBlocked, IsOwnAction: true, Stage: "提测"},
		{ID: "3", Kind: "story", Level: BlockerLevelBlocked, Stage: "提测"},
		{ID: "4", Kind: "demand", Level: BlockerLevelOverdue, IsOwnAction: true, Stage: "联调测试"},
		{ID: "5", Kind: "demand", Level: BlockerLevelCoord, Stage: "澄清"},
	}
	got := append([]BlockerDetail(nil), in...)
	sortBlockers(got)

	// 期望：先 blocked（同级 own-action first），后 overdue，再 risk，最后 coord。
	wantIDs := []string{"2", "3", "4", "1", "5"}
	gotIDs := make([]string, 0, len(got))
	for _, it := range got {
		gotIDs = append(gotIDs, it.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Errorf("sort result mismatch\n got: %v\nwant: %v", gotIDs, wantIDs)
	}
}

func TestBlockerLimitAndOverallLimit(t *testing.T) {
	if BlockerStageLimit <= 0 || BlockerStageLimit >= 100 {
		t.Fatalf("BlockerStageLimit 取值离群：%d", BlockerStageLimit)
	}
	if BlockerOverallLimit <= BlockerStageLimit {
		t.Fatalf("BlockerOverallLimit(%d) 必须大于 BlockerStageLimit(%d)", BlockerOverallLimit, BlockerStageLimit)
	}
}

func TestBlockerStatusesContainHotStages(t *testing.T) {
	want := map[string]bool{
		"clarify":        true,
		"schedule":       true,
		"developing":     true,
		"testing":        true,
		"waitacceptance": true,
		"acceptanced":    true,
	}
	got := make(map[string]bool, len(blockerStatuses))
	for _, s := range blockerStatuses {
		got[s] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("blockerStatuses 集合已偏离合同\n got: %v\nwant: %v", got, want)
	}
}

func TestBlockerReqValidate(t *testing.T) {
	t.Parallel()
	var req BlockerReq
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("BlockerReq.Validate 必须通过（无入参），got errors: %+v", errs)
	}
}
