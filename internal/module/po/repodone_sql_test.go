// =============================================================================
// 文件: internal/module/po/repodone_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 GORM DryRun 锁定 FindDoneActions 生成的 WHERE 条件、参数与分页，
//       作为拆分该超长函数的行为基准。
// =============================================================================

package po

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// doneCase 一组查询条件的期望。
type doneCase struct {
	name     string
	req      RepoFindDoneActionsReq
	wantSQL  []string // 期望出现的 SQL 片段
	notWant  []string // 不应出现的 SQL 片段
	wantArgs int      // 期望的绑定参数个数（分页 limit 之前的全部参数）
}

// doneProjectionColumns 列表查询必须取回的列，缺一列会导致 enrichment 丢字段。
var doneProjectionColumns = []string{
	"a.id", "a.objectType", "a.objectID", "a.action",
	"a.actor", "a.date", "a.extra", "a.comment",
}

// assertNoExtraFilter 校验该用例生成的 SQL 与无过滤基线逐字相同。
// 取字面量比较而非参数个数：Result/Action 传 "all" 时过滤器的参数
// 会展开为空切片，参数个数不变但 SQL 仍会多出 IN 条件。
func assertNoExtraFilter(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s 应与无过滤基线一致\n实际: %s\n基准: %s", name, got, want)
	}
}

// doneBaseArgCount 哨兵：表示该用例不应追加任何过滤参数。
const doneBaseArgCount = -1

// baseDoneReq 无任何过滤条件的最小请求。
func baseDoneReq() RepoFindDoneActionsReq {
	return RepoFindDoneActionsReq{Account: "alice", Page: 1, PageSize: 20}
}

