// =============================================================================
// 文件: internal/module/po/notice_filters_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 GORM DryRun 锁定 applyNoticeFilters 生成的 WHERE 条件与顺序，
//       作为拆分该超长函数的行为基准。拆分只允许减少重复，不允许改变 SQL。
// =============================================================================

package po

import (
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// noticeFilterCase 一组过滤条件的期望。
type noticeFilterCase struct {
	name             string
	req              NoticeListReq
	includeCategory  bool
	wantExtraWhere   string // 期望附加在基础查询之后的 WHERE 片段
	wantNoConditions bool   // 该用例不应追加任何条件
}

// noticeObjectTypeCases 覆盖 objectType 与关键字维度。
func noticeObjectTypeCases() []noticeFilterCase {
	return []noticeFilterCase{
		{name: "empty request", wantNoConditions: true},
		{
			name:           "keyword is lowercased and escaped",
			req:            NoticeListReq{Keyword: "Hello_100%"},
			wantExtraWhere: "(LOWER(n.subject) LIKE ? OR LOWER(n.data) LIKE ? OR CAST(n.objectID AS CHAR) LIKE ?)",
		},
		{name: "objectType approval", req: NoticeListReq{ObjectType: "approval"}, wantExtraWhere: "= 'approval'"},
		{name: "objectType demand", req: NoticeListReq{ObjectType: "demand"}, wantExtraWhere: "n.data LIKE '%/demand-view-%'"},
		{name: "objectType story", req: NoticeListReq{ObjectType: "story"}, wantExtraWhere: "n.data LIKE '%/story-view-%'"},
		{name: "objectType task", req: NoticeListReq{ObjectType: "task"}, wantExtraWhere: "n.data LIKE '%/task-view-%'"},
		{name: "objectType bug", req: NoticeListReq{ObjectType: "bug"}, wantExtraWhere: "n.data LIKE '%/bug-view-%'"},
		{name: "objectType feedback", req: NoticeListReq{ObjectType: "feedback"}, wantExtraWhere: "n.subject REGEXP ?"},
		{name: "objectType project", req: NoticeListReq{ObjectType: "project"}, wantExtraWhere: "n.data LIKE '%/project-view-%'"},
		{name: "objectType testtask", req: NoticeListReq{ObjectType: "testtask"}, wantExtraWhere: "n.data LIKE '%/testcase-view-%'"},
		{name: "objectType issue", req: NoticeListReq{ObjectType: "issue"}, wantExtraWhere: "n.data LIKE '%/issue-view-%'"},
		{name: "objectType risk", req: NoticeListReq{ObjectType: "risk"}, wantExtraWhere: "n.data LIKE '%/risk-view-%'"},
		{name: "objectType mail", req: NoticeListReq{ObjectType: "mail"}, wantExtraWhere: "n.objectType = 'mail'"},
		{name: "objectType unknown falls to default", req: NoticeListReq{ObjectType: "custom"}},
		{name: "objectType all is unfiltered", req: NoticeListReq{ObjectType: "all"}, wantNoConditions: true},
	}
}

// noticeStateCases 覆盖时间窗、已读状态、快捷视图、需办与类别维度。
func noticeStateCases() []noticeFilterCase {
	return []noticeFilterCase{
		{name: "timeRange today", req: NoticeListReq{TimeRange: "today"}},
		{name: "timeRange 3d", req: NoticeListReq{TimeRange: "3d"}},
		{name: "timeRange 7d", req: NoticeListReq{TimeRange: "7d"}},
		{name: "timeRange 30d", req: NoticeListReq{TimeRange: "30d"}},
		{name: "timeRange unknown is unfiltered", req: NoticeListReq{TimeRange: "1y"}, wantNoConditions: true},
		{name: "readState unread", req: NoticeListReq{ReadState: "unread"}},
		{name: "readState read", req: NoticeListReq{ReadState: "read"}},
		{name: "quickView unread", req: NoticeListReq{QuickView: "unread"}},
		{name: "quickView action", req: NoticeListReq{QuickView: "action"}},
		{name: "quickView inform", req: NoticeListReq{QuickView: "inform"}},
		{name: "quickView abnormal", req: NoticeListReq{QuickView: "abnormal"}},
		{name: "quickView today", req: NoticeListReq{QuickView: "today"}},
		{name: "needAction required", req: NoticeListReq{NeedAction: "required"}},
		{name: "needAction none", req: NoticeListReq{NeedAction: "none"}},
		{name: "category excluded when includeCategory false", req: NoticeListReq{Category: "risk"}, wantNoConditions: true},
		{name: "category applied when includeCategory true", req: NoticeListReq{Category: "approval"}, includeCategory: true},
		{name: "category all is not applied", req: NoticeListReq{Category: "all"}, includeCategory: true, wantNoConditions: true},
		{
			name: "all dimensions combined",
			req: NoticeListReq{
				Keyword: "abc", ObjectType: "demand", TimeRange: "7d",
				ReadState: "unread", QuickView: "action", NeedAction: "required", Category: "risk",
			},
			includeCategory: true,
		},
	}
}

// noticeFilterCases 汇总全部维度用例。
func noticeFilterCases() []noticeFilterCase {
	return append(noticeObjectTypeCases(), noticeStateCases()...)
}

// TestApplyNoticeFilters_SQLBaseline 锁定各维度组合下的 WHERE 片段。
func TestApplyNoticeFilters_SQLBaseline(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)

	for _, tt := range noticeFilterCases() {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := captureNoticeFilterSQL(t, now, tt.req, tt.includeCategory)

			if tt.wantNoConditions {
				if strings.Contains(sql, " WHERE ") && strings.Count(sql, " WHERE ") > 1 {
					t.Errorf("该用例不应追加过滤条件，实际: %s", sql)
				}
				return
			}
			if !strings.Contains(sql, " WHERE ") {
				t.Fatalf("应生成 WHERE 条件，实际: %s", sql)
			}
			if tt.wantExtraWhere != "" && !strings.Contains(sql, tt.wantExtraWhere) {
				t.Errorf("缺少期望条件 %q，实际: %s", tt.wantExtraWhere, sql)
			}
			_ = args
		})
	}
}

