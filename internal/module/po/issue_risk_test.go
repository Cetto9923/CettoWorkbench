// =============================================================================
// 文件: internal/module/po/issue_risk_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 /issues/risk 表单校验、未关闭/逾期状态集与服务层映射一致性。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
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

func TestIssueRiskListReqValidateTeamScope(t *testing.T) {
	valid := IssueRiskListReq{Scope: "team", ScopeID: 11, Relation: "allRelated"}
	if errs := valid.Validate(); errs != nil {
		t.Fatalf("team scope should validate: %v", errs)
	}
	for _, req := range []IssueRiskListReq{
		{Scope: "other", ScopeID: 11},
		{ScopeID: 11},
		{Scope: "team", ScopeID: 11, Relation: "myAction"},
	} {
		if errs := req.Validate(); len(errs) == 0 {
			t.Errorf("invalid team scope request should be rejected: %+v", req)
		}
	}
}

func TestFindIssueRiskListUsesAuthorizedTeamMembers(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	openStatuses := []driver.Value{"active", "tracked", "wait", "unconfirmed", "doing", "confirmed"}
	countArgs := []driver.Value{"dev1", "dev2", "dev1", "dev2"}
	countArgs = append(countArgs, openStatuses...)
	mock.ExpectQuery(`SELECT count\(\*\) FROM zt_issue AS i.*i\.createdBy IN \(\?,\?\).*i\.assignedTo IN \(\?,\?\)`).
		WithArgs(countArgs...).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	selectArgs := append([]driver.Value{"dev1", "dev2", "dev1", "dev2"}, openStatuses...)
	selectArgs = append(selectArgs, 20)
	mock.ExpectQuery(`(?s)SELECT i\.id AS id.*FROM zt_issue AS i.*i\.createdBy IN \(\?,\?\).*i\.assignedTo IN \(\?,\?\).*ORDER BY i\.id DESC LIMIT \?`).
		WithArgs(selectArgs...).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	req := IssueRiskListReq{Kind: "issue", Relation: "allRelated", Loop: "open", Page: 1, PageSize: 20, Scope: "team", teamAccounts: []string{"dev1", "dev2"}}
	_, _, err := repo.FindIssueRiskList(context.Background(), "coach1", req)
	if err != nil {
		t.Fatalf("FindIssueRiskList() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestHomeIssueRiskCountsUseMyOpenIssuesAndRisks(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	svc := NewService(repo, nil, nil, nil)
	openStatuses := []driver.Value{"active", "tracked", "wait", "unconfirmed", "doing", "confirmed"}
	issueArgs := []driver.Value{"alice", "alice"}
	issueArgs = append(issueArgs, openStatuses...)
	mock.ExpectQuery(`SELECT count\(\*\) FROM zt_issue AS i`).
		WithArgs(issueArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	issueSelectArgs := append([]driver.Value{"alice", "alice"}, openStatuses...)
	issueSelectArgs = append(issueSelectArgs, 1)
	mock.ExpectQuery(`(?s)SELECT i\.id AS id.*FROM zt_issue AS i`).
		WithArgs(issueSelectArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	riskArgs := []driver.Value{"alice", "alice"}
	riskArgs = append(riskArgs, openStatuses...)
	mock.ExpectQuery(`SELECT count\(\*\) FROM zt_risk AS k`).
		WithArgs(riskArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	riskSelectArgs := append([]driver.Value{"alice", "alice"}, openStatuses...)
	riskSelectArgs = append(riskSelectArgs, 1)
	mock.ExpectQuery(`(?s)SELECT k\.id AS id.*FROM zt_risk AS k`).
		WithArgs(riskSelectArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	counts, err := svc.HomeIssueRiskCounts(context.Background(), &model.User{Account: "alice"})
	if err != nil {
		t.Fatalf("HomeIssueRiskCounts() error = %v", err)
	}
	if counts.Issues != 3 || counts.Risks != 2 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
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
		{"4", "轻微", issueRiskSeverity},
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

func TestIssueRiskOverdueLocalCalendarDay(t *testing.T) {
	originalNow := issueRiskNow
	t.Cleanup(func() { issueRiskNow = originalNow })
	cases := []struct {
		name, plan, status string
		hour               int
		wantOverdue        bool
		wantDays           int
	}{
		{"跨零点", "2026-09-30", "active", 0, true, 1},
		{"当天", "2026-10-01", "active", 0, false, 0},
		{"过去多天", "2026-09-27", "active", 0, true, 4},
		{"空日期", "", "active", 0, false, 0},
		{"已关闭", "2026-09-30", "closed", 0, false, 0},
		{"当天晚间", "2026-10-01", "active", 23, false, 0},
	}
	for _, tc := range cases {
		issueRiskNow = func() time.Time { return time.Date(2026, 10, 1, tc.hour, 30, 0, 0, time.Local) }
		item := mapIssueRiskRow(issueRiskRow{PlanDate: tc.plan, Status: tc.status}, "risk")
		if item.IsOverdue != tc.wantOverdue || item.OverdueDays != tc.wantDays {
			t.Fatalf("%s: overdue=%t/%d, want %t/%d", tc.name, item.IsOverdue, item.OverdueDays, tc.wantOverdue, tc.wantDays)
		}
	}
}
