package schedule

import (
	"context"
	"strings"
	"testing"
	"workbench/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupMockDB sets up an sqlmock DB.
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening gorm database", err)
	}

	return gormDB, mock
}

func TestUpdateWindow_OwnerAuth(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	// Mock FindByID query
	mock.ExpectQuery(".*").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "createdBy"}).
			AddRow(1, "Test Window", "userA"))

	ctx := context.Background()
	req := UpdateReq{
		ID:          1,
		Name:        "New Name",
		ReleaseDate: "2023-12-31",
	}

	actorB := &model.User{Account: "userB"}
	err := svc.Update(ctx, actorB, req)
	if err == nil {
		t.Fatal("expected error when userB updates userA's window")
	}
	if !strings.Contains(err.Error(), "只有创建人可以修改") {
		t.Fatalf("expected owner error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSaveStoryTasks_ProductScopeAuth(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil)

	// Mock GetStoryTaskDetail (the raw query from repo_story_tasks.go)
	mock.ExpectQuery(".*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "product", "fromDemand"}).
			AddRow(1, "Test Story", 100, 0))

	mock.ExpectQuery(".*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"windowID", "windowName", "releaseDate"}).
			AddRow(1, "Window", "2023-12-31"))

	mock.ExpectQuery(".*").
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"projectId", "executionId"}).
			AddRow(1, 1))

	mock.ExpectQuery(".*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "extension"}))

	// GetUserProducts
	mock.ExpectQuery(".*").
		WithArgs("userA", "userA", "userA", "userA", "userA").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"})) // Return empty slice, user has no products

	mock.ExpectQuery(".*").
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(100, "Test Product"))

	ctx := context.Background()
	actor := &model.User{Account: "userA"}
	req := &SaveStoryTasksReq{
		Tasks: []SaveStoryTasksTask{
			{Action: "edit", ID: 1, Name: "Test Task"},
		},
	}

	err := svc.SaveStoryTasks(ctx, actor, 1, req)
	if err == nil {
		t.Fatal("expected error due to lack of product scope access")
	}
	if !strings.Contains(err.Error(), "存在未在窗口中且无匹配计划的系统") {
		// "存在未在窗口中且无匹配计划的系统" is the message from ProductAccessNoticeError
		t.Fatalf("expected ProductAccessNoticeError, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