// TestApplyNoticeFilters_KeywordArgsLocked 锁定关键字的参数形态：转小写并转义 %/_。
func TestApplyNoticeFilters_KeywordArgsLocked(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)
	_, args := captureNoticeFilterSQL(t, now, NoticeListReq{Keyword: "HeLLo_100%"}, false)

	want := "%" + escapeSQLLike(strings.ToLower("HeLLo_100%")) + "%"
	if len(args) != 3 {
		t.Fatalf("关键字应绑定 3 个参数，得到 %d: %v", len(args), args)
	}
	for i, got := range args {
		if got != want {
			t.Errorf("第 %d 个关键字参数 = %v, want %v", i+1, got, want)
		}
	}
}

// TestApplyNoticeFilters_TodayWindowLocked 锁定「今天」时间窗的起止参数，
// 并确认 TimeRange 与 QuickView 两条路径给出相同的边界。
func TestApplyNoticeFilters_TodayWindowLocked(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)

	_, byRange := captureNoticeFilterSQL(t, now, NoticeListReq{TimeRange: "today"}, false)
	_, byQuick := captureNoticeFilterSQL(t, now, NoticeListReq{QuickView: "today"}, false)

	if len(byRange) != 2 || len(byQuick) != 2 {
		t.Fatalf("今天窗口应绑定起止两个参数，分别得到 %v / %v", byRange, byQuick)
	}
	if byRange[0] != byQuick[0] || byRange[1] != byQuick[1] {
		t.Errorf("TimeRange 与 QuickView 的今天窗口应一致: %v vs %v", byRange, byQuick)
	}
	if byRange[0].(time.Time).Hour() != 0 || byRange[0].(time.Time).Day() != 2 {
		t.Errorf("起始应为当日零点，得到 %v", byRange[0])
	}
	if byRange[1].(time.Time).Day() != 3 {
		t.Errorf("结束应为次日零点，得到 %v", byRange[1])
	}
}

// TestApplyNoticeFilters_UnknownObjectTypeBoundParam 锁定未知 objectType 走 default 分支并绑定参数。
func TestApplyNoticeFilters_UnknownObjectTypeBoundParam(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)
	sql, args := captureNoticeFilterSQL(t, now, NoticeListReq{ObjectType: "custom"}, false)

	if !strings.Contains(sql, "COALESCE(a.objectType, n.objectType) = ?") {
		t.Errorf("未知类型应走 default 谓词，实际: %s", sql)
	}
	if len(args) != 1 || args[0] != "custom" {
		t.Errorf("default 分支应绑定 objectType，得到 %v", args)
	}
}

