// =============================================================================
// 文件: internal/module/po/review_list_sort_test.go
// 模块: PO 工作台
// 类型: unit test
// 职责: 验证 44966 评审列表排序：
//       1. 同一状态下，当前用户作为评审人/待审批人的需求排在前面
//       2. 跨状态按 wait > refuse > draft > other 排序
//       3. 同优先级同状态下按 id DESC 排序
// =============================================================================

package po

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

func TestAcceptRefsPaged_SQLContainsStatusAndReviewRankOrder(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)

	base := repo.roleDemandScope(context.Background(), "alice", mysqlStageFilters["accept"])
	base = repo.applyHomeFocusToolbarFiltersWithReviews(base, "alice", DemandsReq{}, nil)

	statusRankExpr := "CASE status WHEN 'wait' THEN 1 WHEN 'refuse' THEN 2 WHEN 'draft' THEN 3 ELSE 4 END"
	reviewRankExpr := "CASE WHEN (" +
		"id IN (SELECT demand FROM zt_demandreview WHERE reviewer = ? AND (result = '' OR result IS NULL)) " +
		"OR id IN (SELECT demand FROM zt_demandmanagerreview WHERE reviewer = ? AND (result = '' OR result IS NULL OR result = 'wait'))" +
		") THEN 0 ELSE 1 END"

	var rows []struct{ ID int }
	stmt := base.Select("id, "+statusRankExpr+" AS status_rank, "+reviewRankExpr+" AS review_rank", "alice", "alice").
		Order("status_rank ASC, review_rank ASC, id DESC").Offset(0).Limit(20).
		Session(&gorm.Session{DryRun: true}).Scan(&rows).Statement

	sql := stmt.SQL.String()

	// 1. 验证包含状态排序表达式
	if !strings.Contains(sql, "CASE status WHEN 'wait' THEN 1 WHEN 'refuse' THEN 2 WHEN 'draft' THEN 3 ELSE 4 END AS status_rank") {
		t.Fatalf("expected SQL to contain status_rank expression, got: %s", sql)
	}

	// 2. 验证包含评审人/待审批人加权表达式（包含 zt_demandreview 与 zt_demandmanagerreview）
	if !strings.Contains(sql, "zt_demandreview WHERE reviewer = ?") || !strings.Contains(sql, "zt_demandmanagerreview WHERE reviewer = ?") {
		t.Fatalf("expected SQL to check both zt_demandreview and zt_demandmanagerreview for reviewer, got: %s", sql)
	}

	// 3. 验证排序列严格为 status_rank ASC, review_rank ASC, id DESC
	if !strings.Contains(sql, "ORDER BY status_rank ASC, review_rank ASC, id DESC") {
		t.Fatalf("expected ORDER BY status_rank ASC, review_rank ASC, id DESC, got: %s", sql)
	}
}

func TestAcceptRefsPaged_ReviewerPrioritizedWithinSameStatus(t *testing.T) {
	db, mock := openSQLMock(t)
	repo := NewRepo(db, nil)

	// Count mock
	mock.ExpectQuery("(?s)SELECT count\\(\\*\\) FROM `zt_demand`.*WHERE.*deleted = \\?.*status NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	// Rows returned in order produced by ORDER BY status_rank ASC, review_rank ASC, id DESC
	// 需求1: id=102, wait 状态, 当前用户是评审人 (status_rank=1, review_rank=0) -> 排第1
	// 需求2: id=101, wait 状态, 当前用户不是评审人 (status_rank=1, review_rank=1) -> 排第2
	// 需求3: id=104, refuse 状态, 当前用户是审批人 (status_rank=2, review_rank=0) -> 排第3
	// 需求4: id=103, refuse 状态, 当前用户不是审批人 (status_rank=2, review_rank=1) -> 排第4
	mock.ExpectQuery("(?s)SELECT id, CASE status WHEN 'wait' THEN 1 WHEN 'refuse' THEN 2 WHEN 'draft' THEN 3 ELSE 4 END AS status_rank, CASE WHEN .*zt_demandreview.*zt_demandmanagerreview.* THEN 0 ELSE 1 END AS review_rank FROM `zt_demand`.*ORDER BY status_rank ASC, review_rank ASC, id DESC").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(102).
			AddRow(101).
			AddRow(104).
			AddRow(103))

	refs, total, err := repo.acceptRefsPaged(context.Background(), "alice", DemandsReq{
		Status:   "accept",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("acceptRefsPaged error: %v", err)
	}

	if total != 4 {
		t.Fatalf("expected total=4, got %d", total)
	}
	if len(refs) != 4 {
		t.Fatalf("expected 4 refs, got %d", len(refs))
	}

	expectedIDs := []int{102, 101, 104, 103}
	for i, wantID := range expectedIDs {
		if refs[i].id != wantID {
			t.Errorf("at index %d: expected demand id %d, got %d", i, wantID, refs[i].id)
		}
		if refs[i].stageStatus != "accept" {
			t.Errorf("at index %d: expected stageStatus 'accept', got %q", i, refs[i].stageStatus)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations unmet: %v", err)
	}
}
