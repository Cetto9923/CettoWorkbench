// =============================================================================
// 文件: internal/module/agileteam/repo_scope_test.go
// 模块: 敏捷小组治理
// 类型: security regression
// 职责: 锁定部门负责人只能按本人负责部门及下级确定范围。
// =============================================================================

package agileteam

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListDeptTreeIDsUsesManagedDepartmentsAndDescendants(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)SELECT id, COALESCE\\(path, ''\\) AS path FROM zt_dept WHERE COALESCE\\(manager, ''\\) REGEXP \\?").
		WithArgs("(^|[[:space:],;])lead1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).
			AddRow(uint(20), ",1,20,").
			AddRow(uint(40), ",1,40,"))
	mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?,\\?\\) OR path LIKE \\? OR path LIKE \\?").
		WithArgs(uint(20), uint(40), ",1,20,%", ",1,40,%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(20)).
			AddRow(uint(21)).
			AddRow(uint(40)).
			AddRow(uint(41)))

	got, err := repo.ListDeptTreeIDs(context.Background(), " lead1 ")
	if err != nil {
		t.Fatalf("ListDeptTreeIDs() error = %v", err)
	}
	want := []uint{20, 21, 40, 41}
	if len(got) != len(want) {
		t.Fatalf("ListDeptTreeIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDeptTreeIDs() = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListDeptTreeIDsReturnsEmptyWhenAccountManagesNoDepartment(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)SELECT id, COALESCE\\(path, ''\\) AS path FROM zt_dept WHERE COALESCE\\(manager, ''\\) REGEXP \\?").
		WithArgs("(^|[[:space:],;])member1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	got, err := repo.ListDeptTreeIDs(context.Background(), "member1")
	if err != nil {
		t.Fatalf("ListDeptTreeIDs() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListDeptTreeIDs() = %v, want empty", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListMappedTeamgroupIDsByDeptsUsesOnlyActiveMappings(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)FROM zt_wb_agileteam_orgmap m.*mapped.type = 'parent'.*m.status = 'active'.*m.deptId IN \\(\\?,\\?\\)").
		WithArgs(uint(20), uint(21)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)).AddRow(uint(11)).AddRow(uint(12)))

	got, err := repo.ListMappedTeamgroupIDsByDepts(context.Background(), []uint{20, 21})
	if err != nil {
		t.Fatalf("ListMappedTeamgroupIDsByDepts() error = %v", err)
	}
	want := []uint{3, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("mapped teamgroup IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mapped teamgroup IDs = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListManagedTeamgroupIDsScopesParentCoachToFamilyAndChildCoachToSelf(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*scoped.parent = managed.id.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])lead1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)).AddRow(uint(11)).AddRow(uint(12)))

	got, err := repo.ListManagedTeamgroupIDs(context.Background(), " lead1 ")
	if err != nil {
		t.Fatalf("ListManagedTeamgroupIDs() error = %v", err)
	}
	want := []uint{3, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("managed teamgroup IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("managed teamgroup IDs = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestAccountTokenRegexpDoesNotMatchAccountSubstring(t *testing.T) {
	rx, err := regexp.Compile(accountTokenRegexp("lead1"))
	if err != nil {
		t.Fatalf("account token regexp: %v", err)
	}
	if rx.MatchString("teamlead1") || rx.MatchString("lead10") {
		t.Fatal("account token must not match a longer account")
	}
	if !rx.MatchString("lead1, coach2") || !rx.MatchString("coach2;lead1") {
		t.Fatal("account token should match exact delimited accounts")
	}
}

func newTestRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	svc, mock := newTestService(t)
	return svc.repo, mock
}