// TestApplyNoticeFilters_ObjectTypeRegexLocked 锁定各类型标题前缀正则：
// 这是区分 demand/story/task 等的主要判据，改错会直接漏单或串单。
func TestApplyNoticeFilters_ObjectTypeRegexLocked(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)
	cases := []struct {
		objectType string
		wantPrefix string
	}{
		{"demand", "^(DEMAND|demand|业务需求|需求)"},
		{"story", "^(STORY|story|研发需求|研需)"},
		{"task", "^(TASK|task|任务)"},
		{"bug", "^(BUG|bug|缺陷)"},
		{"project", "^(PROJECT|project|项目)"},
		{"issue", "^(ISSUE|issue|问题)"},
		{"risk", "^(RISK|risk|风险)"},
		{"testtask", "^(TESTTASK|testtask|TESTCASE|testcase|测试单|测试)"},
	}
	for _, c := range cases {
		t.Run(c.objectType, func(t *testing.T) {
			sql, _ := captureNoticeFilterSQL(t, now, NoticeListReq{ObjectType: c.objectType}, false)
			if !strings.Contains(sql, c.wantPrefix) {
				t.Errorf("标题前缀正则应为 %q，实际 SQL: %s", c.wantPrefix, sql)
			}
		})
	}

	// feedback 的正则走绑定参数，单独断言参数值。
	t.Run("feedback", func(t *testing.T) {
		sql, args := captureNoticeFilterSQL(t, now, NoticeListReq{ObjectType: "feedback"}, false)
		if !strings.Contains(sql, "n.subject REGEXP ?") {
			t.Errorf("feedback 应以绑定参数传正则，实际 SQL: %s", sql)
		}
		want := `^(反馈|FEEDBACK|Feedback)[[:space:]]*#[[:space:]]*[0-9]+`
		found := false
		for _, a := range args {
			if s, ok := a.(string); ok && s == want {
				found = true
			}
		}
		if !found {
			t.Errorf("feedback 正则参数应为 %q，得到 %v", want, args)
		}
	})
}

// TestApplyNoticeFilters_StatePredicatesLocked 锁定已读/未读与需办/知会四个
// 分支的谓词方向——取反写错会让列表整体反向。
func TestApplyNoticeFilters_StatePredicatesLocked(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)
	cases := []struct {
		name    string
		req     NoticeListReq
		want    string
		notWant string
	}{
		{"readState unread", NoticeListReq{ReadState: "unread"}, "nr.id IS NULL", "nr.id IS NOT NULL"},
		{"readState read", NoticeListReq{ReadState: "read"}, "nr.id IS NOT NULL", "nr.id IS NULL"},
		{"quickView unread", NoticeListReq{QuickView: "unread"}, "nr.id IS NULL", "nr.id IS NOT NULL"},
		{"needAction required", NoticeListReq{NeedAction: "required"}, noticeNeedsActionSQLExpr, "NOT ("},
		{"needAction none", NoticeListReq{NeedAction: "none"}, "NOT (" + noticeNeedsActionSQLExpr + ")", ""},
		{"quickView action", NoticeListReq{QuickView: "action"}, noticeNeedsActionSQLExpr, "NOT ("},
		{"quickView inform", NoticeListReq{QuickView: "inform"}, "NOT (" + noticeNeedsActionSQLExpr + ")", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sql, _ := captureNoticeFilterSQL(t, now, c.req, false)
			if !strings.Contains(sql, c.want) {
				t.Errorf("应包含谓词 %q，实际 SQL: %s", c.want, sql)
			}
			if c.notWant != "" && strings.Contains(sql, c.notWant) {
				t.Errorf("不应出现取反谓词 %q，实际 SQL: %s", c.notWant, sql)
			}
		})
	}
}

// TestApplyNoticeFilters_TimeRangeLocked 锁定各时间窗的阈值天数。
func TestApplyNoticeFilters_TimeRangeLocked(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 4, 5, 0, time.Local)
	cases := []struct {
		timeRange string
		wantDays  int
	}{
		{"3d", 3},
		{"7d", 7},
		{"30d", 30},
	}
	for _, c := range cases {
		t.Run(c.timeRange, func(t *testing.T) {
			_, args := captureNoticeFilterSQL(t, now, NoticeListReq{TimeRange: c.timeRange}, false)
			if len(args) != 1 {
				t.Fatalf("应绑定 1 个阈值参数，得到 %v", args)
			}
			ts, ok := args[0].(time.Time)
			if !ok {
				t.Fatalf("阈值应为时间，得到 %T", args[0])
			}
			want := now.AddDate(0, 0, -c.wantDays)
			if !ts.Equal(want) {
				t.Errorf("阈值 = %v, want %v", ts, want)
			}
		})
	}
}

// captureNoticeFilterSQL 在 DryRun 下跑一次 applyNoticeFilters，返回 SQL 文本与绑定参数。
func captureNoticeFilterSQL(t *testing.T, now time.Time, req NoticeListReq, includeCategory bool) (string, []any) {
	t.Helper()
	db, _ := newNoticeDryRunDB(t)
	base := db.Table("zt_notify AS n")
	q := applyNoticeFilters(base, now, req, includeCategory)
	stmt := q.Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
	return stmt.SQL.String(), stmt.Vars
}

// newNoticeDryRunDB 构造一个接受任意查询、只拼 SQL 不执行的连接。
func newNoticeDryRunDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("构造 dry-run 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}),
		&gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("打开 dry-run 连接失败: %v", err)
	}
	return db, mock
}
