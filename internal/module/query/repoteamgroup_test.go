package query

import (
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// 业务需求直接比 zt_demand.teamGroup，按字符串形态比较避免隐式数值转换。
func TestListDemandsFiltersByTeamgroupAsString(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM .*zt_demand d.*d\.teamGroup = \?`).
		WithArgs("0", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*d\.teamGroup = \?.*ORDER BY d\.id DESC`).
		WithArgs("0", "147", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).
			AddRow(1, "需求", "3", "active", "", "张三", "核心系统", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Group: "147", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Rows) != 1 {
		t.Fatalf("unexpected filtered response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 研发需求按需求树根需求的 teamGroup 继承（与排期页口径一致）：顶层取自身，子需求取父需求。
func TestListStoriesInheritsTeamgroupFromRootDemand(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	inherited := regexp.QuoteMeta(groupCaliberExpr + " = ?")
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM zt_story s.*LEFT JOIN zt_demand gd.*LEFT JOIN zt_demand gg.*`+inherited).
		WithArgs("0", 0, "story", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_story s.*LEFT JOIN zt_demand gd.*LEFT JOIN zt_demand gg.*`+inherited+`.*ORDER BY s\.id DESC`).
		WithArgs("0", 0, "story", "147", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).
			AddRow(2, "研发需求", 2, "active", "", "李四", "核心系统", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Tab: tabRD, Group: "147", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Rows) != 1 {
		t.Fatalf("unexpected filtered response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 小组筛选与状态筛选同时出现时必须是 AND 叠加，而非互相覆盖。
func TestListDemandsGroupAndStatusAreConjunctive(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM .*zt_demand d.*d\.status = \?.*d\.teamGroup = \?`).
		WithArgs("0", "active", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*d\.status = \?.*d\.teamGroup = \?.*ORDER BY d\.id DESC`).
		WithArgs("0", "active", "147", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).
			AddRow(1, "需求一", "3", "active", "", "张三", "核心系统", nil, "").
			AddRow(2, "需求二", "2", "active", "", "李四", "核心系统", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Status: "active", Group: "147", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 || len(resp.Rows) != 2 {
		t.Fatalf("unexpected conjunctive response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 敏捷小组选项覆盖全部未删除小组，按 name + id 稳定排序。
func TestListGroupOptionsCoversAllAliveGroups(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_teamgroup.*WHERE deleted = \?.*ORDER BY name ASC, id ASC`).
		WithArgs("0").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow(147, "对公一组").AddRow(7, "MCP小组"),
	)

	got, err := repo.ListGroupOptions(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 147 || got[0].Name != "对公一组" || got[1].ID != 7 {
		t.Fatalf("unexpected group options: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 小组筛选与关键词筛选同时出现时必须是 AND 叠加：关键词条件不能覆盖小组条件。
func TestListDemandsGroupAndKeywordAreConjunctive(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	like := "%项目%"
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM .*zt_demand d.*LOWER\(d\.name\) LIKE.*d\.teamGroup = \?`).
		WithArgs("0", like, like, like, like, "项目", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*LOWER\(d\.name\) LIKE.*d\.teamGroup = \?.*ORDER BY d\.id DESC`).
		WithArgs("0", like, like, like, like, "项目", "147", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}).
			AddRow(7, "统一认证项目", "2", "active", "", "张三", "核心系统", nil, ""),
	)

	resp, err := repo.List(t.Context(), ListReq{Keyword: "项目", Group: "147", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Rows) != 1 || resp.Rows[0].ID != 7 {
		t.Fatalf("unexpected conjunctive response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// storyRows 构造列表查询的结果集，避免每个用例重复拼列名。
// demand 为真时标题列是 zt_demand.name，zt_story 则是 title。
func storyRows(demand bool, rows ...[]driver.Value) *sqlmock.Rows {
	nameCol := "title"
	if demand {
		nameCol = "name"
	}
	out := sqlmock.NewRows([]string{"id", nameCol, "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"})
	for _, r := range rows {
		out.AddRow(r...)
	}
	return out
}

// assertListRows 校验 total 与返回条数一致且无重复 id，
// 重复 id 或 count 大于实际条数都意味着 join 造成行放大。
func assertListRows(t *testing.T, resp ListResp) {
	t.Helper()
	if resp.Total != int64(len(resp.Rows)) {
		t.Fatalf("count/list mismatch would mean join row amplification: total=%d rows=%d", resp.Total, len(resp.Rows))
	}
	seen := map[uint]bool{}
	for _, row := range resp.Rows {
		if seen[row.ID] {
			t.Fatalf("duplicated row %d would mean join row amplification", row.ID)
		}
		seen[row.ID] = true
	}
}

// 未选择小组时 SQL 不得出现 teamGroup 条件，也不得出现继承小组的两个 join：
// 这是小组筛选的回归保护，确保老行为逐字不变。
func TestListWithoutGroupKeepsOriginalSQL(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM .*zt_demand d`).
		WithArgs("0").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*ORDER BY d\.id DESC`).
		WithArgs("0", 15).WillReturnRows(
		storyRows(true,
			[]driver.Value{1, "需求一", "3", "active", "", "张三", "核心系统", nil, ""},
			[]driver.Value{2, "需求二", "2", "active", "", "李四", "核心系统", nil, ""},
			[]driver.Value{3, "需求三", "1", "closed", "", "王五", "核心系统", nil, ""},
		),
	)

	resp, err := repo.List(t.Context(), ListReq{Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	assertListRows(t, resp)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 研发需求未选择小组时同样不得引入 zt_demand join。
func TestListStoriesWithoutGroupKeepsOriginalSQL(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM zt_story s`).
		WithArgs("0", 0, "story").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`(?s)SELECT .*FROM zt_story s.*ORDER BY s\.id DESC`).
		WithArgs("0", 0, "story", 15).WillReturnRows(
		storyRows(false,
			[]driver.Value{11, "研需一", 2, "active", "", "张三", "核心系统", nil, ""},
			[]driver.Value{12, "研需二", 3, "closed", "", "李四", "核心系统", nil, ""},
		),
	)

	resp, err := repo.List(t.Context(), ListReq{Tab: tabRD, Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	assertListRows(t, resp)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 研发需求按小组筛选时 total 必须等于实际返回条数（pageSize 足够大时），
// 防止 LEFT JOIN zt_demand 造成行放大导致 count 虚高。
func TestListStoriesGroupCountMatchesRowCount(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	inherited := regexp.QuoteMeta(groupCaliberExpr + " = ?")
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM zt_story s.*LEFT JOIN zt_demand gd.*LEFT JOIN zt_demand gg.*`+inherited).
		WithArgs("0", 0, "story", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`(?s)SELECT .*FROM zt_story s.*LEFT JOIN zt_demand gd.*LEFT JOIN zt_demand gg.*`+inherited+`.*ORDER BY s\.id DESC`).
		WithArgs("0", 0, "story", "147", 100).WillReturnRows(
		storyRows(false,
			[]driver.Value{21, "研需一", 2, "active", "", "张三", "核心系统", nil, ""},
			[]driver.Value{22, "研需二", 3, "active", "", "李四", "核心系统", nil, ""},
			[]driver.Value{23, "研需三", 1, "closed", "", "王五", "核心系统", nil, ""},
		),
	)

	resp, err := repo.List(t.Context(), ListReq{Tab: tabRD, Group: "147", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	assertListRows(t, resp)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 顶层研需（fromDemand=0）经 join 后 gd 为 NULL，expr 得到空串，
// 因此选择任一小组都不应命中它。
func TestTopLevelStoryIsNeverMatchedByGroup(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	inherited := regexp.QuoteMeta(groupCaliberExpr + " = ?")
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM zt_story s.*LEFT JOIN zt_demand gd.*`+inherited).
		WithArgs("0", 0, "story", "147").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?s)SELECT .*FROM zt_story s.*LEFT JOIN zt_demand gd.*`+inherited+`.*ORDER BY s\.id DESC`).
		WithArgs("0", 0, "story", "147", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}),
	)

	resp, err := repo.List(t.Context(), ListReq{Tab: tabRD, Group: "147", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 0 || len(resp.Rows) != 0 {
		t.Fatalf("top-level story must not match any group: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// groupCaliberExpr 的语义锚点：顶层父需求取自身 teamGroup，
// 非顶层父需求取其父需求（祖父）的 teamGroup。改动此表达式时此测试必须同步。
// 这里断言 SQL 字面量而不是引用常量本身，否则把 gg 接到错误的主键上时本测试仍会通过。
func TestGroupCaliberExprSemantics(t *testing.T) {
	if !strings.Contains(groupCaliberJoins, "LEFT JOIN zt_demand gd ON gd.id = s.fromDemand") {
		t.Error("gd must join the story's originating demand")
	}
	if !strings.Contains(groupCaliberJoins, "LEFT JOIN zt_demand gg ON gg.id = gd.parent") {
		t.Error("gg must join the parent of gd so non-top-level cases reach the root demand")
	}
	if !strings.Contains(groupCaliberExpr, "gd.parent IN (0, -1)") {
		t.Error("caliber must branch on the parent demand being top-level")
	}
	if !strings.Contains(groupCaliberExpr, "THEN COALESCE(gd.teamGroup, '')") {
		t.Error("top-level parent demand must use its own teamGroup")
	}
	if !strings.Contains(groupCaliberExpr, "ELSE COALESCE(gg.teamGroup, '')") {
		t.Error("non-top-level parent demand must inherit from the root demand")
	}
	// gg 只能通过 gd.parent 上溯一层；接到 gd.id 会退化成"取父需求自身"，
	// 那正是与排期页口径不一致的写法，必须被测试挡住。
	if strings.Contains(groupCaliberJoins, "gg.id = gd.id") {
		t.Error("gg must not join gd itself; the caliber would fall back to the immediate parent")
	}
}

// Normalize 需对 Group 去空白，避免 " 147 " 查不到结果；纯空白视为不筛选。
func TestListReqNormalizeTrimsGroup(t *testing.T) {
	req := ListReq{Group: "  147  "}
	req.Normalize()
	if req.Group != "147" {
		t.Fatalf("group must be trimmed, got %q", req.Group)
	}
	blank := ListReq{Group: "   "}
	blank.Normalize()
	if blank.Group != "" {
		t.Fatalf("blank group must normalize to empty, got %q", blank.Group)
	}
}

// 不存在的 / 已删除的小组 id 不报错，按查不到结果处理。
func TestListDemandsUnknownGroupYieldsEmptyResult(t *testing.T) {
	repo, mock := newQueryRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM .*zt_demand d.*d\.teamGroup = \?`).
		WithArgs("0", "999999").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?s)SELECT .*FROM .*zt_demand d.*d\.teamGroup = \?.*ORDER BY d\.id DESC`).
		WithArgs("0", "999999", 15).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "pri", "status", "stage", "owner", "system_name", "estimateLaunch", "source"}),
	)

	resp, err := repo.List(t.Context(), ListReq{Group: "999999", Page: 1, PageSize: 15})
	if err != nil {
		t.Fatalf("unknown group must not error: %v", err)
	}
	if resp.Total != 0 || len(resp.Rows) != 0 {
		t.Fatalf("unknown group must yield empty result: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
