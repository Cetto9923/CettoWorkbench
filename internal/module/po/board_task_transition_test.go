package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

type fakeTaskStatusGateway struct {
	called bool
	params zentao.TaskStatusParams
}

func (f *fakeTaskStatusGateway) UpdateTaskStatus(ctx context.Context, p zentao.TaskStatusParams) error {
	f.called = true
	f.params = p
	return nil
}

func newBoardTaskTransitionService(t *testing.T) (*Service, sqlmock.Sqlmock, *fakeTaskStatusGateway) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	gateway := &fakeTaskStatusGateway{}
	svc := NewService(NewRepo(db, db), nil, nil, zap.NewNop())
	svc.SetTaskStatusGateway(gateway)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return svc, mock, gateway
}

func TestTransitionBoardTaskDoneDefaultsFinisherToCurrentAssignee(t *testing.T) {
	svc, mock, gateway := newBoardTaskTransitionService(t)
	mock.ExpectQuery("SELECT id, status, assignedTo FROM `zt_task`").
		WithArgs(216573, "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo"}).AddRow(216573, "doing", "dev_a"))

	err := svc.TransitionBoardTask(context.Background(), &model.User{Account: "003030", IsSuperAdmin: true}, 216573, BoardTaskTransitionReq{Status: "done"})
	if err != nil {
		t.Fatalf("TransitionBoardTask returned error: %v", err)
	}
	if !gateway.called {
		t.Fatal("expected zentao task status gateway call")
	}
	if gateway.params.FinishedBy != "dev_a" {
		t.Fatalf("FinishedBy = %q, want current assignee", gateway.params.FinishedBy)
	}
	if gateway.params.Status != "done" || gateway.params.TaskID != 216573 || gateway.params.Account != "003030" {
		t.Fatalf("unexpected params: %+v", gateway.params)
	}
	if gateway.params.FinishedDate == "" {
		t.Fatal("FinishedDate should default to current time")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardTaskTransitionReqValidate(t *testing.T) {
	req := BoardTaskTransitionReq{Status: "已完成", FinishedDate: "2026-09-11T12:30"}
	if errs := req.Validate(); len(errs) > 0 {
		t.Fatalf("Validate returned errors: %+v", errs)
	}
	if req.Status != "done" {
		t.Fatalf("Status = %q, want done", req.Status)
	}

	bad := BoardTaskTransitionReq{Status: "closed"}
	if errs := bad.Validate(); len(errs) == 0 {
		t.Fatal("expected invalid status error")
	}
}
