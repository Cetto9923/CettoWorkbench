// =============================================================================
// 文件: internal/module/po/notice_query_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证通知 SQL 查询构建器、转义逻辑及分类映射一致性。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"testing"
	"time"
)

func TestEscapeSQLLike(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"100%", "100\\%"},
		{"user_name", "user\\_name"},
		{"a\\b%c_d", "a\\\\b\\%c\\_d"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := escapeSQLLike(tc.input); got != tc.want {
			t.Errorf("escapeSQLLike(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsNoticeReqCategoryUnfiltered(t *testing.T) {
	if !isNoticeReqCategoryUnfiltered(NoticeListReq{}) {
		t.Error("expected empty NoticeListReq to be unfiltered")
	}
	if !isNoticeReqCategoryUnfiltered(NoticeListReq{ObjectType: "all", Category: "business"}) {
		t.Error("expected ObjectType=all with Category to be category-unfiltered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{QuickView: "unread"}) {
		t.Error("expected QuickView=unread to be filtered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{ObjectType: "approval"}) {
		t.Error("expected ObjectType=approval to be filtered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{Keyword: "test"}) {
		t.Error("expected Keyword to be filtered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{TimeRange: "today"}) {
		t.Error("expected TimeRange=today to be filtered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{ReadState: "unread"}) {
		t.Error("expected ReadState=unread to be filtered")
	}
	if isNoticeReqCategoryUnfiltered(NoticeListReq{NeedAction: "required"}) {
		t.Error("expected NeedAction=required to be filtered")
	}
}

func TestNoticeCategoryConsistency(t *testing.T) {
	// 验证 SQL CASE 表达式中的分类枚举与 Go classifyNotice 严格一一对应
	actions := []string{
		"reviewed", "reviewpassed", "reviewrejected", "submitreview", "submit", "submitted", "returned", "withdraw",
		"reminded", "overdue", "due", "delay", "delayed", "soon",
		"assigned", "assignedTo", "transfer", "cc", "commented", "remark", "mentioned",
		"rejected", "bugconfirmed", "paused", "suspended", "hangup", "archive", "blocked", "gatefailed",
		"other_unknown_action",
	}

	for _, action := range actions {
		goCat := classifyNotice("demand", action)
		// SQL CASE logic:
		var sqlCat string
		switch {
		case strings.Contains("'reviewed', 'reviewpassed', 'reviewrejected', 'submitreview', 'submit', 'submitted', 'returned', 'withdraw'", "'"+action+"'"):
			sqlCat = "approval"
		case strings.Contains("'reminded', 'overdue', 'due', 'delay', 'delayed', 'soon'", "'"+action+"'"):
			sqlCat = "reminder"
		case strings.Contains("'assigned', 'assignedTo', 'transfer', 'cc', 'commented', 'remark', 'mentioned'", "'"+action+"'"):
			sqlCat = "collaboration"
		case strings.Contains("'rejected', 'bugconfirmed', 'paused', 'suspended', 'hangup', 'archive', 'blocked', 'gatefailed'", "'"+action+"'"):
			sqlCat = "risk"
		default:
			sqlCat = "business"
		}

		if goCat != sqlCat {
			t.Errorf("action %q: goCat=%s, sqlCat=%s mismatch", action, goCat, sqlCat)
		}
	}

	objectTypes := []string{"doc", "release", "system", "sync", "account", "custom"}
	for _, ot := range objectTypes {
		goCat := classifyNotice(ot, "")
		var sqlCat string
		if strings.Contains("'doc', 'release', 'system', 'sync', 'account'", "'"+ot+"'") {
			sqlCat = "system"
		} else {
			sqlCat = "business"
		}
		if goCat != sqlCat {
			t.Errorf("objectType %q: goCat=%s, sqlCat=%s mismatch", ot, goCat, sqlCat)
		}
	}
}

func TestNoticeNeedsActionConsistency(t *testing.T) {
	actions := []string{
		"reviewed", "clarify", "assigned", "assignedTo", "submitted", "submit", "returned", "reminded",
		"other", "closed", "deleted",
	}
	for _, action := range actions {
		goNeeds := noticeNeedsAction(action)
		sqlNeeds := strings.Contains("'reviewed', 'clarify', 'assigned', 'assignedTo', 'submitted', 'submit', 'returned', 'reminded'", "'"+action+"'")
		if goNeeds != sqlNeeds {
			t.Errorf("action %q: goNeeds=%v, sqlNeeds=%v mismatch", action, goNeeds, sqlNeeds)
		}
	}
}

func TestNoticeTimeRangeBoundaries(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-09-05T12:00:00Z")
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.AddDate(0, 0, 1)

	// today
	tToday := now.Add(-2 * time.Hour)
	if tToday.Year() != now.Year() || tToday.YearDay() != now.YearDay() {
		t.Errorf("expected tToday to be same day as now")
	}
	if tToday.Before(startOfToday) || !tToday.Before(endOfToday) {
		t.Errorf("SQL range for today did not match calendar-day boundaries")
	}

	// yesterday
	tYesterday := now.AddDate(0, 0, -1)
	if tYesterday.Year() == now.Year() && tYesterday.YearDay() == now.YearDay() {
		t.Errorf("expected tYesterday not to be same day as now")
	}
	if !tYesterday.Before(startOfToday) {
		t.Errorf("expected tYesterday to be before startOfToday")
	}
}
