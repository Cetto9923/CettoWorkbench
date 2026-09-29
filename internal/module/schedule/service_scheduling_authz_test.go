// =============================================================================
// 文件: internal/module/schedule/service_scheduling_authz_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 排期写路径对象级鉴权回归：
//   - 无写权限账号调用 SaveScheduling 返回 403 且完全不触发写库 / 同步禅道
//   - 有写权限账号（干系人 / 超管）通过对象级闸门
// =============================================================================

package schedule

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
)

func newAuthzMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open stub database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm database: %v", err)
	}
	return db, mock
}

const (
	authzPMOQuery     = `SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = 'pmo'`
	authzRelatedQuery = `SELECT COUNT\(\*\) FROM zt_demand d`
	authzManagedQuery = `SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`
)

// TestSaveScheduling_NoWritePermissionReturnsForbiddenWithoutTouchingDB 证明：
// 非干系人（仅持 perm.ScheduleList）保存排期时返回 403，
// 且事务、zt_demand 写入、禅道同步全部未发生（mock 无任何写/额外查询预期）。
func TestSaveScheduling_NoWritePermissionReturnsForbiddenWithoutTouchingDB(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	// 仅 ScheduleList 的只读放行不构成写权限。
	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})

	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(15865)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(authzRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(authzManagedQuery).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	actor := &model.User{ID: 15865, Account: "demo_leader"}
	err := svc.SaveScheduling(ctx, actor, 63450, &SaveSchedulingReq{WindowID: 1})

	bizErr, ok := errorx.IsBizError(err)
	if !ok {
		t.Fatalf("expected BizError forbidden, got %v", err)
	}
	if bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected %s, got %s", errorx.ErrCodeForbidden, bizErr.Code)
	}
	// 关键断言：GetDemandMainSystem / BEGIN / 任何写库都未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden save must not touch the database beyond authz reads: %v", err)
	}
}

// TestRequireDemandWriteAccess_StakeholderAllowed 证明：需求干系人通过写权限闸门。
func TestRequireDemandWriteAccess_StakeholderAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(15864)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(authzRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := svc.RequireDemandWriteAccess(t.Context(), &model.User{ID: 15864, Account: "demo_po"}, 63450)
	if err != nil {
		t.Fatalf("stakeholder should pass write authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// TestRequireDemandWriteAccess_SuperAdminAllowed 证明：超管免关系查询直接放行。
func TestRequireDemandWriteAccess_SuperAdminAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	if err := svc.RequireDemandWriteAccess(t.Context(), &model.User{ID: 7, Account: "003030", IsSuperAdmin: true}, 63450); err != nil {
		t.Fatalf("super admin should pass write authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("super admin must not trigger relation queries: %v", err)
	}
}

// TestRequireDemandWriteAccess_InvalidDemandID 证明：非法 ID 返回 400 级参数错误而非 403。
func TestRequireDemandWriteAccess_InvalidDemandID(t *testing.T) {
	db, _ := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	err := svc.RequireDemandWriteAccess(t.Context(), &model.User{ID: 1, Account: "demo_po", IsSuperAdmin: true}, 0)
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam, got %v", err)
	}
}
