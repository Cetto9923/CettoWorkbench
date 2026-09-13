package demandstage

import "testing"

func TestMapDevelopingIsSubmitTest(t *testing.T) {
	code, label := Map("developing", "")
	if code != "submittest" || label != "提测" {
		t.Fatalf("Map(developing,\"\") = (%q,%q), want (submittest,提测)", code, label)
	}
	code, label = Map("", "developing")
	if code != "submittest" || label != "提测" {
		t.Fatalf("Map(\"\",developing) = (%q,%q), want (submittest,提测)", code, label)
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
		code, label := Map(tc.stage, tc.status)
		if code != tc.wantCode || label != tc.wantLabel {
			t.Errorf("Map(%q,%q) = (%q,%q), want (%q,%q)",
				tc.stage, tc.status, code, label, tc.wantCode, tc.wantLabel)
		}
		if got := Label(tc.stage, tc.status); got != tc.wantLabel {
			t.Errorf("Label(%q,%q) = %q, want %q", tc.stage, tc.status, got, tc.wantLabel)
		}
	}
}
