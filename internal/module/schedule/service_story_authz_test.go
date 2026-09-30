// =============================================================================
// 文件: internal/module/schedule/service_story_authz_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 研发需求两个写入口的对象级鉴权回归：
//   - 产品不可见且非超管/PMO：403，且不进入事务、不写库、不同步禅道
//   - 产品可见 / 超管 / PMO：通过闸门，不被误拦
//   - actor 缺失：403
// =============================================================================

package schedule

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/workbenchroles"
)

const (
	storyProductQuery  = `SELECT product FROM zt_story WHERE id = \? AND deleted = '0' LIMIT 1`
	storyIsAdminQuery  = `SELECT 1 AS ok FROM zt_company`
	storyUserProducts  = `SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist FROM zt_product`
	storyListAllProdID = `SELECT id FROM zt_product WHERE deleted = '0' AND status != 'closed'`
)

// expectStoryAuthzDenied 铺好「非 PMO、非超管、产品不可见」所需的全部查询期望。
// storyProduct 是该需求所属产品，visibleProduct 是账号可见的产品；两者必须不同，
// 否则闸门会正确放行，用例就失去意义。
func expectStoryAuthzDenied(mock sqlmock.Sqlmock, userID int64, storyID, storyProduct, visibleProduct uint) {
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(userID, workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(storyProductQuery).
		WithArgs(storyID).
		WillReturnRows(sqlmock.NewRows([]string{"product"}).AddRow(storyProduct))
	mock.ExpectQuery(storyIsAdminQuery).
		WithArgs("demo_outsider").
		WillReturnRows(sqlmock.NewRows([]string{"ok"}))
	// 可见产品列表不含该需求所属产品。
	mock.ExpectQuery(storyUserProducts).
		WithArgs("demo_outsider", "demo_outsider", "demo_outsider", "demo_outsider", "demo_outsider").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(visibleProduct, "别人的产品", "", "normal", "", "", "", "", ""))
}

// expectStoryProductVisible 铺好「非 PMO、非超管、但产品可见」所需的查询期望。
func expectStoryProductVisible(mock sqlmock.Sqlmock, userID int64, storyID, productID uint) {
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(userID, workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(storyProductQuery).
		WithArgs(storyID).
		WillReturnRows(sqlmock.NewRows([]string{"product"}).AddRow(productID))
	mock.ExpectQuery(storyIsAdminQuery).
		WithArgs("demo_po").
		WillReturnRows(sqlmock.NewRows([]string{"ok"}))
	mock.ExpectQuery(storyUserProducts).
		WithArgs("demo_po", "demo_po", "demo_po", "demo_po", "demo_po").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "status", "PO", "QD", "RD", "createdBy", "whitelist"}).
			AddRow(productID, "我的产品", "", "normal", "demo_po", "", "", "", ""))
}

func assertForbidden(t *testing.T, err error, wantMsg string) {
	t.Helper()
	bizErr, ok := errorx.IsBizError(err)
	if !ok {
		t.Fatalf("expected BizError, got %v", err)
	}
	if bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected %s, got %s", errorx.ErrCodeForbidden, bizErr.Code)
	}
	if bizErr.Msg != wantMsg {
		t.Fatalf("message = %q, want %q", bizErr.Msg, wantMsg)
	}
}

// 1. 保存研发需求排期：产品不可见且非超管/PMO → 403，且完全不进入事务与写库。
func TestSaveStoryScheduling_ProductNotVisibleReturnsForbiddenWithoutWrites(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	// 仅 ScheduleList 的只读放行不构成写权限。
	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})
	actor := &model.User{ID: 15865, Account: "demo_outsider"}
	expectStoryAuthzDenied(mock, 15865, 7001, 900, 901)

	err := svc.SaveStoryScheduling(ctx, actor, 7001, &SaveSchedulingReq{WindowID: 1})
	assertForbidden(t, err, StoryWriteDenialMessage)

	// 关键断言：事务、zt_story 写入、禅道同步全部未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden save must not touch the database beyond authz reads: %v", err)
	}
}

// 2. 保存维护任务：产品不可见 → 403，且不读需求详情、不进入事务。
func TestSaveStoryTasks_ProductNotVisibleReturnsForbiddenWithoutWrites(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})
	actor := &model.User{ID: 15865, Account: "demo_outsider"}
	expectStoryAuthzDenied(mock, 15865, 7002, 900, 901)

	err := svc.SaveStoryTasks(ctx, actor, 7002, &SaveStoryTasksReq{})
	assertForbidden(t, err, StoryWriteDenialMessage)

	// GetStoryTaskDetail / BEGIN / 任何写库都未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden save must not touch the database beyond authz reads: %v", err)
	}
}

// 3. 产品可见 → 通过闸门，直接断言闸门无错误（防止误拦）。
func TestRequireStoryWriteAccess_ProductVisibleAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	expectStoryProductVisible(mock, 15864, 7003, 555)

	if err := svc.RequireStoryWriteAccess(t.Context(), &model.User{ID: 15864, Account: "demo_po"}, 7003); err != nil {
		t.Fatalf("visible product should pass story write authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 3b. 产品可见时 SaveStoryScheduling 必须越过闸门：后续错误只能是业务原因，绝不能是 403。
func TestSaveStoryScheduling_ProductVisibleIsNotDenied(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	actor := &model.User{ID: 15864, Account: "demo_po"}
	expectStoryProductVisible(mock, 15864, 7003, 555)

	// 闸门放行后进入主系统反查；此处不提供期望，sqlmock 会让后续查询失败。
	err := svc.SaveStoryScheduling(t.Context(), actor, 7003, &SaveSchedulingReq{WindowID: 1})
	if bizErr, ok := errorx.IsBizError(err); ok && bizErr.Code == errorx.ErrCodeForbidden {
		t.Fatal("visible product must not be denied write access")
	}
}

// 4. 超管免关系查询直接放行。
func TestRequireStoryWriteAccess_SuperAdminAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	if err := svc.RequireStoryWriteAccess(t.Context(), &model.User{ID: 7, Account: "003030", IsSuperAdmin: true}, 7004); err != nil {
		t.Fatalf("super admin should pass story write authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("super admin must not trigger relation queries: %v", err)
	}
}

// 5. PMO 放行：不查产品可见性。
func TestRequireStoryWriteAccess_PMOAllowed(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(20), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := svc.RequireStoryWriteAccess(t.Context(), &model.User{ID: 20, Account: "pmo_user"}, 7005); err != nil {
		t.Fatalf("PMO should pass story write authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 6. 未登录账号（actor 为 nil）→ 403，不触发任何查询。
func TestRequireStoryWriteAccess_NilActorForbidden(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	assertForbidden(t, svc.RequireStoryWriteAccess(t.Context(), nil, 7006), StoryWriteDenialMessage)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("nil actor must not touch the database: %v", err)
	}
}

// 7. 非法 storyID → 参数错误而非 403。
func TestRequireStoryWriteAccess_InvalidStoryID(t *testing.T) {
	db, _ := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)

	err := svc.RequireStoryWriteAccess(t.Context(), &model.User{ID: 1, Account: "demo_po", IsSuperAdmin: true}, 0)
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam, got %v", err)
	}
}
