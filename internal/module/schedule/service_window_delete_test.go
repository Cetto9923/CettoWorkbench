package schedule

import (
	"context"
	"strings"
	"testing"
	"workbench/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
)

// mockFindWindow 模拟 service.Delete 第一步 FindByID。
func mockFindWindow(mock sqlmock.Sqlmock, windowID uint64, createdBy string) {
	mock.ExpectQuery(`^SELECT \* FROM `+"`zt_versionwindow`"+` WHERE id = \? AND `+"`zt_versionwindow`"+`\.`+"`deletedAt`"+` IS NULL ORDER BY `+"`zt_versionwindow`"+`\.`+"`id`"+` LIMIT \?`).
		WithArgs(windowID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "createdBy"}).
			AddRow(windowID, "Test Window", createdBy))
}

// mockCountAssociations 模拟 CountWindowAssociations 的两条 count。
func mockCountAssociations(mock sqlmock.Sqlmock, windowID uint64, demandCount, productCount int) {
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM ` + "`zt_demandwindow`" + ` WHERE versionWindow = \? AND ` + "`zt_demandwindow`" + `\.` + "`deletedAt`" + ` IS NULL`).
		WithArgs(windowID).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(demandCount))
	mock.ExpectQuery(`(?s)SELECT count\(\*\) FROM ` + "`zt_versionwindowproduct`" + ` WHERE versionWindow = \? AND ` + "`zt_versionwindowproduct`" + `\.` + "`deletedAt`" + ` IS NULL`).
		WithArgs(windowID).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(productCount))
}

func TestDeleteWindow_WithDemandWindowRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	mockFindWindow(mock, 1, "userA")
	mockCountAssociations(mock, 1, 1, 0) // 有业需级窗口关联

	err := svc.Delete(context.Background(), &model.User{Account: "userA"}, DeleteReq{ID: 1})
	if err == nil {
		t.Fatal("expected error when window has demand-window association")
	}
	if !strings.Contains(err.Error(), "无法删除") {
		t.Fatalf("expected association rejection, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestDeleteWindow_WithProductRejected(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	mockFindWindow(mock, 1, "userA")
	mockCountAssociations(mock, 1, 0, 1) // 有窗口-产品关联

	err := svc.Delete(context.Background(), &model.User{Account: "userA"}, DeleteReq{ID: 1})
	if err == nil {
		t.Fatal("expected error when window has product association")
	}
	if !strings.Contains(err.Error(), "无法删除") {
		t.Fatalf("expected association rejection, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestDeleteWindow_NoAssociationSucceeds(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	mockFindWindow(mock, 1, "userA")
	mockCountAssociations(mock, 1, 0, 0) // 无任何关联
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE `+"`zt_versionwindow`"+` SET `+"`deletedAt`"+`=\? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := svc.Delete(context.Background(), &model.User{Account: "userA"}, DeleteReq{ID: 1})
	if err != nil {
		t.Fatalf("expected successful delete with no association, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
