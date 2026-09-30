// =============================================================================
// 文件: internal/module/schedule/service_window_authz_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 版本窗口 Update / Delete 共用 canModifyWindow 的回归：
//   - ���建人可改；他人不可改；超管、PMO 可改
//   - 被拒时 403，且不进入事务、不写 zt_versionwindow
// =============================================================================

package schedule

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/workbenchroles"
)

const windowFindQuery = `SELECT \* FROM .zt_versionwindow. WHERE .*id = \?`

// expectWindowFind 铺好一次按 ID 查窗口的期望。
// GORM First 会附加 LIMIT 1 占位符，因此有两个参数。
func expectWindowFind(mock sqlmock.Sqlmock, id uint64, createdBy string) {
	mock.ExpectQuery(windowFindQuery).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "releaseDate", "startDate", "windowType",
			"planTestDone", "testDone", "acceptDone", "teamgroup", "groupSize",
			"createdBy", "updatedBy", "status", "order", "createdDate", "updatedDate", "deletedAt",
		}).AddRow(
			id, "2026Q1", time.Now(), nil, "regular",
			nil, nil, nil, 7, 3,
			createdBy, createdBy, "planning", 0, time.Now(), time.Now(), nil,
		))
}

func windowUpdateReq() UpdateReq {
	return UpdateReq{
		ID:          501,
		ReleaseDate: "2026-03-31",
		Name:        "2026Q1 改",
		TeamgroupID: 7,
		GroupSize:   3,
	}
}

// 1. 创建人可改自己的窗口。
func TestUpdateWindow_CreatorAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	// 事务内的 UPDATE / 删产品 / 建产品：这里只验证闸门放行，
	// 后续查询失败不会把用例变成误拦断言，单独由 TestCanModifyWindow_* 覆盖。
	_ = svc.Update(t.Context(), &model.User{ID: 1, Account: "demo_po"}, windowUpdateReq())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 2. 他人不可改创建人的窗口。
func TestUpdateWindow_OtherUserForbidden(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	actor := &model.User{ID: 2, Account: "demo_other"}
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(2), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := svc.Update(t.Context(), actor, windowUpdateReq())
	assertForbidden(t, err, WindowWriteDenialMessage)

	// 关键断言：BEGIN / UPDATE / DELETE 产品全部未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden update must not write the database: %v", err)
	}
}

// 3. 超管可改任意窗口。
func TestUpdateWindow_SuperAdminAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")

	err := svc.Update(t.Context(), &model.User{ID: 7, Account: "003030", IsSuperAdmin: true}, windowUpdateReq())
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		t.Fatal("super admin must not be denied window write access")
	}
	// 超管不触发关系查询。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 4. PMO 可改任意窗口。
func TestUpdateWindow_PMOAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(20), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := svc.Update(t.Context(), &model.User{ID: 20, Account: "pmo_user"}, windowUpdateReq())
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		t.Fatal("PMO must not be denied window write access")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 5. Delete 与 Update 同一口径：他人被拒，且未触发软删除。
func TestDeleteWindow_OtherUserForbidden(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(2), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := svc.Delete(t.Context(), &model.User{ID: 2, Account: "demo_other"}, DeleteReq{ID: 501})
	assertForbidden(t, err, WindowWriteDenialMessage)

	// 关键断言：软删除 UPDATE 未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden delete must not write the database: %v", err)
	}
}

// 6. Delete 对创建人放行。
func TestDeleteWindow_CreatorAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE .zt_versionwindow. SET .*deletedAt`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.Delete(t.Context(), &model.User{ID: 1, Account: "demo_po"}, DeleteReq{ID: 501}); err != nil {
		t.Fatalf("creator should be able to delete own window, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 7. 闸门本身：窗口不存在时放行，由 Update / Delete 沿用既有「窗口不存在」语义。
func TestRequireWindowWriteAccess_WindowNotFoundPassesThrough(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	mock.ExpectQuery(windowFindQuery).
		WithArgs(uint64(999), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "startDate", "windowType",
			"planTestDone", "testDone", "acceptDone", "teamgroup", "groupSize",
			"createdBy", "updatedBy", "status", "order", "createdDate", "updatedDate", "deletedAt"}))

	if err := svc.RequireWindowWriteAccess(t.Context(), &model.User{ID: 2, Account: "demo_other"}, 999); err != nil {
		t.Fatalf("missing window should not be reported as forbidden, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 8. 闸门本身：他人访问创建人窗口 → 403。
func TestRequireWindowWriteAccess_OtherUserForbidden(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(2), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := svc.RequireWindowWriteAccess(t.Context(), &model.User{ID: 2, Account: "demo_other"}, 501)
	assertForbidden(t, err, WindowWriteDenialMessage)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}