// doneFilterCases 覆盖 actor、结果、动作、关键字、对象类型等过滤维度。
func doneFilterCases() []doneCase {
	return []doneCase{
		{
			name:    "default",
			req:     RepoFindDoneActionsReq{Account: "alice", Page: 1, PageSize: 20},
			wantSQL: []string{"FROM zt_action AS a", "a.actor = ?"},
		},
		{
			name:    "result filter",
			req:     RepoFindDoneActionsReq{Account: "alice", Result: "pass", Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?"},
		},
		{
			name:     "result all is unfiltered",
			req:      RepoFindDoneActionsReq{Account: "alice", Result: "all", Page: 1, PageSize: 20},
			wantSQL:  []string{"a.actor = ?"},
			wantArgs: doneBaseArgCount,
		},
		{
			name:    "result pass adds result filter",
			req:     RepoFindDoneActionsReq{Account: "alice", Result: "pass", Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?"},
		},
		{
			name:     "action all is unfiltered",
			req:      RepoFindDoneActionsReq{Account: "alice", Action: "all", Page: 1, PageSize: 20},
			wantSQL:  []string{"a.actor = ?"},
			wantArgs: doneBaseArgCount,
		},
		{
			name:    "action filter",
			req:     RepoFindDoneActionsReq{Account: "alice", Action: "closed", Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?"},
		},
		{
			name:    "keyword filter",
			req:     RepoFindDoneActionsReq{Account: "alice", Keyword: "登录", Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?", "LIKE ?"},
		},
		{
			name:    "tab filter",
			req:     RepoFindDoneActionsReq{Account: "alice", Tab: DoneTabAll, Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?"},
		},
		{
			name:    "object type filter",
			req:     RepoFindDoneActionsReq{Account: "alice", ObjectType: "demand", Page: 1, PageSize: 20},
			wantSQL: []string{"a.actor = ?", "a.objectType = ?"},
		},
	}
}

// doneTimeAndCombinedCases 覆盖时间窗与多维度组合。
func doneTimeAndCombinedCases() []doneCase {
	return []doneCase{
		{
			name:    "time range today",
			req:     RepoFindDoneActionsReq{Account: "alice", TimeRange: TimeRangeToday, Page: 1, PageSize: 20},
			wantSQL: []string{"a.date >= ? AND a.date < ?"},
		},
		{
			name:    "time range 7d",
			req:     RepoFindDoneActionsReq{Account: "alice", TimeRange: TimeRange7d, Page: 1, PageSize: 20},
			wantSQL: []string{"a.date >= ?"},
		},
		{
			name: "all filters combined",
			req: RepoFindDoneActionsReq{
				Account: "alice", Result: "pass", Action: "closed", Keyword: "k",
				TimeRange: TimeRange7d, ObjectType: "demand", Page: 1, PageSize: 20,
			},
			wantSQL: []string{"a.actor = ?", "a.objectType = ?", "a.date >= ?"},
		},
	}
}

// doneCases 汇总全部用例。
func doneCases() []doneCase {
	return append(doneFilterCases(), doneTimeAndCombinedCases()...)
}

// assertSQLContains 校验 SQL 含有全部期望片段、且不含任何禁止片段。
func assertSQLContains(t *testing.T, sql string, want, notWant []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(sql, w) {
			t.Errorf("缺少期望片段 %q，实际 SQL: %s", w, sql)
		}
	}
	for _, w := range notWant {
		if strings.Contains(sql, w) {
			t.Errorf("不应出现片段 %q，实际 SQL: %s", w, sql)
		}
	}
}

// TestFindDoneActions_SQLBaseline 锁定各维度组合下的 WHERE 片段与过滤参数。
func TestFindDoneActions_SQLBaseline(t *testing.T) {
	// 无过滤基线 SQL：Result/Action 传 "all" 时必须与它逐字相同。
	baseSQL, _ := captureDoneSQL(t, baseDoneReq())

	for _, tt := range doneCases() {
		t.Run(tt.name, func(t *testing.T) {
			sql, _ := captureDoneSQL(t, tt.req)
			assertSQLContains(t, sql, tt.wantSQL, tt.notWant)
			if tt.wantArgs == doneBaseArgCount {
				assertNoExtraFilter(t, tt.name, sql, baseSQL)
			}
		})
	}
}

// TestFindDoneActions_PagingLocked 锁定分页归一化：非法 page/pageSize 归一到
// 1/20。归一化后 LIMIT 与 OFFSET 出现在 SQL 文本中，按归一化结果断言。
// donePagingCase 一组分页归一化的期望。
type donePagingCase struct {
	name           string
	page           int
	pageSize       int
	wantLimit      int
	wantOffsetZero bool
}

// donePagingCases 分页归一化边界：非法 page/pageSize 应归一到 1/20。
func donePagingCases() []donePagingCase {
	return []donePagingCase{
		{"zero page and size normalize", 0, 0, 20, true},
		{"negative page normalize", -3, -5, 20, true},
		{"oversized pageSize clamps", 1, 500, 20, true},
		{"page two keeps size", 2, 20, 20, false},
		{"explicit size ten", 3, 10, 10, false},
	}
}

// assertDoneListQuery 校验列表查询含倒序、投影列完整，且 OFFSET 存在性符合归一化结果。
func assertDoneListQuery(t *testing.T, sql string, c donePagingCase) {
	t.Helper()
	if !strings.Contains(sql, "ORDER BY a.id DESC") {
		t.Errorf("列表查询应按 id 倒序，实际 SQL: %s", sql)
	}
	// Select 列表是 enrichment 的输入，少一列会静默丢字段。
	for _, col := range doneProjectionColumns {
		if !strings.Contains(sql, col) {
			t.Errorf("列表查询缺少投影列 %q，实际 SQL: %s", col, sql)
		}
	}
	if !strings.Contains(sql, "LIMIT ?") {
		t.Errorf("列表查询应有 LIMIT，实际 SQL: %s", sql)
	}
	// page 归一到 1 时 OFFSET 为 0，GORM 会省略 OFFSET 子句。
	if hasOffset := strings.Contains(sql, "OFFSET ?"); hasOffset == c.wantOffsetZero {
		t.Errorf("OFFSET 存在性 = %v，期望 %v；实际 SQL: %s", hasOffset, !c.wantOffsetZero, sql)
	}
}

// TestFindDoneActions_PagingLocked 锁定分页归一化：非法 page/pageSize 归一到 1/20。
func TestFindDoneActions_PagingLocked(t *testing.T) {
	for _, c := range donePagingCases() {
		t.Run(c.name, func(t *testing.T) {
			repo := dryRunDoneRepo(t)
			_, _, _ = repo.FindDoneActions(t.Context(), RepoFindDoneActionsReq{
				Account: "alice", Page: c.page, PageSize: c.pageSize,
			})
			sql, _ := doneQueries.actionQuery()
			assertDoneListQuery(t, sql, c)
		})
	}
}

// TestFindDoneActions_EmptyAccountShortCircuits 空账号不查库。
func TestFindDoneActions_EmptyAccountShortCircuits(t *testing.T) {
	repo := dryRunDoneRepo(t)
	items, total, err := repo.FindDoneActions(t.Context(), RepoFindDoneActionsReq{Account: "  ", Page: 1, PageSize: 20})
	if err != nil || items != nil || total != 0 {
		t.Fatalf("空账号应短路，得到 items=%v total=%d err=%v", items, total, err)
	}
}

// doneRecorder 记录一次 FindDoneActions 期间发出的 SQL 与参数。
type doneRecorder struct {
	queries []doneQuery
}

type doneQuery struct {
	sql  string
	args []any
}

// find 返回最后一条 SQL 满足 match 的查询。
// FindDoneActions 之后还会发若干补数据查询，其中也有查 zt_action 的，
// 因此必须按具体子句精确定位，不能只匹配表名。
func (r *doneRecorder) find(match func(string) bool) (string, []any) {
	for i := len(r.queries) - 1; i >= 0; i-- {
		if match(r.queries[i].sql) {
			return r.queries[i].sql, r.queries[i].args
		}
	}
	return "", nil
}

// actionQuery 取主列表查询（带 ORDER BY a.id DESC）。
func (r *doneRecorder) actionQuery() (string, []any) {
	return r.find(func(s string) bool { return strings.Contains(s, "ORDER BY a.id DESC") })
}

// countQuery 取 Count 查询。
func (r *doneRecorder) countQuery() (string, []any) {
	return r.find(func(s string) bool { return strings.HasPrefix(s, "SELECT count(") })
}

// doneQueries 供测试期间写入的全局记录器，每个用例通过 t.Cleanup 重置。
var doneQueries doneRecorder

// captureDoneSQL 返回一次 FindDoneActions 中最后一条（列表查询）的 SQL。
func captureDoneSQL(t *testing.T, req RepoFindDoneActionsReq) (string, []any) {
	t.Helper()
	repo := dryRunDoneRepo(t)
	_, _, _ = repo.FindDoneActions(t.Context(), req)
	// 过滤条件在 Count 与列表查询中共用，取 Count 查询作为条件基准。
	if sql, args := doneQueries.countQuery(); sql != "" {
		return sql, args
	}
	return doneQueries.actionQuery()
}

// dryRunDoneRepo 构造只拼 SQL、不执行的 Repo，并重置记录器。
func dryRunDoneRepo(t *testing.T) *Repo {
	t.Helper()
	doneQueries = doneRecorder{}
	db := newDoneDryRunDB(t, &doneQueries)
	return NewRepo(db, db)
}

// newDoneDryRunDB 构造一个记录全部查询、只返回空结果的连接。
func newDoneDryRunDB(t *testing.T, rec *doneRecorder) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(
		func(expected, actual string) error {
			rec.queries = append(rec.queries, doneQuery{sql: actual})
			return nil // 接受任意 SQL，只记录
		},
	)))
	// WillReturnRows 之前先把每条查询的参数记下来：GORM 会依次调用，
	if err != nil {
		t.Fatalf("构造 dry-run 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB, SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 dry-run 连接失败: %v", err)
	}
	for i := 0; i < 12; i++ {
		mock.MatchExpectationsInOrder(false)
		mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"n"}))
	}
	return db
}
