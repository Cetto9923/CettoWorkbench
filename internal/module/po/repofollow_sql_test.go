// =============================================================================
// 文件: internal/module/po/repofollow_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 SQL 基准锁定 FindFollowedDemands 的查询构成，作为拆分该超长函数的
//       行为基准。基准固定三件事：查询条数与顺序（统计 / Count / 列表 / 姓名）、
//       Count 与列表共享的过滤条件逐字一致、关键字与 watch 子句的绑定参数个数。
//       拆分只允许把代码搬到同包私有函数，不允许改 SQL、改条件顺序、改分页。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/driver/mysql, gorm.io/gorm
// =============================================================================

package po

import (
	"strings"
	"testing"
)

// isCountQuery 识别 Count 查询。
func isCountQuery(s string) bool { return strings.HasPrefix(s, "SELECT count(*)") }

// isListQuery 识别关注列表查询。
func isListQuery(s string) bool { return strings.Contains(s, "ORDER BY d.id DESC") }

// followCountQuery 取 Count 查询。
func (r *sqlBaselineRecorder) followCountQuery() string {
	if q := r.findLast(isCountQuery); q != nil {
		return q.sql
	}
	return ""
}

// followCountArgs 取 Count 查询的绑定参数。
func (r *sqlBaselineRecorder) followCountArgs() []any {
	if q := r.findLast(isCountQuery); q != nil {
		return q.args
	}
	return nil
}

// followListQuery 取列表查询（含 ORDER BY d.id DESC）。
func (r *sqlBaselineRecorder) followListQuery() string {
	if q := r.findLast(isListQuery); q != nil {
		return q.sql
	}
	return ""
}

// followListArgs 取列表查询的绑定参数。
func (r *sqlBaselineRecorder) followListArgs() []any {
	if q := r.findLast(isListQuery); q != nil {
		return q.args
	}
	return nil
}

// toInt64 把 GORM 下发的整型绑定参数转为 int64，非整型返回 -1。
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	default:
		return -1
	}
}

// followFilterCase 一组关注列表过滤条件的期望。
type followFilterCase struct {
	name      string
	req       RepoFindFollowedDemandsReq
	wantExtra []string // Count 上必须出现的附加过滤片段
	notWant   []string // Count 上不应出现的片段
	wantArgs  int      // 期望的绑定参数个数（watch 3 + deleted 1 之外附加的参数数）
}

// followBaseReq 无任何过滤的最小请求。
func followBaseReq() RepoFindFollowedDemandsReq {
	return RepoFindFollowedDemandsReq{Account: "alice", Page: 1, PageSize: 20}
}

// followScopeCases 覆盖 6 个 scope 取值（含零值默认）各自追加的条件。
func followScopeCases() []followFilterCase {
	return []followFilterCase{
		{name: "scope default is open", req: followBaseReq(),
			wantExtra: []string{"d.status <> ?"}, notWant: []string{"isNeedFocus"}, wantArgs: 1},
		{name: "scope open", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeOpen},
			wantExtra: []string{"d.status <> ?"}, notWant: []string{"isNeedFocus"}, wantArgs: 1},
		{name: "scope key", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeKey},
			wantExtra: []string{"d.isNeedFocus = ?"}, notWant: []string{"d.status <> ?"}, wantArgs: 1},
		{name: "scope keyopen", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeKeyOpen},
			wantExtra: []string{"d.isNeedFocus = ? AND d.status <> ?"}, wantArgs: 2},
		{name: "scope closed", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeClosed},
			wantExtra: []string{"d.status = ?"}, notWant: []string{"<> ?"}, wantArgs: 1},
		{name: "scope openclean", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeOpenClean},
			wantExtra: []string{"d.isNeedFocus IS NULL OR d.isNeedFocus <> ?", "d.status <> ?"}, wantArgs: 2},
		{name: "scope all adds nothing", req: RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeAll},
			notWant: []string{"isNeedFocus", "d.status <> ?", "d.status = ?"}},
	}
}

