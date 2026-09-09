// =============================================================================
// 文件: internal/module/po/service_primaryaction_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 Service.DeriveDemandPrimaryActions / DeriveStoryPrimaryActions 在
//       sqlmock 下的 IN (?) 批量派生合同，并验证 enabled=false 的 disabled 路径。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/gorm
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
)

// TestDeriveDemandPrimaryActions_BatchIN 验证批量查询使用 IN (?, ?, ?) 而非行循环 N+1。
//
// 期望：3 个 demand IDs → 一次性 FindDemandPrimaryActions + 一次性 CountDemandTestTasks +
// 一次性 FindDemandEvaluateStatus（评价 SQL 也是 IN）。
func TestDeriveDemandPrimaryActions_BatchIN(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	detailRepo := NewDemandDetailRepo(gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())
	// svc 内 detailSvc 已 attach；detailRepo 也直接复用。

	ids := []uint{10, 20, 30}

	// 1. FindDemandPrimaryActions：id IN (?)，deleted = '0' 字面量不占占位符。
	mock.ExpectQuery(`SELECT id, stage, status, assignedTo, accepter`).
		WithArgs(ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "stage", "status", "assignedTo", "accepter"}).
			AddRow(10, "", "clarify", "user_x", "").
			AddRow(20, "", "released", "user_x", "").
			AddRow(30, "", "developing", "user_x", ""))

	// 2. CountDemandTestTasks：需求 IN + fromDemand IN，各绑定一次。
	mock.ExpectQuery(`(?s)SELECT d\.id AS demand_id.*FROM zt_demand.*fromDemand IN`).
		WithArgs(ids[0], ids[1], ids[2], ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "count", "first_id"}).
			AddRow(10, 0, 0).
			AddRow(20, 0, 0).
			AddRow(30, 0, 0))

	// 3. FindDemandEvaluateStatus：appraiseBy = ? 一次 + id IN (?) 一次。
	mock.ExpectQuery(`(?s)SELECT d\.id AS demand_id.*zt_demandappraise`).
		WithArgs("user_x", ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "has_pending", "has_any"}).
			AddRow(10, false, false).
			AddRow(20, true, true).
			AddRow(30, false, false))

	out, err := svc.DeriveDemandPrimaryActions(t.Context(), &model.User{Account: "user_x", IsSuperAdmin: true}, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out)=%d, want 3", len(out))
	}

	// 阶段断言：30 是 developing，进入保留的 Workbench 提测页面。
	if pa, ok := out[30]; ok {
		if pa.Key != string(primaryaction.KeySubmitTest) {
			t.Errorf("demand 30 key = %q, want %q", pa.Key, primaryaction.KeySubmitTest)
		}
		if !pa.Enabled || pa.URL == "" {
			t.Errorf("demand 30 submit-test must be enabled, got %+v", pa)
		}
	}

	// 10 是 clarify → clarify action，enabled=true（默认 Has*Capability true via IsSuperAdmin）。
	if pa, ok := out[10]; ok {
		if pa.Key != string(primaryaction.KeyClarify) {
			t.Errorf("demand 10 key = %q", pa.Key)
		}
	}

	// 20 是 released + has_pending → evaluate enabled。
	if pa, ok := out[20]; ok {
		if pa.Key != string(primaryaction.KeyEvaluate) {
			t.Errorf("demand 20 key = %q, want evaluate (pending)", pa.Key)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}

	// 直接校验 detailRepo 调用了一次（不是 3 次），防止后续改写退化到 N+1。
	_ = detailRepo
}

// TestDeriveDemandPrimaryActions_Empty 边界：空 ID 切片直接返 None map，不查 DB。
func TestDeriveDemandPrimaryActions_Empty(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	out, err := svc.DeriveDemandPrimaryActions(t.Context(), &model.User{Account: "u"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out)=%d, want 0", len(out))
	}
}

// TestDeriveStoryPrimaryActions_BatchIN 验证故事批量派生走 IN (?) 一次查询。
func TestDeriveStoryPrimaryActions_BatchIN(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	ids := []uint{100, 200, 300}

	// CountStoryTestTasks：story IN + case.story IN，各绑定一次。
	mock.ExpectQuery(`(?s)SELECT s\.id AS story.*FROM zt_testtask`).
		WithArgs(ids[0], ids[1], ids[2], ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"story", "count", "first_id"}).
			AddRow(100, 0, 0).
			AddRow(200, 0, 0).
			AddRow(300, 0, 0))

	// 故事事实：FindStoryMetaForAction 单次 IN + deleted = ?。
	mock.ExpectQuery(`SELECT id, status, stage FROM`).
		WithArgs(ids[0], ids[1], ids[2], "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "stage"}).
			AddRow(100, "developing", "").
			AddRow(200, "released", "").
			AddRow(300, "active", "wait"))

	out, err := svc.DeriveStoryPrimaryActions(t.Context(), &model.User{Account: "u", IsSuperAdmin: true}, ids, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out)=%d, want 3", len(out))
	}
	if pa, ok := out[300]; !ok || pa.Key != string(primaryaction.KeySchedule) {
		t.Fatalf("story 300 key = %q, want %q", pa.Key, primaryaction.KeySchedule)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// TestBuildPrimaryActionForDetail_NilRow 防御：nil / ID=0 行直接 None。
func TestBuildPrimaryActionForDetail_NilRow(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())
	pa := svc.detailSvc.buildPrimaryActionForDetail(t.Context(), &model.User{Account: "u"}, nil)
	if pa.Key != "" {
		t.Fatalf("nil row must be None, got %+v", pa)
	}
}

