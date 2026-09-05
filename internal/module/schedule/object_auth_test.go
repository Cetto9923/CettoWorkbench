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

	// Mock FindByID query with exact GORM First query pattern
	mock.ExpectQuery(`^SELECT \* FROM `+"`zt_versionwindow`"+` WHERE id = \? AND `+"`zt_versionwindow`"+`\.`+"`deletedAt`"+` IS NULL ORDER BY `+"`zt_versionwindow`"+`\.`+"`id`"+` LIMIT \?`).
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

	// 1. GetStoryTaskDetail: zt_story join zt_storyspec
	mock.ExpectQuery(`(?s)SELECT\s+s\.id,\s+s\.title,\s+s\.product.*FROM zt_story s`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "product", "fromDemand"}).
			AddRow(1, "Test Story", 100, 0))

	// 2. findStoryWindowDetail: zt_planstory join zt_versionwindowproduct join zt_versionwindow
	mock.ExpectQuery(`(?s)SELECT\s+vw\.id AS windowID.*FROM zt_planstory ps`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"windowID", "windowName", "releaseDate"}).
			AddRow(1, "Window", "2023-12-31"))

	// 3. findRecentProductTaskProject: zt_task join zt_story
	mock.ExpectQuery(`(?s)SELECT\s+t\.project AS projectId.*FROM zt_task t`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"projectId", "executionId"}).
			AddRow(1, 1))

	// 4. findStoryAttachments: zt_file
	mock.ExpectQuery(`(?s)SELECT id, title, extension\s+FROM zt_file`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "extension"}))

	// 5. GetUserProducts: zt_product with account checks
	mock.ExpectQuery(`(?s)SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist\s+FROM zt_product`).
		WithArgs("userA", "userA", "userA", "userA", "userA").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"})) // Return empty slice, user has no products

	// 6. GetProductsByIDs: unassigned product names
	mock.ExpectQuery(`(?s)SELECT id, name\s+FROM zt_product\s+WHERE id IN \(\?\) AND deleted = '0'`).
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