// followLifecycleCases 覆盖 4 个生命周期取值各自追加的状态集合。
func followLifecycleCases() []followFilterCase {
	return []followFilterCase{
		{name: "lifecycle clarifying", req: RepoFindFollowedDemandsReq{Account: "alice", Lifecycle: FollowLifecycleClarifying},
			wantExtra: []string{"d.status IN (?,?,?,?)"}, wantArgs: 5}, // 默认 scope 1 + 状态 4
		{name: "lifecycle implementing", req: RepoFindFollowedDemandsReq{Account: "alice", Lifecycle: FollowLifecycleImplementing},
			wantExtra: []string{"d.status IN (?,?,?,?,?,?,?)"}, wantArgs: 8}, // 默认 scope 1 + 状态 7
		{name: "lifecycle released", req: RepoFindFollowedDemandsReq{Account: "alice", Lifecycle: FollowLifecycleReleased},
			wantExtra: []string{"d.status = ?"}, wantArgs: 2}, // 默认 scope 1 + 状态 1
		{name: "lifecycle closed", req: RepoFindFollowedDemandsReq{Account: "alice", Lifecycle: FollowLifecycleClosed},
			wantExtra: []string{"d.status = ?"}, wantArgs: 2}, // 默认 scope 1 + 状态 1
	}
}

// TestFindFollowedDemands_SQLBaseline 锁定 scope 与 lifecycle 各组合下
// Count 查询追加的过滤片段与绑定参数。
func TestFindFollowedDemands_SQLBaseline(t *testing.T) {
	cases := append(followScopeCases(), followLifecycleCases()...)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, rec := newSQLBaselineRepo(t, 8)
			if _, _, _, err := repo.FindFollowedDemands(t.Context(), c.req); err != nil {
				t.Fatalf("FindFollowedDemands 返回错误: %v", err)
			}
			sql := rec.followCountQuery()
			if sql == "" {
				t.Fatalf("未捕获到 Count 查询，实际发出 %d 条", rec.count())
			}
			assertSQLContains(t, sql, c.wantExtra, c.notWant)
			// watch 子句固定绑定 3 个 account，deleted 固定 1 个。
			args := rec.followCountArgs()
			if want := 4 + c.wantArgs; len(args) != want {
				t.Errorf("Count 参数个数 = %d，期望 %d\n实际参数: %#v", len(args), want, args)
			}
		})
	}
}

// TestFindFollowedDemands_CountAndListShareFilters 锁定 Count 与列表查询的
// 过滤条件完全一致：拆分不得让两条查询走不同的过滤分支。
func TestFindFollowedDemands_CountAndListShareFilters(t *testing.T) {
	cases := []struct {
		name string
		req  RepoFindFollowedDemandsReq
	}{
		{"plain", RepoFindFollowedDemandsReq{Account: "alice", Page: 1, PageSize: 20}},
		{"keyword", RepoFindFollowedDemandsReq{Account: "alice", Keyword: "登录", Page: 1, PageSize: 20}},
		{"scope+lifecycle", RepoFindFollowedDemandsReq{Account: "alice", Scope: FollowScopeKeyOpen, Lifecycle: FollowLifecycleImplementing, Page: 1, PageSize: 20}},
		{"all combined", RepoFindFollowedDemandsReq{
			Account: "alice", Keyword: "登录", Scope: FollowScopeOpenClean,
			Lifecycle: FollowLifecycleReleased, Page: 3, PageSize: 5,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, rec := newSQLBaselineRepo(t, 8)
			if _, _, _, err := repo.FindFollowedDemands(t.Context(), c.req); err != nil {
				t.Fatalf("FindFollowedDemands 返回错误: %v", err)
			}
			countSQL, listSQL := rec.followCountQuery(), rec.followListQuery()
			if countSQL == "" || listSQL == "" {
				t.Fatalf("未同时捕获到 Count 与列表查询：count=%q list=%q", countSQL, listSQL)
			}
			// Count 的 WHERE 整体必须原样出现在列表查询里。
			_, where, ok := strings.Cut(countSQL, " WHERE ")
			if !ok {
				t.Fatalf("Count 查询缺少 WHERE: %s", countSQL)
			}
			if !strings.Contains(listSQL, where) {
				t.Errorf("列表查询未复用 Count 的过滤条件\nCount WHERE: %s\n列表: %s", where, listSQL)
			}
		})
	}
}

// TestFindFollowedDemands_PagingLocked 锁定分页归一化与 LIMIT/OFFSET。
func TestFindFollowedDemands_PagingLocked(t *testing.T) {
	cases := []struct {
		name              string
		page, size        int
		wantLimit, offset int
		wantOffsetClause  bool
	}{
		{"zero page and size normalize", 0, 0, 20, 0, false},
		{"negative page normalize", -3, -5, 20, 0, false},
		{"oversized pageSize clamps", 1, 500, 20, 0, false},
		{"page two keeps size", 2, 20, 20, 20, true},
		{"explicit size ten", 3, 10, 10, 20, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, rec := newSQLBaselineRepo(t, 8)
			if _, _, _, err := repo.FindFollowedDemands(t.Context(), RepoFindFollowedDemandsReq{
				Account: "alice", Page: c.page, PageSize: c.size,
			}); err != nil {
				t.Fatalf("FindFollowedDemands 返回错误: %v", err)
			}
			assertFollowPaging(t, rec, c.wantLimit, c.offset, c.wantOffsetClause)
		})
	}
}

