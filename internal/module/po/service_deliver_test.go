// =============================================================================
// 文件: internal/module/po/service_deliver_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证发起交付状态门、权限门及表单数据加载与前置检查。
// =============================================================================

package po

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"workbench/internal/model"
)

func newServiceForDeliverTest(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	repo := NewRepo(db, db)
	return &Service{repo: repo}, mock
}

func TestDeliverDemand_Unauthenticated(t *testing.T) {
	s, _ := newServiceForDeliverTest(t)
	err := s.DeliverDemand(context.Background(), nil, DemandDeliverReq{ID: 1001})
	if !errors.Is(err, errHomeActionForbidden) {
		t.Fatalf("expected errHomeActionForbidden, got %v", err)
	}
}

func TestDeliverDemand_WrongStatus(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).
		WithArgs(uint(1002)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "deleted", "assignedTo", "distributedBy", "createdBy", "submitedBy", "submitBy", "QD", "RD", "BRA", "accepter", "veriFier", "deliverDate", "isCarReview", "isGrayVerifyPlan", "verifyDate", "verifyPlan", "verifyFinish", "product", "mainSystem",
		}).AddRow(
			1002, "测试业需", "testing", "0", "alice", "", "alice", "", "", "qd", "rd", "alice", "alice", "alice", "2026-09-15", "0", "0", "1", "验证计划", "2026-09-12", "1", "1",
		))

	err := s.DeliverDemand(context.Background(), &model.User{Account: "alice"}, DemandDeliverReq{
		ID:          1002,
		DeliverDate: "2026-09-15",
		Verifier:    "alice",
		VerifyPlan:  "验证计划",
	})
	if !errors.Is(err, errHomeActionConflict) {
		t.Fatalf("expected errHomeActionConflict for testing status, got %v", err)
	}
}

func TestDeliverDemand_NonOwnerForbidden(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).
		WithArgs(uint(1003)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "deleted", "assignedTo", "distributedBy", "createdBy", "submitedBy", "submitBy", "QD", "RD", "BRA", "accepter", "veriFier", "deliverDate", "isCarReview", "isGrayVerifyPlan", "verifyDate", "verifyPlan", "verifyFinish", "product", "mainSystem",
		}).AddRow(
			1003, "测试业需", "acceptanced", "0", "alice", "", "bob", "", "", "qd", "rd", "alice", "charlie", "charlie", "2026-09-15", "0", "0", "1", "验证计划", "2026-09-12", "1", "1",
		))

	err := s.DeliverDemand(context.Background(), &model.User{Account: "eve"}, DemandDeliverReq{
		ID:          1003,
		DeliverDate: "2026-09-15",
		Verifier:    "charlie",
		VerifyPlan:  "验证计划",
	})
	if !errors.Is(err, errHomeActionForbidden) {
		t.Fatalf("expected errHomeActionForbidden for non-owner, got %v", err)
	}
}

func TestGetDemandDeliverMeta_PrecheckCalculation(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	// 1. FindDeliverDemand
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).
		WithArgs(uint(1004)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "deleted", "assignedTo", "distributedBy", "createdBy", "submitedBy", "submitBy", "QD", "RD", "BRA", "accepter", "veriFier", "deliverDate", "isCarReview", "isGrayVerifyPlan", "verifyDate", "verifyPlan", "verifyFinish", "product", "mainSystem",
		}).AddRow(
			1004, "测试业需1004", "acceptanced", "0", "alice", "", "bob", "", "", "qd", "rd", "alice", "alice", "alice", "2026-09-20", "0", "1", "2", "详细验证计划", "2026-09-10", "1", "1",
		))

	// 2. CheckDeliverBlockers
	mock.ExpectQuery(`(?s)SELECT.*severe_count.*open_count.*FROM zt_bug`).
		WithArgs(uint(1004)).
		WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(0, 2))

	// 3. FindDemandLinkedWindow
	mock.ExpectQuery(`(?s)SELECT.*dw\.versionWindow.*FROM zt_demandwindow dw`).
		WithArgs(uint(1004)).
		WillReturnRows(sqlmock.NewRows([]string{"window_id", "window_name", "release_date"}).AddRow(20, "20260920窗口", "2026-09-20"))

	// 4. ListUpcomingDeliverWindows
	mock.ExpectQuery(`(?s)SELECT id, name.*FROM zt_versionwindow`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate"}).AddRow(20, "20260920窗口", "2026-09-20"))

	// 5. ListInsideUsersForDeliver
	mock.ExpectQuery(`(?s)SELECT account AS value.*FROM zt_user`).
		WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).AddRow("alice", "爱丽丝 (alice)"))

	meta, err := s.GetDemandDeliverMeta(context.Background(), &model.User{Account: "alice"}, 1004)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !meta.Precheck.CanSubmit {
		t.Fatalf("expected CanSubmit = true, got false, reason: %s", meta.Precheck.BlockReason)
	}
	if meta.WindowID != 20 {
		t.Fatalf("expected WindowID = 20, got %d", meta.WindowID)
	}
	if meta.Title != "测试业需1004" {
		t.Fatalf("expected Title = 测试业需1004, got %s", meta.Title)
	}
}

func TestDeliverDemand_BraOwnerAllowed(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	// assignedTo 为他人 (002397)，但 BRA 为当前用户 (003030)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_demand.*WHERE id = \?`).
		WithArgs(uint(1005)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "deleted", "assignedTo", "distributedBy", "createdBy", "submitedBy", "submitBy", "QD", "RD", "BRA", "accepter", "veriFier", "deliverDate", "isCarReview", "isGrayVerifyPlan", "verifyDate", "verifyPlan", "verifyFinish", "product", "mainSystem",
		}).AddRow(
			1005, "MCP定位服务", "acceptanced", "0", "002397", "", "009058", "", "", "004686", "009058", "003030", "003030", "003030", "2026-09-20", "0", "0", "1", "验证计划", "2026-09-10", "1", "1",
		))

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT id, status, deleted, product FROM .zt_demand. WHERE id = \? LIMIT \? FOR UPDATE`).
		WithArgs(uint(1005), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "product"}).AddRow(1005, "acceptanced", "0", "1"))
	mock.ExpectExec(`(?s)UPDATE .zt_demand. SET .* WHERE id = \? AND deleted = '0'`).
		WithArgs("2026-09-20", "0", "0", "003030", sqlmock.AnyArg(), "waitdeliver", "003030", "1", "验证计划", 1005).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE .zt_story. SET .* WHERE \(fromDemand = \? AND deleted = '0'\) AND \(closedReason IS NULL OR closedReason NOT IN \(\?,\?,\?,\?,\?\)\)`).
		WithArgs("2026-09-20", "0", "003030", "1", "验证计划", 1005, "willnotdo", "duplicate", "postponed", "cancel", "bydesign").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)INSERT INTO.*zt_action.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := s.DeliverDemand(context.Background(), &model.User{Account: "003030"}, DemandDeliverReq{
		ID:               1005,
		DeliverDate:      "2026-09-20",
		IsCarReview:      "0",
		IsGrayVerifyPlan: "0",
		VerifyDate:       "1",
		Verifier:         "003030",
		VerifyPlan:       "验证计划",
	})
	if err != nil {
		t.Fatalf("expected BRA owner allowed to deliver, got error: %v", err)
	}
}
