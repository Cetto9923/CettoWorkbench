package testtask

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"workbench/internal/model"
)

func jointRequest() CreateTesttasksReq {
	return CreateTesttasksReq{Joint: 1, Products: []uint{1, 2}, Builds: [][]uint{{101}, {102}}, Name: "joint", Begin: "2026-09-12", End: "2026-09-13", Owner: "qa-owner", Members: []string{"dev", "qa-owner", "dev"}, Type: "feature", Pri: 2}
}

func TestJointSubmissionUsesActorAndSelectedOwner(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 2, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).AddRow(101, 1, 11, 22, "a").AddRow(102, 2, 11, 23, "b"))
	svc := NewService(NewRepo(db), nil, zt.client(), nil)
	resp, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "joint-operator"}, 123, jointRequest())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Tasks) != 1 || resp.Tasks[0].TesttaskID == 0 {
		t.Fatalf("missing created task: %+v", resp)
	}
	payload := zt.lastPayload.Load().(map[string]any)
	if payload["owner"] != "qa-owner" || payload["joint"] != "1" {
		t.Fatalf("bad payload: %+v", payload)
	}
	if zt.tokenAccount.Load() != "joint-operator" {
		t.Fatalf("operator identity lost: %v", zt.tokenAccount.Load())
	}
	if len(payload["members"].([]any)) != 2 {
		t.Fatal("members should include owner exactly once")
	}
	if len(payload["builds"].(map[string]any)) != 2 {
		t.Fatal("product/build alignment lost")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestJointRejectsCrossProjectBeforeWriting(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 2, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).AddRow(101, 1, 11, 22, "a").AddRow(102, 2, 12, 23, "b"))
	_, err := NewService(NewRepo(db), nil, zt.client(), nil).CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, jointRequest())
	if err == nil || zt.createCalls.Load() != 0 {
		t.Fatal("cross-project request must not write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIndependentKeepsSelectedOwner(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).AddRow(101, 1, 11, 22, "a"))
	expectProjectBelongs(mock, 22, 11, 1)
	_, err := NewService(NewRepo(db), nil, zt.client(), nil).CreateTesttasks(context.Background(), &model.User{Account: "independent-operator"}, 123, CreateTesttasksReq{Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 101, Owner: "qa-owner", Name: "test", Begin: "2026-09-12", End: "2026-09-13", Type: "integrate", Pri: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if zt.lastPayload.Load().(map[string]any)["owner"] != "qa-owner" || zt.tokenAccount.Load() != "independent-operator" {
		t.Fatal("actor and owner must stay separate")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLaterInvalidBuildPreventsEarlierWrite(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 2, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).AddRow(101, 1, 11, 22, "a"))
	expectProjectBelongs(mock, 22, 11, 1)
	item := CreateTesttaskItem{ProductID: 1, BuildID: 101, Name: "test", Begin: "2026-09-12", End: "2026-09-13", Owner: "qa", Type: "feature", Pri: 2}
	missing := item
	missing.BuildID = 102
	_, err := NewService(NewRepo(db), nil, zt.client(), nil).CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{Tasks: []CreateTesttaskItem{item, missing}})
	if err == nil || zt.createCalls.Load() != 0 {
		t.Fatal("whole batch must validate before the first write")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
