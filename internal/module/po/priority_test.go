// =============================================================================
// 文件: internal/module/po/priority_test.go
// 模块: PO 工作台
// 类型: readonly
// 职责: zt_demand / zt_story 真实优先级列归一为可排序权重的纯函数单测。
// 依赖: 无
// =============================================================================

package po

import "testing"

func TestParsePriDemand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in          string
		wantRank    int
		wantLabel   string
	}{
		{in: "", wantRank: PriRankUnknown, wantLabel: ""},
		{in: "   ", wantRank: PriRankUnknown, wantLabel: ""},
		{in: "1", wantRank: 1, wantLabel: "P1"},
		{in: "2", wantRank: 2, wantLabel: "P2"},
		{in: "3", wantRank: 3, wantLabel: "P3"},
		{in: "4", wantRank: 4, wantLabel: "P4"},
		{in: "0", wantRank: PriRankUnknown, wantLabel: ""},
		{in: "5", wantRank: PriRankUnknown, wantLabel: ""},
		{in: "PX", wantRank: PriRankUnknown, wantLabel: ""},
	}
	for _, tc := range cases {
		r, l := ParsePriDemand(tc.in)
		if r != tc.wantRank || l != tc.wantLabel {
			t.Errorf("ParsePriDemand(%q)=(%d,%q) want (%d,%q)", tc.in, r, l, tc.wantRank, tc.wantLabel)
		}
	}
}

func TestParsePriStory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in        int
		wantRank  int
		wantLabel string
	}{
		{in: 1, wantRank: 1, wantLabel: "P1"},
		{in: 3, wantRank: 3, wantLabel: "P3"},
		{in: 0, wantRank: PriRankUnknown, wantLabel: ""},
		{in: 5, wantRank: PriRankUnknown, wantLabel: ""},
		{in: -1, wantRank: PriRankUnknown, wantLabel: ""},
	}
	for _, tc := range cases {
		r, l := ParsePriStory(tc.in)
		if r != tc.wantRank || l != tc.wantLabel {
			t.Errorf("ParsePriStory(%d)=(%d,%q) want (%d,%q)", tc.in, r, l, tc.wantRank, tc.wantLabel)
		}
	}
}
