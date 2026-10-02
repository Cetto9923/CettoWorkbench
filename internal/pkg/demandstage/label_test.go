package demandstage

import "testing"

// Map 只产出首页价值流编码，不再返回中文名；developing 必须原样返回，
// 否则详情页「研发」节点永远不会成为当前阶段。
func TestMapDevelopingStaysDeveloping(t *testing.T) {
	if got := Map("developing", ""); got != "developing" {
		t.Fatalf("Map(developing,\"\") = %q, want developing", got)
	}
	if got := Map("", "developing"); got != "developing" {
		t.Fatalf("Map(\"\",developing) = %q, want developing", got)
	}
}

func TestLabelDevelopingIsRDInProgress(t *testing.T) {
	if got := Label("developing", ""); got != "研发中" {
		t.Fatalf("Label(developing,\"\") = %q, want 研发中", got)
	}
	if got := Label("", "developing"); got != "研发中" {
		t.Fatalf("Label(\"\",developing) = %q, want 研发中", got)
	}
}

// Map 与 Label 在非 developing 分支上仍同源，只在 developing 上有意分叉。
func TestMapAndLabelShareCommonCases(t *testing.T) {
	cases := []struct {
		stage, status, wantCode, wantLabel string
	}{
		{"", "closed", "closed", "已关闭"},
		{"", "testing", "testing", "测试中"},
		{"", "clarified", "schedule", "已排期"},
		{"wait", "", "accept", "已受理"},
		{"delivered", "", "publish", "发布"},
	}
	for _, tc := range cases {
		if code := Map(tc.stage, tc.status); code != tc.wantCode {
			t.Errorf("Map(%q,%q) = %q, want %q", tc.stage, tc.status, code, tc.wantCode)
		}
		if got := Label(tc.stage, tc.status); got != tc.wantLabel {
			t.Errorf("Label(%q,%q) = %q, want %q", tc.stage, tc.status, got, tc.wantLabel)
		}
	}
}

// 验收在首页是两个独立阶段，Map 不得再合并成单个编码。
func TestMapAcceptanceNotMerged(t *testing.T) {
	if got := Map("", "waitacceptance"); got != "waitacceptance" {
		t.Errorf("Map(waitacceptance) = %q, want waitacceptance", got)
	}
	if got := Map("", "acceptanced"); got != "acceptanced" {
		t.Errorf("Map(acceptanced) = %q, want acceptanced", got)
	}
	if got := Map("", "released"); got != "released" {
		t.Errorf("Map(released) = %q, want released", got)
	}
}
