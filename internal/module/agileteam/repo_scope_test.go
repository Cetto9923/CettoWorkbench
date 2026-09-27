// =============================================================================
// 文件: internal/module/agileteam/repo_scope_test.go
// 模块: 敏捷小组治理
// 类型: security regression
// 职责: 锁定部门负责人只能按本人负责部门及下级确定范围。
// =============================================================================

package agileteam

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListDeptTreeIDsUsesManagedDepartmentsAndDescendants(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)SELECT id, COALESCE\\(path, ''\\) AS path FROM zt_dept WHERE manager = \\?").
		WithArgs("lead1").
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
	mock.ExpectQuery("(?s)SELECT id, COALESCE\\(path, ''\\) AS path FROM zt_dept WHERE manager = \\?").
		WithArgs("member1").
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

func newTestRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	svc, mock := newTestService(t)
	return svc.repo, mock
}
