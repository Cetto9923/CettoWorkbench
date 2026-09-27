package agileteam

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
)

func TestBuildTeamFamiliesKeepsParentContext(t *testing.T) {
	t.Parallel()
	parent := ListItem{ID: 1, Name: "信贷专项团队", Type: "parent"}
	child := ListItem{ID: 11, Name: "对公1组", ParentID: 1, ParentName: "信贷专项团队", Type: "child"}
	orphan := ListItem{ID: 99, Name: "独立组", Type: "parent"}
	all := map[uint]ListItem{1: parent, 11: child, 99: orphan}

	families := buildTeamFamilies([]ListItem{child}, all)
	if len(families) != 1 {
		t.Fatalf("want 1 family, got %d", len(families))
	}
	if families[0].Parent.ID != 1 || families[0].Parent.ChildCount != 1 {
		t.Fatalf("parent context missing: %+v", families[0].Parent)
	}
	if len(families[0].Children) != 1 || families[0].Children[0].ID != 11 {
		t.Fatalf("child missing: %+v", families[0].Children)
	}
}

func TestSelectLeadScopeDefaultsToAvailableScopeAndPreservesExplicitChoice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		requested string
		available []string
		want      string
	}{
		{name: "department manager only", available: []string{"dept"}, want: "dept"},
		{name: "coach and manager default to team", available: []string{"team", "dept"}, want: "team"},
		{name: "preserve explicit department", requested: "dept", available: []string{"team", "dept"}, want: "dept"},
		{name: "reject unavailable choice", requested: "team", available: []string{"dept"}, want: "dept"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectLeadScope(tc.requested, tc.available); got != tc.want {
				t.Fatalf("selectLeadScope(%q, %v) = %q, want %q", tc.requested, tc.available, got, tc.want)
			}
		})
	}
}

func TestBuildTeamFamiliesCanUseSanitizedParentContext(t *testing.T) {
	t.Parallel()
	child := ListItem{ID: 11, Name: "对公1组", ParentID: 1, ParentName: "信贷专项团队", Type: "child"}
	all := map[uint]ListItem{child.ID: child}
	all[1] = ListItem{ID: 1, Name: "信贷专项团队", Type: "parent", ContextOnly: true}

	families := buildTeamFamilies([]ListItem{child}, all)
	if len(families) != 1 || families[0].Parent.ID != 1 || !families[0].Parent.ContextOnly {
		t.Fatalf("sanitized parent context missing: %+v", families)
	}
	if len(families[0].Children) != 1 || families[0].Children[0].ID != 11 {
		t.Fatalf("unexpected visible siblings: %+v", families[0].Children)
	}
}

func TestDepartmentManagerScopeIncludesOnlyMappedGroupNotSiblings(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM zt_dept WHERE COALESCE\\(manager, ''\\) REGEXP \\?").
		WithArgs("(^|[[:space:],;])lead1([[:space:],;]|$)").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("(?s)SELECT id, COALESCE\\(path, ''\\) AS path FROM zt_dept WHERE COALESCE\\(manager, ''\\) REGEXP \\?").
		WithArgs("(^|[[:space:],;])lead1([[:space:],;]|$)").WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(20), ",1,20,"))
	mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
		WithArgs(uint(20), ",1,20,%").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(20)))
	mock.ExpectQuery("(?s)FROM zt_wb_agileteam_orgmap m.*m.deptId IN \\(\\?\\)").
		WithArgs(uint(20)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	mock.ExpectQuery("SELECT id, name FROM zt_teamgroup WHERE deleted = '0' AND id IN \\(\\?\\) ORDER BY id ASC").
		WithArgs(uint(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uint(1), "信贷专项团队"))

	rows := []TeamgroupRow{
		{ID: 1, Name: "信贷专项团队", Type: "parent"},
		{ID: 11, Name: "对公1组", Parent: 1, ParentName: "信贷专项团队", Type: "child"},
		{ID: 12, Name: "对公2组", Parent: 1, ParentName: "信贷专项团队", Type: "child"},
	}
	_, visible, err := svc.applyLeadScope(context.Background(), &model.User{Account: "lead1"}, ListReq{View: "lead", Scope: "dept"}, rows)
	if err != nil {
		t.Fatalf("applyLeadScope() error = %v", err)
	}
	if len(visible) != 1 || visible[0].ID != 11 {
		t.Fatalf("department manager can see groups %v, want only mapped group 11", visible)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestAgileCoachScopeIncludesOnlyManagedGroupNotSibling(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*scoped.parent = managed.id.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])coach1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	mock.ExpectQuery("SELECT id, name FROM zt_teamgroup WHERE deleted = '0' AND id IN \\(\\?\\) ORDER BY id ASC").
		WithArgs(uint(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uint(1), "信贷专项团队"))

	rows := []TeamgroupRow{
		{ID: 1, Name: "信贷专项团队", Type: "parent"},
		{ID: 11, Name: "对公1组", Parent: 1, ParentName: "信贷专项团队", Type: "child"},
		{ID: 12, Name: "对公2组", Parent: 1, ParentName: "信贷专项团队", Type: "child"},
	}
	_, visible, err := svc.applyLeadScope(context.Background(), &model.User{Account: "coach1"}, ListReq{View: "lead", Scope: "team"}, rows)
	if err != nil {
		t.Fatalf("applyLeadScope() error = %v", err)
	}
	if len(visible) != 1 || visible[0].ID != 11 {
		t.Fatalf("coach can see groups %v, want only managed group 11", visible)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestPageFamiliesDoesNotSplitParentFromChildren(t *testing.T) {
	t.Parallel()
	families := []teamFamily{
		{Parent: ListItem{ID: 1, Name: "P1"}, Children: []ListItem{{ID: 2}, {ID: 3}}},
		{Parent: ListItem{ID: 10, Name: "P2"}, Children: []ListItem{{ID: 11}}},
	}
	items, pages := pageFamilies(families, 1, 3)
	if pages != 2 {
		t.Fatalf("want 2 pages (P1 family=3 rows fills page1), got %d", pages)
	}
	if len(items) != 3 || items[0].ID != 1 || items[2].ID != 3 {
		t.Fatalf("page1 should be entire P1 family, got %+v", items)
	}
	items2, _ := pageFamilies(families, 2, 3)
	if len(items2) != 2 || items2[0].ID != 10 {
		t.Fatalf("page2 should be P2 family, got %+v", items2)
	}
}
