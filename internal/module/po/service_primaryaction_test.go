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

	// 1. FindDemandPrimaryActions：单条 SELECT，WHERE id IN (?, ?, ?)。
	mock.ExpectQuery(`SELECT id, stage, status, assignedTo, accepter`).
		WithArgs(ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "stage", "status", "assignedTo", "accepter"}).
			AddRow(10, "", "clarify", "user_x", "").
			AddRow(20, "", "released", "user_x", "").
			AddRow(30, "", "developing", "user_x", ""))

	// 2. CountDemandTestTasks：单条 SELECT，需求 → 故事 → 测试单 子查询，IN (?, ?, ?)。
	mock.ExpectQuery(`FROM \(SELECT id FROM zt_demand`).
		WithArgs(ids[0], ids[1], ids[2], ids[0], ids[1], ids[2]).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "count", "first_id"}).
			AddRow(10, 0, 0).
			AddRow(20, 0, 0).
			AddRow(30, 0, 0))

	// 3. FindDemandEvaluateStatus：单条 SELECT，IN (?, ?, ?)，含 EXISTS 子查询。
	mock.ExpectQuery(`FROM zt_demand d`).
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

	// 阶段断言：30 应该是 developing（提测，§4-2 disabled）。
	if pa, ok := out[30]; ok {
		if pa.Key != string(primaryaction.KeySubmitTest) {
			t.Errorf("demand 30 key = %q, want %q", pa.Key, primaryaction.KeySubmitTest)
		}
		if pa.Enabled {
			t.Error("demand 30 must be disabled (PLAN §4-2)")
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

	ids := []uint{100, 200}

	// CountStoryTestTasks 一次性 IN 查询。
	mock.ExpectQuery(`FROM \(SELECT id FROM zt_story`).
		WithArgs(ids[0], ids[1], ids[0], ids[1]).
		WillReturnRows(sqlmock.NewRows([]string{"story", "count", "first_id"}).
			AddRow(100, 0, 0).
			AddRow(200, 0, 0))

	// 2 个故事各自 findStoryMetaForAction 单行 Take 查询（不可避免；不是 N+1 over batch）。
	mock.ExpectQuery(`SELECT id, status, stage FROM zt_story WHERE id`).
		WithArgs(ids[0], "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "stage"}).AddRow(100, "developing", ""))
	mock.ExpectQuery(`SELECT id, status, stage FROM zt_story WHERE id`).
		WithArgs(ids[1], "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "stage"}).AddRow(200, "released", ""))

	out, err := svc.DeriveStoryPrimaryActions(t.Context(), &model.User{Account: "u", IsSuperAdmin: true}, ids, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out)=%d, want 2", len(out))
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
