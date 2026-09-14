package dept

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
)

func setupDeptMockDB(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm: %v", err)
	}
	return NewRepo(gormDB), mock
}

func TestCreateEnablingAncestors(t *testing.T) {
	repo, mock := setupDeptMockDB(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM `zt_depts`").
		WithArgs(10, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "parentId", "name", "leader", "phone", "email", "status", "sort", "createdAt", "updatedAt", "deletedAt",
		}).AddRow(10, 1, "parent", "", "", "", 1, 1, nil, nil, nil))
	mock.ExpectQuery("SELECT .* FROM `zt_depts`").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "parentId", "name", "leader", "phone", "email", "status", "sort", "createdAt", "updatedAt", "deletedAt",
		}).AddRow(1, 0, "root", "", "", "", 0, 1, nil, nil, nil))
	mock.ExpectExec("UPDATE `zt_depts`").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO `zt_depts`").
		WillReturnResult(sqlmock.NewResult(99, 1))
	mock.ExpectCommit()

	child := &model.Dept{
		ParentID: 10,
		Name:     "child",
		Status:   0,
		Sort:     1,
	}
	if err := repo.CreateEnablingAncestors(ctx, 10, child); err != nil {
		t.Fatalf("CreateEnablingAncestors: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}