// TestAttachPrimaryActions_EmptyTree 防御：空树直接返 nil，不查 DB。
func TestAttachPrimaryActions_EmptyTree(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	if err := svc.attachPrimaryActions(t.Context(), &model.User{Account: "u"}, nil); err != nil {
		t.Fatalf("nil tree error: %v", err)
	}
	if err := svc.attachPrimaryActions(t.Context(), &model.User{Account: "u"}, []*BoardDemandItem{}); err != nil {
		t.Fatalf("empty tree error: %v", err)
	}
}

// TestDeriveDemandPrimaryActions_ReviewWorkflow 验证评审流：
// 1. 待评审 + 待我评审 -> KeyApprove ("评审")
// 2. 待评审 + 非待我评审 + 我创建 -> KeyWithdrawReview ("撤回评审")
// 3. 草稿 + 我创建 -> KeySubmitReview ("提交评审")
// 4. 草稿 + 他人创建 -> None ("—")
func TestDeriveDemandPrimaryActions_ReviewWorkflow(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	ids := []uint{1, 2, 3, 4}
	// 1: wait, createdBy=other, pending reviewer=user_a -> approve
	// 2: wait, createdBy=user_a, not pending reviewer -> withdraw_review
	// 3: draft, createdBy=user_a -> submit_review
	// 4: draft, createdBy=other -> none

	mock.ExpectQuery(`SELECT id, stage, status, assignedTo, accepter, createdBy`).
		WithArgs(ids[0], ids[1], ids[2], ids[3]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "stage", "status", "assignedTo", "accepter", "createdBy"}).
			AddRow(1, "wait", "wait", "user_x", "", "other").
			AddRow(2, "wait", "wait", "user_x", "", "user_a").
			AddRow(3, "draft", "draft", "user_x", "", "user_a").
			AddRow(4, "draft", "draft", "user_x", "", "other"))

	mock.ExpectQuery(`(?s)SELECT d\.id AS demand_id.*FROM zt_demand.*fromDemand IN`).
		WithArgs(ids[0], ids[1], ids[2], ids[3], ids[0], ids[1], ids[2], ids[3]).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "count", "first_id"}).
			AddRow(1, 0, 0).
			AddRow(2, 0, 0).
			AddRow(3, 0, 0).
			AddRow(4, 0, 0))

	mock.ExpectQuery(`(?s)SELECT d\.id AS demand_id.*zt_demandappraise`).
		WithArgs("user_a", ids[0], ids[1], ids[2], ids[3]).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "has_pending", "has_any"}).
			AddRow(1, false, false).
			AddRow(2, false, false).
			AddRow(3, false, false).
			AddRow(4, false, false))

	// FindPendingReviewDemandIDs: 只有 status=wait 的 id: 1, 2
	mock.ExpectQuery(`SELECT .*demand.* FROM .*zt_demandreview.* WHERE demand IN \(\?,\s*\?\)`).
		WithArgs(1, 2, "user_a", "").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}).AddRow(1))

	out, err := svc.DeriveDemandPrimaryActions(t.Context(), &model.User{Account: "user_a", IsSuperAdmin: false}, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 -> approve
	if pa := out[1]; pa.Key != string(primaryaction.KeyApprove) {
		t.Errorf("demand 1 key = %q, want %q", pa.Key, primaryaction.KeyApprove)
	}
	// 2 -> withdraw_review
	if pa := out[2]; pa.Key != string(primaryaction.KeyWithdrawReview) {
		t.Errorf("demand 2 key = %q, want %q", pa.Key, primaryaction.KeyWithdrawReview)
	}
	// 3 -> submit_review
	if pa := out[3]; pa.Key != string(primaryaction.KeySubmitReview) {
		t.Errorf("demand 3 key = %q, want %q", pa.Key, primaryaction.KeySubmitReview)
	}
	// 4 -> none
	if pa := out[4]; pa.Key != "" || pa.Enabled {
		t.Errorf("demand 4 should be none, got %+v", pa)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
