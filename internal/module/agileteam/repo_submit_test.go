// =============================================================================
// 文件: internal/module/agileteam/repo_submit_test.go
// 模块: 敏捷小组治理
// 类型: test
// 职责: 同一 teamgroup 提交调整必须在事务内加锁串行，最新 pending 生效且单号不能并发重复。
// =============================================================================

package agileteam

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
)

func TestSubmitPendingAdjustmentLocksTeamgroupBeforeSupersede(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	adj := &Adjustment{
		TeamgroupID: 3, Status: StatusPending, Reason: "扩编",
		SubmittedBy: "po1", CreatedBy: "po1", UpdatedBy: "po1",
		CreatedDate: now, UpdatedDate: now,
	}
	items := []AdjustmentItem{{
		Account: "newbie", ActionType: ActionAdd, Role: "研发", AvailableHours: 7,
		CreatedBy: "po1", UpdatedBy: "po1", CreatedDate: now, UpdatedDate: now,
	}}
	history := &History{TeamgroupID: 3, EventType: EventSubmit, Actor: "po1", CreatedDate: now}

	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT id, parent, grade.*FROM zt_teamgroup.*FOR UPDATE").
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment` SET .*`status`=\\?.*WHERE \\(teamgroupId = \\? AND status = \\?\\)").
		WithArgs(StatusSuperseded, "po1", sqlmock.AnyArg(), uint(3), StatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("(?s)SELECT `adjustNo` FROM `zt_wb_agileteam_adjustment`").
		WillReturnRows(sqlmock.NewRows([]string{"adjustNo"}))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_adjustment`").
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_adjustment_item`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_history`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	id, err := svc.repo.SubmitPendingAdjustmentAtomically(context.Background(), adj, items, history)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if id == 0 {
		t.Fatal("expected created adjustment id")
	}
	if adj.AdjustNo == "" || strings.Contains(adj.AdjustNo, "TA") == false {
		t.Fatalf("adjustNo not generated: %q", adj.AdjustNo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSubmitPendingAdjustmentSupersedesPreviousPendingInsideLock(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	adj := &Adjustment{
		TeamgroupID: 3, Status: StatusPending, Reason: "再提交",
		SubmittedBy: "po1", CreatedBy: "po1", UpdatedBy: "po1",
		CreatedDate: now, UpdatedDate: now,
	}

	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT id, parent, grade.*FROM zt_teamgroup.*FOR UPDATE").
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment` SET .*`status`=\\?.*WHERE \\(teamgroupId = \\? AND status = \\?\\)").
		WithArgs(StatusSuperseded, "po1", sqlmock.AnyArg(), uint(3), StatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("(?s)SELECT `adjustNo` FROM `zt_wb_agileteam_adjustment`").
		WillReturnRows(sqlmock.NewRows([]string{"adjustNo"}).AddRow("TA20260825001"))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_adjustment`").
		WillReturnResult(sqlmock.NewResult(12, 1))
	mock.ExpectCommit()

	id, err := svc.repo.SubmitPendingAdjustmentAtomically(context.Background(), adj, nil, nil)
	if err != nil {
		t.Fatalf("submit latest adjustment: %v", err)
	}
	if id != 12 {
		t.Fatalf("id=%d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSubmitPendingAdjustmentRetriesAdjustNoDuplicate(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	adj := &Adjustment{
		TeamgroupID: 3, Status: StatusPending, Reason: "扩编",
		SubmittedBy: "po1", CreatedBy: "po1", UpdatedBy: "po1",
		CreatedDate: now, UpdatedDate: now,
	}

	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT id, parent, grade.*FROM zt_teamgroup.*FOR UPDATE").
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(uint(3), uint(0), 1, ",3,"))
	mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment` SET .*`status`=\\?.*WHERE \\(teamgroupId = \\? AND status = \\?\\)").
		WithArgs(StatusSuperseded, "po1", sqlmock.AnyArg(), uint(3), StatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("(?s)SELECT `adjustNo` FROM `zt_wb_agileteam_adjustment`").
		WillReturnRows(sqlmock.NewRows([]string{"adjustNo"}).AddRow("TA20260825001"))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_adjustment`").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'TA20260825002' for key 'uk_adjustNo'"})
	mock.ExpectQuery("(?s)SELECT `adjustNo` FROM `zt_wb_agileteam_adjustment`").
		WillReturnRows(sqlmock.NewRows([]string{"adjustNo"}).AddRow("TA20260825002"))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_adjustment`").
		WillReturnResult(sqlmock.NewResult(12, 1))
	mock.ExpectCommit()

	id, err := svc.repo.SubmitPendingAdjustmentAtomically(context.Background(), adj, nil, nil)
	if err != nil {
		t.Fatalf("submit after retry: %v", err)
	}
	if id == 0 || adj.AdjustNo == "" {
		t.Fatalf("id=%d adjustNo=%q", id, adj.AdjustNo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIsAdjustNoDuplicate(t *testing.T) {
	if !isAdjustNoDuplicate(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry for key 'uk_adjustNo'"}) {
		t.Fatal("expected duplicate detect")
	}
	if isAdjustNoDuplicate(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry for key 'uk_other'"}) {
		t.Fatal("other unique key must not map to adjustNo conflict")
	}
}

func TestSubmitAdjustmentSerializesPendingInSource(t *testing.T) {
	b, err := os.ReadFile("repo_submit.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	for _, want := range []string{
		"LIMIT 1 FOR UPDATE",
		"StatusSuperseded",
		"nextAdjustNoTx(tx)",
		"SubmitPendingAdjustmentAtomically",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("atomic submit missing %q", want)
		}
	}
	svcSrc, err := os.ReadFile("service_adjustment.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(svcSrc)
	if strings.Contains(text[strings.Index(text, "func (s *Service) SubmitAdjustment"):strings.Index(text, "func countAction")], "FindPendingByTeamgroup(") {
		t.Fatal("SubmitAdjustment must not check pending outside the locked transaction")
	}
	if strings.Contains(text, "s.repo.NextAdjustNo(") {
		t.Fatal("adjust number must be generated inside the locked transaction")
	}
}
