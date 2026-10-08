package query

import (
	"regexp"
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
