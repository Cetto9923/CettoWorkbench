// =============================================================================
// 文件: internal/module/testtask/repo_execution_test.go
// 模块: 提测办理
// 类型: action
// 职责: FindProductProjectIDs / FindExecutionsByProjectIDs / FindCRExecution /
//       FindDemandInvolvedProducts / ListInsideUsers 的 sqlmock 回归。
// 依赖: DATA-DOG/go-sqlmock
// =============================================================================

package testtask

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFindProductProjectIDs_StripsExecutionProject(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`(?s)SELECT DISTINCT\s+CASE WHEN p\.type.*FROM zt_projectproduct.*WHERE pp\.product = \?`).
		WithArgs(uint(11)).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).
			AddRow(uint64(101)).
			AddRow(uint64(102)).
			AddRow(uint64(101)))

	got, err := repo.FindProductProjectIDs(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 101 || got[1] != 102 {
		t.Fatalf("unexpected ids: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}

func TestFindProductProjectIDs_ZeroProduct(t *testing.T) {
	db, _ := setupMockTesttaskDB(t)
	repo := NewRepo(db)
	got, err := repo.FindProductProjectIDs(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len: got %d, want 0", len(got))
	}
}

func TestFindExecutionsByProjectIDs_TrimsAndSkipsZero(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`(?s)SELECT\s+e\.id.*FROM zt_project e.*AND e\.project IN \(\?\).*ORDER BY e\.project ASC, e\.id ASC`).
		WithArgs(driver.Value(uint(101))).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "project", "parent", "grade", "attribute", "status", "type", "project_name", "project_model"}).
			AddRow(uint64(0), "ignored", uint64(101), uint64(0), 1, "request", "wait", "sprint", "  P1  ", "scrum").
			AddRow(uint64(11), "  S1  ", uint64(101), uint64(0), 1, " request ", " wait ", "sprint", "  P1  ", "scrum"))

	got, err := repo.FindExecutionsByProjectIDs(context.Background(), []uint{101})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len: got %d, want 1 (%+v)", len(got), got)
	}
	if got[0].Name != "S1" || got[0].Attribute != "request" || got[0].Status != "wait" {
		t.Fatalf("trim mismatch: %+v", got[0])
	}
}

func TestFindExecutionsByProjectIDs_Empty(t *testing.T) {
	db, _ := setupMockTesttaskDB(t)
	repo := NewRepo(db)
	got, err := repo.FindExecutionsByProjectIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len: got %d, want 0", len(got))
	}
}

func TestFindCRExecution_ReturnsZeroOnMissing(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`SELECT .value. FROM .zt_config. WHERE module = \? AND .key. = \? ORDER BY id DESC LIMIT \?`).
		WithArgs("common", "CRExecution", 1).
		WillReturnError(sqlmock.ErrCancelled)

	got, err := repo.FindCRExecution(context.Background())
	if err == nil {
		t.Fatalf("expected ErrCancelled, got nil; got=%d", got)
	}
	if !errors.Is(err, sqlmock.ErrCancelled) {
		t.Fatalf("error chain missing sqlmock.ErrCancelled: %v", err)
	}
}

func TestFindCRExecution_ParsesValue(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`SELECT .value. FROM .zt_config. WHERE module = \? AND .key. = \? ORDER BY id DESC LIMIT \?`).
		WithArgs("common", "CRExecution", 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("1"))

	got, err := repo.FindCRExecution(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
}

func TestFindCRExecution_InvalidValueFallsBackZero(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`SELECT .value. FROM .zt_config. WHERE module = \? AND .key. = \? ORDER BY id DESC LIMIT \?`).
		WithArgs("common", "CRExecution", 1).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("not-a-number"))

	got, err := repo.FindCRExecution(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestFindDemandInvolvedProducts(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`(?s)SELECT DISTINCT p\.id, p\.name.*FROM zt_demandclarify dc.*WHERE dc\.demand = \?`).
		WithArgs(uint(63411)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uint64(0), "ignored").
			AddRow(uint64(11), "  打印中心  "))

	got, err := repo.FindDemandInvolvedProducts(context.Background(), 63411)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 11 || got[0].Name != "打印中心" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestListInsideUsers_FiltersEmptyAndDefaultsRealname(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`(?s)SELECT account, realname, pinyin\s+FROM zt_user\s+WHERE deleted = '0'\s+AND type = 'inside'`).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("", "no-account").
			AddRow("  003030  ", "  程锐  ").
			AddRow("kj001", ""))

	got, err := repo.ListInsideUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len: got %d, want 2 (%+v)", len(got), got)
	}
	if got[0].Account != "003030" || got[0].Realname != "程锐" {
		t.Fatalf("[0]: %+v", got[0])
	}
	if got[1].Account != "kj001" || got[1].Realname != "kj001" {
		t.Fatalf("[1]: %+v", got[1])
	}
}
