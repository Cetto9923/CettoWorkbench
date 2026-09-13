// =============================================================================
// 文件: internal/module/po/issue_risk_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 /issues/risk 表单校验、未关闭/逾期状态集与服务层映射一致性。
// 依赖: 无
// =============================================================================

package po

import (
	"testing"
)

func TestIssueRiskListReqValidateDefaults(t *testing.T) {
	r := IssueRiskListReq{}
	if errs := r.Validate(); errs != nil {
		t.Fatalf("expected default Validate to pass, got %v", errs)
	}
	if r.Kind != "issue" {
		t.Errorf("expected default kind=issue, got %q", r.Kind)
	}
	if r.Relation != "allRelated" {
		t.Errorf("expected default relation=allRelated, got %q", r.Relation)
	}
	if r.Loop != "open" {
		t.Errorf("expected default loop=open, got %q", r.Loop)
	}
	if r.Page != 1 || r.PageSize != 20 {
		t.Errorf("expected default pagination (1,20), got (%d,%d)", r.Page, r.PageSize)
	}
	if r.Overdue {
		t.Error("expected overdue default false")
	}
}

func TestIssueRiskListReqValidateLoop(t *testing.T) {
	for _, ok := range []string{"all", "open", "closed", "  open  "} {
		r := IssueRiskListReq{Loop: ok}
		if errs := r.Validate(); errs != nil {
			t.Errorf("loop=%q expected pass, got %v", ok, errs)
		}
	}
	r := IssueRiskListReq{Loop: "weird"}
	if len(r.Validate()) == 0 {
		t.Error("expected loop=weird to fail")
	}
	r = IssueRiskListReq{Loop: "Open"}
	if len(r.Validate()) == 0 {
		t.Error("expected loop=Open (case sensitive) to fail")
	}
}

func TestIssueRiskListReqValidateOverdueOnlyWithOpen(t *testing.T) {
	r := IssueRiskListReq{Loop: "open", Overdue: true}
	if errs := r.Validate(); errs != nil {
		t.Errorf("expected overdue with loop=open to pass, got %v", errs)
	}
	for _, loop := range []string{"all", "closed"} {
		r := IssueRiskListReq{Loop: loop, Overdue: true}
		if errs := r.Validate(); len(errs) == 0 {
			t.Errorf("expected overdue with loop=%s to fail", loop)
		}
	}
}

func TestIssueRiskLoopStatusSet(t *testing.T) {
	wantOpen := map[string]bool{
		"active": true, "tracked": true, "wait": true,
		"unconfirmed": true, "doing": true, "confirmed": true,
	}
	for _, s := range irOpenStatuses {
		if !wantOpen[s] {
			t.Errorf("irOpenStatuses contains unexpected status %q", s)
		}
	}
	wantClosed := map[string]bool{
		"resolved": true, "closed": true, "cancel": true, "canceled": true,
	}
	for _, s := range irClosedStatuses {
		if !wantClosed[s] {
			t.Errorf("irClosedStatuses contains unexpected status %q", s)
		}
	}
	// open 与 closed 必须互斥且穷尽 (现有 helper issueRiskClosed 对 kind=risk 不包含 resolved)
	overlap := false
	for _, s := range irOpenStatuses {
		for _, c := range irClosedStatuses {
			if s == c {
				overlap = true
				t.Errorf("status %q appears in both open and closed", s)
			}
		}
	}
	if overlap {
		t.Fatal("open/closed sets overlap")
	}
}

func TestIssueRiskSeverityAndPriorityLabels(t *testing.T) {
	cases := []struct {
		raw   string
		want  string
		sevFn func(string) string
	}{
		{"1", "致命", issueRiskSeverity},
		{"2", "严重", issueRiskSeverity},
		{"3", "一般", issueRiskSeverity},
		{"", "一般", issueRiskSeverity},
		{"未知", "一般", issueRiskSeverity},
	}
	for _, tc := range cases {
		if got := tc.sevFn(tc.raw); got != tc.want {
			t.Errorf("issueRiskSeverity(%q)=%q want %q", tc.raw, got, tc.want)
		}
	}
	priCases := []struct {
		raw  string
		want string
	}{
		{"1", "P1"},
		{"4", "P4"},
		{"  2 ", "P2"},
		{"", "-"},
	}
	for _, tc := range priCases {
		if got := priorityLabel(tc.raw); got != tc.want {
			t.Errorf("priorityLabel(%q)=%q want %q", tc.raw, got, tc.want)
		}
	}
}

func TestIssueRiskStatusLabel(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"active", "激活"},
		{"doing", "处理中"},
		{"resolved", "已解决"},
		{"closed", "已关闭"},
		{"unknown", "未处理"},
	}
	for _, tc := range cases {
		if got := issueRiskStatusLabel(tc.raw); got != tc.want {
			t.Errorf("issueRiskStatusLabel(%q)=%q want %q", tc.raw, got, tc.want)
		}
	}
}

func TestIssueRiskClosedHelper(t *testing.T) {
	closedIssue := []string{"resolved", "closed", "cancel", "canceled"}
	for _, s := range closedIssue {
		if !issueRiskClosed("issue", s) {
			t.Errorf("issue should be closed at %q", s)
		}
	}
	openIssue := []string{"active", "wait", "doing", "confirmed", "tracked"}
	for _, s := range openIssue {
		if issueRiskClosed("issue", s) {
			t.Errorf("issue should NOT be closed at %q", s)
		}
	}
	if !issueRiskClosed("risk", "closed") {
		t.Error("risk should be closed at closed")
	}
	if issueRiskClosed("risk", "resolved") {
		t.Error("risk should NOT be closed at resolved per existing helper")
	}
}
