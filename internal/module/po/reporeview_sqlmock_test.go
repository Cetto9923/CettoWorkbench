// =============================================================================
// 文件: internal/module/po/reporeview_sqlmock_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 sqlmock 锁定 SaveDemandReview 发出的 SQL 语句与参数，作为拆分前后的
//       行为基准；拆分只允许搬运，不允许改变任何一条 SQL。
// =============================================================================

package po

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// lockDemandSQL 主表加锁查询，Select 列表与锁语义不可改动。
var lockDemandSQL = regexp.QuoteMeta("SELECT id, status, deleted, createdBy, reviewedBy, " +
	"mailto, isNeedFocus, product FROM `zt_demand` WHERE id = ? LIMIT ? FOR UPDATE")

// loadReviewerSQL 取当前评审人既有结论的查询。
var loadReviewerSQL = regexp.QuoteMeta("SELECT `result` FROM `zt_demandreview` " +
	"WHERE demand = ? AND reviewer = ? LIMIT ?")

// countUnpassedSQL 统计剩余未通过评审人的查询。
var countUnpassedSQL = regexp.QuoteMeta("SELECT count(*) FROM `zt_demandreview` " +
	"WHERE demand = ? AND reviewer <> ? AND (result IS NULL OR result <> ?)")

// expectLockedDemand 声明一次 FOR UPDATE 锁行查询。
func expectLockedDemand(mock sqlmock.Sqlmock, demandID int64, status string) {
	mock.ExpectQuery(lockDemandSQL).
		WithArgs(demandID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "deleted", "createdBy", "reviewedBy",
			"mailto", "isNeedFocus", "product",
		}).AddRow(demandID, status, "0", "creator_a", "", nil, "0", "1"))
}

// TestSaveDemandReview_PassAllApproved 锁定「通过 + 无剩余评审人」路径的完整 SQL 序列。
func TestSaveDemandReview_PassAllApproved(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	const demandID int64 = 1001
	const account = "reviewer_a"

	mock.ExpectBegin()
	expectLockedDemand(mock, demandID, "wait")

	mock.ExpectQuery(loadReviewerSQL).
		WithArgs(demandID, account, 1).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(""))

	// 写回 zt_demandreview 的 result / reviewDate
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_demandreview` SET `result`=?,`reviewDate`=?")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 统计未通过的其余评审人 → 为 0 则状态置 active 并追加 reviewpassed
	mock.ExpectQuery(countUnpassedSQL).
		WithArgs(demandID, "", "pass").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 更新 zt_demand 主表
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_demand` SET")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 操作历史：reviewed + reviewpassed 两条
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `zt_action`")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `zt_action`")).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	mailto := "reviewer_a@example.com"
	err := repo.SaveDemandReview(t.Context(), saveDemandReviewIn{
		DemandID:    demandID,
		Account:     account,
		Result:      "pass",
		IsNeedFocus: "0",
		Mailto:      &mailto,
		ReviewedBy:  account,
		Comment:     "ok",
		Product:     "1",
	})
	if err != nil {
		t.Fatalf("SaveDemandReview 失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期（说明拆分改变了 SQL 序列）: %v", err)
	}
}

// TestSaveDemandReview_PassPendingOthers 通过但仍有他人未评：不改状态、不写状态动作。
func TestSaveDemandReview_PassPendingOthers(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	const demandID int64 = 1004
	const account = "reviewer_d"

	mock.ExpectBegin()
	expectLockedDemand(mock, demandID, "wait")

	mock.ExpectQuery(loadReviewerSQL).
		WithArgs(demandID, account, 1).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(""))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_demandreview` SET `result`=?,`reviewDate`=?")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(countUnpassedSQL).
		WithArgs(demandID, "", "pass").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_demand` SET")).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 仍会写 reviewed，但因状态未流转不追加 reviewpassed
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `zt_action`")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	if err := repo.SaveDemandReview(t.Context(), saveDemandReviewIn{
		DemandID: demandID,
		Account:  account,
		Result:   "pass",
		Product:  "1",
	}); err != nil {
		t.Fatalf("SaveDemandReview 失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期（他人未评完时不应追加状态动作）: %v", err)
	}
}

// TestSaveDemandReview_AlreadyReviewed 已评审直接短路，只走锁行 + 读结论两条 SQL。
func TestSaveDemandReview_AlreadyReviewed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	const demandID int64 = 1002
	const account = "reviewer_b"

	mock.ExpectBegin()
	expectLockedDemand(mock, demandID, "wait")

	mock.ExpectQuery(loadReviewerSQL).
		WithArgs(demandID, account, 1).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow("pass"))

	if err := repo.SaveDemandReview(t.Context(), saveDemandReviewIn{
		DemandID: demandID,
		Account:  account,
		Result:   "pass",
		Product:  "1",
	}); err != errAlreadyReviewed {
		t.Fatalf("已评审应短路返回 errAlreadyReviewed，得到 %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期（短路后不应再发 SQL）: %v", err)
	}
}

// TestSaveDemandReview_NotReviewable 状态非 wait 时在任何写入前短路。
func TestSaveDemandReview_NotReviewable(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	const demandID int64 = 1003

	mock.ExpectBegin()
	expectLockedDemand(mock, demandID, "active")

	if err := repo.SaveDemandReview(t.Context(), saveDemandReviewIn{
		DemandID: demandID,
		Account:  "reviewer_c",
		Result:   "pass",
		Product:  "1",
	}); err != errDemandNotReviewable {
		t.Fatalf("非待评审状态应返回 errDemandNotReviewable，得到 %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期（短路后不应再发 SQL）: %v", err)
	}
}
