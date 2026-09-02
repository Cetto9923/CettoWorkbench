// =============================================================================
// 文件: internal/module/po/focus_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 「今日推进焦点」排序、限长、去重等纯函数单测。
// 依赖: 无
// =============================================================================

package po

import (
	"reflect"
	"testing"
)

func TestPriLabelToRank(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want int
	}{
		{in: "P1", want: 1},
		{in: "P2", want: 2},
		{in: "P3", want: 3},
		{in: "P4", want: 4},
		{in: "", want: PriRankUnknown},
		{in: "p1", want: PriRankUnknown}, // 大小写敏感，禅道约定大写
		{in: "PX", want: PriRankUnknown},
		{in: "P5", want: PriRankUnknown}, // 禅道 pri 范围 1-4
	}
	for _, tc := range cases {
		if got := priLabelToRank(tc.in); got != tc.want {
			t.Errorf("priLabelToRank(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestEffectiveRank(t *testing.T) {
	t.Parallel()

	// PriRank 字段优先级高于 Pri 标签（PriRank 在 Service 层填充，避免重复解析）
	if got := effectiveRank(WorkItemDetail{Pri: "P1", PriRank: 3}); got != 3 {
		t.Errorf("PriRank 优先：got %d want 3", got)
	}
	// PriRank 为 0 时回退到 Pri 标签
	if got := effectiveRank(WorkItemDetail{Pri: "P2"}); got != 2 {
		t.Errorf("回退到 Pri：got %d want 2", got)
	}
	// 全部为空 → 未识别
	if got := effectiveRank(WorkItemDetail{}); got != PriRankUnknown {
		t.Errorf("未识别：got %d want %d", got, PriRankUnknown)
	}
}

func TestSortItemsByPriASC(t *testing.T) {
	t.Parallel()

	in := []WorkItemDetail{
		{ID: "1", Pri: "P4", PriRank: 4, Kind: "demand", ValueStream: "排期"},
		{ID: "2", Pri: "P1", PriRank: 1, Kind: "demand", ValueStream: "澄清"},
		{ID: "3", Pri: "", PriRank: PriRankUnknown, Kind: "story", ValueStream: "提测"},
		{ID: "4", Pri: "P2", PriRank: 2, Kind: "demand", ValueStream: "验收"},
		{ID: "5", Pri: "P1", PriRank: 1, Kind: "demand", ValueStream: "验收"},
	}
	got := append([]WorkItemDetail(nil), in...)
	sortItemsByPriASC(got)

	// 期望 P1 优先；同级按 valueStream asc, kind asc, id asc；未识别排末位
	wantIDs := []string{"2", "5", "4", "1", "3"}
	gotIDs := make([]string, 0, len(got))
	for _, it := range got {
		gotIDs = append(gotIDs, it.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Errorf("sort result mismatch\n got: %v\nwant: %v", gotIDs, wantIDs)
	}
}

func TestFocusLimitIsFive(t *testing.T) {
	if FocusLimit != 5 {
		t.Fatalf("FocusLimit = %d want 5（首页 UX 对齐前 5 条）", FocusLimit)
	}
}

func TestFocusStatusesContainRequiredStages(t *testing.T) {
	// 断言根据用户与 PO 心智合同固定的价值流阶段集合。
	want := map[string]bool{
		"clarify":        true, // 澄清
		"schedule":       true, // 排期
		"developing":     true, // 提测
		"waitacceptance": true, // 验收
		"released":       true, // 发起交付
	}
	got := make(map[string]bool, len(focusStatuses))
	for _, s := range focusStatuses {
		got[s] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("focusStatuses 集合已偏离合同\n got: %v\nwant: %v", got, want)
	}
}

func TestFocusReqValidate(t *testing.T) {
	t.Parallel()
	var req FocusReq
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("FocusReq.Validate 必须通过（无入参），got errors: %+v", errs)
	}
}
