package po

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/pkg/errorx"
)

func TestFindBoardIssueActionsReadsZenTaoAuditFromCreation(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	mock.ExpectQuery(`SELECT count\(\*\) FROM .*zt_issue.*`).
		WithArgs(int64(4212), "0", "po", "po").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	created := time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local)
	mock.ExpectQuery(`SELECT a.id, a.date, a.actor, COALESCE\(u.realname, a.actor\) AS actor_name`).
		WithArgs(int64(4212), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "date", "actor", "actor_name", "action", "extra", "comment"}).
			AddRow(101, created, "po", "产品负责人", "opened", "", "创建问题"))

	page, err := repo.FindBoardIssueActions(context.Background(), "po", 4212, 0)
	if err != nil {
		t.Fatalf("FindBoardIssueActions() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Action != "opened" || page.Items[0].ActorName != "产品负责人" {
		t.Fatalf("unexpected actions: %+v", page.Items)
	}
	if page.Items[0].Date != "2026-09-07 10:00:00" {
		t.Fatalf("action date = %q", page.Items[0].Date)
	}
	if page.NextAfterID != 0 {
		t.Fatalf("NextAfterID = %d, want 0", page.NextAfterID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindBoardIssueActionsRejectsInvisibleIssue(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	mock.ExpectQuery(`SELECT count\(\*\) FROM .*zt_issue.*`).
		WithArgs(int64(4212), "0", "po", "po").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	page, err := repo.FindBoardIssueActions(context.Background(), "po", 4212, 0)
	if page != nil {
		t.Fatalf("page = %+v, want nil", page)
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("error = %v, want forbidden", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