// assertFollowPaging 校验列表查询的倒序、LIMIT / OFFSET 存在性与绑定值。
// page 归一到 1 时 offset 为 0，GORM 省略 OFFSET 子句。
func assertFollowPaging(t *testing.T, rec *sqlBaselineRecorder, wantLimit, wantOffset int, wantOffsetClause bool) {
	t.Helper()
	sql := rec.followListQuery()
	if !strings.Contains(sql, "ORDER BY d.id DESC") {
		t.Errorf("列表应按 id 倒序，实际: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT ?") {
		t.Errorf("列表应带 LIMIT，实际: %s", sql)
	}
	if hasOffset := strings.Contains(sql, "OFFSET ?"); hasOffset != wantOffsetClause {
		t.Errorf("OFFSET 存在性 = %v，期望 %v；实际 SQL: %s", hasOffset, wantOffsetClause, sql)
	}
	// SQL 写作 LIMIT ? OFFSET ?，绑定参数同序；offset 为 0 时 GORM 省略 OFFSET。
	args := rec.followListArgs()
	if len(args) == 0 {
		t.Fatalf("列表查询缺少绑定参数")
	}
	limitIdx := len(args) - 1
	if wantOffsetClause {
		limitIdx = len(args) - 2
		if got := toInt64(args[len(args)-1]); got != int64(wantOffset) {
			t.Errorf("OFFSET = %d，期望 %d", got, wantOffset)
		}
	}
	if got := toInt64(args[limitIdx]); got != int64(wantLimit) {
		t.Errorf("LIMIT = %d，期望 %d", got, wantLimit)
	}
}

// TestFindFollowedDemands_QueryOrderAndCount 锁定查询条数与顺序：
// 生命周期统计 → Count → 列表 → 账号姓名。拆分不得改变这个次序。
func TestFindFollowedDemands_QueryOrderAndCount(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, _, _, err := repo.FindFollowedDemands(t.Context(), followBaseReq()); err != nil {
		t.Fatalf("FindFollowedDemands 返回错误: %v", err)
	}
	if rec.count() != 4 {
		t.Fatalf("查询条数 = %d，期望 4（统计 / Count / 列表 / 姓名）", rec.count())
	}
	if !strings.Contains(rec.sql(0), "FROM zt_demand d") || !strings.Contains(rec.sql(0), "c_open") {
		t.Errorf("第 1 条应为生命周期统计，实际: %s", rec.sql(0))
	}
	if !strings.HasPrefix(rec.sql(1), "SELECT count(*)") {
		t.Errorf("第 2 条应为 Count，实际: %s", rec.sql(1))
	}
	if !strings.Contains(rec.sql(2), "ORDER BY d.id DESC") {
		t.Errorf("第 3 条应为列表查询，实际: %s", rec.sql(2))
	}
	if !strings.Contains(rec.sql(3), "zt_user") {
		t.Errorf("第 4 条应为账号姓名查询，实际: %s", rec.sql(3))
	}
}

// TestFindFollowedDemands_EmptyAccountShortCircuits 空账号不查库，
// 返回空结果与空统计而非报错。
func TestFindFollowedDemands_EmptyAccountShortCircuits(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	items, total, stats, err := repo.FindFollowedDemands(t.Context(),
		RepoFindFollowedDemandsReq{Account: "  ", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("空账号不应报错: %v", err)
	}
	if items != nil || total != 0 || stats == nil {
		t.Fatalf("空账号应短路，得到 items=%v total=%d stats=%v", items, total, stats)
	}
	if rec.count() != 0 {
		t.Errorf("空账号不应查库，实际发出 %d 条查询", rec.count())
	}
}

// TestFindFollowedDemands_StatsKeywordMatchesList 锁定关键字在统计与列表两侧
// 都被应用：统计不能漏掉 keyword，否则侧栏计数会与列表不一致。
func TestFindFollowedDemands_StatsKeywordMatchesList(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, _, _, err := repo.FindFollowedDemands(t.Context(), RepoFindFollowedDemandsReq{
		Account: "alice", Keyword: "登录", Page: 1, PageSize: 20,
	}); err != nil {
		t.Fatalf("FindFollowedDemands 返回错误: %v", err)
	}
	statsSQL := rec.sql(0)
	if !strings.Contains(statsSQL, "AND (d.name LIKE ? OR CAST(d.id AS CHAR) = ?)") {
		t.Errorf("统计查询应带上关键字条件，实际: %s", statsSQL)
	}
	if len(rec.queries[0].args) != 5 {
		t.Errorf("统计参数个数 = %d，期望 5（watch 3 + 关键字 2）", len(rec.queries[0].args))
	}
	// 关键字不做通配符包裹以外的加工，原样传入。
	if got := rec.queries[0].args[3]; got != "%登录%" {
		t.Errorf("统计关键字参数 = %v，期望 %%登录%%", got)
	}
	if got := rec.queries[0].args[4]; got != "登录" {
		t.Errorf("统计 ID 匹配参数 = %v，期望 登录", got)
	}
}

// TestFindFollowedDemands_StatsQueryShape 锁定生命周期统计的聚合口径：
// 9 个分桶列名固定，缺失任何一列都会让侧栏计数静默变 0。
func TestFindFollowedDemands_StatsQueryShape(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, _, _, err := repo.FindFollowedDemands(t.Context(), followBaseReq()); err != nil {
		t.Fatalf("FindFollowedDemands 返回错误: %v", err)
	}
	sql := rec.sql(0)
	for _, col := range []string{
		"c_open", "c_all", "c_clarifying", "c_implementing", "c_released",
		"c_closed", "c_key", "c_key_open", "c_open_clean",
	} {
		if !strings.Contains(sql, "AS "+col) {
			t.Errorf("统计查询缺少聚合列 %s", col)
		}
	}
	if !strings.Contains(sql, "FROM zt_demand d") || !strings.Contains(sql, "d.deleted = '0'") {
		t.Errorf("统计查询应限定未删除需求，实际: %s", sql)
	}
}

// TestFindFollowedDemands_ListProjectionLocked 锁定列表投影列，
// 缺一列会让负责人 / 关键时间 / 关注来源静默丢值。
func TestFindFollowedDemands_ListProjectionLocked(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, _, _, err := repo.FindFollowedDemands(t.Context(), followBaseReq()); err != nil {
		t.Fatalf("FindFollowedDemands 返回错误: %v", err)
	}
	sql := rec.followListQuery()
	for _, col := range []string{
		"d.id", "d.name", "d.status", "d.pri", "d.BRA", "d.QD", "d.RD",
		"d.deadline", "d.developFinish", "d.testFinish",
		"d.isNeedFocus AS need_focus", "AS system_name", "AS watch_source",
	} {
		if !strings.Contains(sql, col) {
			t.Errorf("列表查询缺少投影列 %q", col)
		}
	}
	// 关注来源必须由 zt_starinfo 判定，不能用 mailto 兜底成 star。
	if !strings.Contains(sql, "s3.followed = '1'") || !strings.Contains(sql, "THEN 'star' ELSE 'mailto' END") {
		t.Errorf("watch_source 判定口径不符基准，实际: %s", sql)
	}
	if !strings.Contains(sql, "LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED)") {
		t.Errorf("列表应 LEFT JOIN zt_product 推导系统名，实际: %s", sql)
	}
}

// TestFindFollowedDemands_WatchClauseLocked 锁定关注真源判定：
// zt_starinfo followed='1' 优先，mailto 兼容且被显式取消关注压制。
func TestFindFollowedDemands_WatchClauseLocked(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, _, _, err := repo.FindFollowedDemands(t.Context(), followBaseReq()); err != nil {
		t.Fatalf("FindFollowedDemands 返回错误: %v", err)
	}
	sql := rec.followCountQuery()
	for _, frag := range []string{
		"s.objectType = 'demand'",
		"s.objectID = d.id",
		"s.followed = '1'",
		"FIND_IN_SET(?, REPLACE(COALESCE(d.mailto, ''), ' ', '')) > 0",
		"s2.followed = '0'",
		"NOT EXISTS",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("关注判定缺少片段 %q\n实际: %s", frag, sql)
		}
	}
	// watch 子句固定绑定 3 次账号（starinfo + FIND_IN_SET + 压制判定），
	// 末位是默认 scope 的 d.status <> ?。
	if got := rec.followCountArgs(); len(got) != 5 || got[0] != "0" ||
		got[1] != "alice" || got[2] != "alice" || got[3] != "alice" || got[4] != "closed" {
		t.Errorf("Count 参数 = %#v，期望 [0 alice alice alice closed]", got)
	}
}
