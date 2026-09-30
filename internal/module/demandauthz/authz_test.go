// =============================================================================
// 文件: internal/module/demandauthz/authz_test.go
// 模块: 需求对象级授权
// 类型: test
// 职责: 写权限口径回归：超管、PMO、干系人、团队长管辖可写；
//       仅 ScheduleList 与完全无关系者不可写。
// =============================================================================

package demandauthz

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/workbenchroles"
)

const (
	pmoQuery     = `SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`
	relatedQuery = `SELECT COUNT\(\*\) FROM zt_demand d`
	managedQuery = `SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`
	deptTree     = `SELECT DISTINCT .id. FROM .zt_dept. WHERE id IN \(\?\)`
)

// 1. 超级管理员：不需要任何关系查询即可写。
func TestEvaluate_SuperAdminCanWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	ok, err := repo.CanWriteDemand(t.Context(), &model.User{ID: 1, Account: "root", IsSuperAdmin: true}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("super admin should be able to write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("super admin must not trigger relation queries: %v", err)
	}
}

// 2. 组织角色 PMO：全量可写。
func TestEvaluate_PMOCanWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(pmoQuery).
		WithArgs(int64(20), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.CanWriteDemand(t.Context(), &model.User{ID: 20, Account: "pmo_user"}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("PMO should be able to write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 3. 个人干系人：可写，且不再下探团队长判断。
func TestEvaluate_StakeholderCanWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(pmoQuery).
		WithArgs(int64(30), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(relatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.CanWriteDemand(t.Context(), &model.User{ID: 30, Account: "demo_po"}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("stakeholder should be able to write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 4. 团队长管辖：需求干系人属于其管辖部门，可写。
func TestEvaluate_LeaderScopeCanWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(pmoQuery).
		WithArgs(int64(40), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(relatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(managedQuery).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",53,52,14,"))
	mock.ExpectQuery(deptTree).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))
	mock.ExpectQuery(relatedQuery).
		WithArgs(uint(1001), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.CanWriteDemand(t.Context(), &model.User{ID: 40, Account: "demo_leader"}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("dept leader should be able to write for demands in scope")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

//  5. 仅持有 perm.ScheduleList：无任何对象级关系，判定为只读（不可写）。
//     ScheduleList 是读权限位，Evaluate 完全不读取权限上下文。
func TestEvaluate_ScheduleListHolderIsReadOnly(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)
	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})

	mock.ExpectQuery(pmoQuery).
		WithArgs(int64(50), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(relatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(managedQuery).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	ok, err := repo.CanWriteDemand(ctx, &model.User{ID: 50, Account: "schedule_viewer"}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("ScheduleList holder must not be able to write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 6. 无任何关系与权限：不可写。
func TestEvaluate_NoRelationCannotWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	mock.ExpectQuery(pmoQuery).
		WithArgs(int64(60), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(relatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(managedQuery).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	ok, err := repo.CanWriteDemand(t.Context(), &model.User{ID: 60, Account: "stranger"}, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("unrelated user must not be able to write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 7. 入参非法（无账号 / 无需求 ID）：直接不可写，不查库。
func TestEvaluate_InvalidInputCannotWrite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := New(gormDB)

	cases := []struct {
		name     string
		actor    *model.User
		demandID uint
	}{
		{"nil actor", nil, 1001},
		{"blank account", &model.User{ID: 70}, 1001},
		{"zero demand", &model.User{ID: 70, Account: "demo_po"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := repo.CanWriteDemand(t.Context(), tc.actor, tc.demandID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok {
				t.Fatal("invalid input must not be able to write")
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("invalid input must not trigger any query: %v", err)
	}
}
