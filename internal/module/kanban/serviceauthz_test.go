// =============================================================================
// 文件: internal/module/kanban/serviceauthz_test.go
// 模块: 工作看板
// 类型: test
// 职责: 验证任务范围拦截及真实操作人审计字段，不连接真实库。
// =============================================================================
package kanban

import (
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func kanbanMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db, mock
}

func TestTaskOutsideMemberScopeDenied(t *testing.T) {
	for _, account := range []string{"outsider", ""} {
		t.Run(account, func(t *testing.T) {
			db, mock := kanbanMockDB(t)
			mock.ExpectQuery("SELECT id, status, assignedTo FROM `zt_task`").WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo"}).AddRow(5, "wait", account))
			if account != "" {
				mock.ExpectQuery("SELECT g.id, g.name").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "PO", "manager"}))
			}
			svc := NewService(NewRepo(db), nil, nil, nil)
			err := svc.UpdateTaskStatus(t.Context(), &model.User{ID: 3, Account: "tester"}, UpdateTaskStatusReq{ID: 5, Status: "doing"})
			biz, ok := errorx.IsBizError(err)
			if !ok || biz.Code != errorx.ErrCodeForbidden {
				t.Fatalf("want forbidden got %v", err)
			}
		})
	}
}

func TestTaskEditorIsActualActor(t *testing.T) {
	for _, pair := range [][2]string{{"wait", "doing"}, {"doing", "wait"}} {
		body := buildKanbanTaskUpdateBody(pair[0], pair[1], "actor", time.Now())
		if body["lastEditedBy"] != "actor" {
			t.Fatalf("wrong editor: %v", body)
		}
	}
}
