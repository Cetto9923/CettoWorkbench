// =============================================================================
// 文件: internal/module/po/service_detail_writeauthz_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 详情页写权限闸门回归：RequireDemandWrite 的关系口径，
//       以及无写权限时 primaryAction / canEdit / canWithdrawReview 的降级。
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/errorx"
)

const (
	writeAuthzPMOQuery     = `SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = 'pmo'`
	writeAuthzRelatedQuery = `SELECT COUNT\(\*\) FROM zt_demand d`
	writeAuthzManagedQuery = `SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`
)

// 1. 干系人可写。
func TestRequireDemandWrite_StakeholderAllowed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	svc := NewDetailService(NewDemandDetailRepo(gormDB))

	mock.ExpectQuery(writeAuthzPMOQuery).
		WithArgs(int64(15864)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(writeAuthzRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := svc.RequireDemandWrite(t.Context(), &model.User{ID: 15864, Account: "demo_po"}, 63450); err != nil {
		t.Fatalf("stakeholder should be allowed to write, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 2. 超管可写，且不触发关系查询。
func TestRequireDemandWrite_SuperAdminAllowed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	svc := NewDetailService(NewDemandDetailRepo(gormDB))

	if err := svc.RequireDemandWrite(t.Context(), &model.User{ID: 7, Account: "003030", IsSuperAdmin: true}, 63450); err != nil {
		t.Fatalf("super admin should be allowed to write, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("super admin must not trigger relation queries: %v", err)
	}
}

// 3. 团队长管辖可写。
func TestRequireDemandWrite_LeaderScopeAllowed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	svc := NewDetailService(NewDemandDetailRepo(gormDB))

	mock.ExpectQuery(writeAuthzPMOQuery).
		WithArgs(int64(15865)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(writeAuthzRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(writeAuthzManagedQuery).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",53,52,14,"))
	mock.ExpectQuery(`SELECT DISTINCT id FROM zt_dept WHERE id IN \(\?\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))
	mock.ExpectQuery(writeAuthzRelatedQuery).
		WithArgs(uint(63457), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if err := svc.RequireDemandWrite(t.Context(), &model.User{ID: 15865, Account: "demo_leader"}, 63457); err != nil {
		t.Fatalf("dept leader in scope should be allowed to write, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 4. 无任何关系（非干系人、非团队长）→ 403。
func TestRequireDemandWrite_NoRelationForbidden(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	svc := NewDetailService(NewDemandDetailRepo(gormDB))

	mock.ExpectQuery(writeAuthzPMOQuery).
		WithArgs(int64(15865)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(writeAuthzRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(writeAuthzManagedQuery).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	err := svc.RequireDemandWrite(t.Context(), &model.User{ID: 15865, Account: "demo_leader"}, 63450)
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 5. 无写权限时主操作被降级为不可点击，且不再下发跳转 URL。
func TestGatePrimaryActionForReadOnly_DisablesActionAndClearsURL(t *testing.T) {
	pa := primaryaction.PrimaryAction{
		Key:     string(primaryaction.KeySubmitTest),
		Label:   "提测",
		Kind:    string(primaryaction.KindDrawer),
		URL:     "/po/demands/63450/submit-test",
		Enabled: true,
	}
	gatePrimaryActionForReadOnly(&pa)

	if pa.Enabled {
		t.Fatal("primaryAction.enabled must be false for read-only viewers")
	}
	if pa.URL != "" {
		t.Fatalf("primaryAction.url must be cleared for read-only viewers, got %q", pa.URL)
	}
	if pa.Reason != ReadOnlyDemandReason {
		t.Fatalf("primaryAction.reason = %q, want %q", pa.Reason, ReadOnlyDemandReason)
	}
}

// 6. 无写权限时 canEdit 降级（即便生命周期规则本身允许编辑）。
func TestPopulateDemandEditability_ReadOnlyViewerDowngradesCanEdit(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	svc := NewDetailService(NewDemandDetailRepo(gormDB))

	// 每次调用都会查一次已出评审结论的人数；本用例调用两次。
	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 生命周期规则本身：draft + 创建人 → canEdit = true。
	actor := &model.User{ID: 15865, Account: "demo_biz"}
	row := &DemandDetailRow{ID: 63450, Status: "draft", CreatedBy: "demo_biz", AssignedTo: ""}

	summary := &DemandSummary{}
	svc.populateDemandEditability(t.Context(), actor, row, false, summary)
	if summary.CanEdit {
		t.Fatal("canEdit must be false without write permission")
	}
	if summary.EditDisabledReason != ReadOnlyDemandReason {
		t.Fatalf("EditDisabledReason = %q, want %q", summary.EditDisabledReason, ReadOnlyDemandReason)
	}

	// 有写权限时保持原生命周期结论。
	summary2 := &DemandSummary{}
	svc.populateDemandEditability(t.Context(), actor, row, true, summary2)
	if !summary2.CanEdit {
		t.Fatal("canEdit must stay true for a writable creator")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}
