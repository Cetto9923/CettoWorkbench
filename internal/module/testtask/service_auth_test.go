package testtask

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func TestListProductExecutionsRequiresAuthenticatedActor(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.ListProductExecutions(context.Background(), nil, 12)
	assertForbidden(t, err)
}

func TestListProductBuildsRequiresAuthenticatedActor(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.ListProductBuilds(context.Background(), nil, 12)
	assertForbidden(t, err)
}

func TestCreateBuildsRejectsInvalidDemandBeforeRemoteWrite(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)

	_, err := svc.CreateBuilds(context.Background(), &model.User{Account: "tester"}, 0, CreateBuildsReq{})
	if bizErr, ok := errorx.IsBizError(err); !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalid demand error, got %v", err)
	}
}

func TestCreateBuildsRejectsMissingDemandBeforeRemoteWrite(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)
	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(99999, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "main_system_id", "product_name", "estimate_launch"}))

	_, err := svc.CreateBuilds(context.Background(), &model.User{Account: "tester"}, 99999, CreateBuildsReq{Builds: []CreateBuildItem{{ProductID: 1, ProjectID: 2, ExecutionID: 3, Name: "v1", Date: "2026-09-11"}}})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeNotFound {
		t.Fatalf("expected notfound error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database calls: %v", err)
	}
}

func assertForbidden(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected authentication error, got nil")
	}
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected %q, got %v", errorx.ErrCodeForbidden, err)
	}
}
