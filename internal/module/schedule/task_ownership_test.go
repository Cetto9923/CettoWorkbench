package schedule

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidateTaskOwnership_UsesDatabaseRelationship(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(`SELECT id, story, deleted FROM zt_task WHERE id = \? FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}).AddRow(42, 7, "0"))

	err := repo.ValidateTaskOwnership(context.Background(), 42, 8)
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationForbidden {
		t.Fatalf("expected cross-story 403, got %v", err)
	}
	if !strings.Contains(err.Error(), "不属于") {
		t.Fatalf("expected ownership error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTaskOwnership_RejectsMissingAndDeleted(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  []driverValue
	}{
		{name: "missing"},
		{name: "deleted", row: []driverValue{{42, 7, "1"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			repo := NewRepo(db)
			query := mock.ExpectQuery(`SELECT id, story, deleted FROM zt_task WHERE id = \? FOR UPDATE`).WithArgs(42)
			if len(tc.row) == 0 {
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}))
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}).AddRow(tc.row[0][0], tc.row[0][1], tc.row[0][2]))
			}
			var mutationErr *TaskMutationError
			if err := repo.ValidateTaskOwnership(context.Background(), 42, 7); !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationNotFound {
				t.Fatalf("expected missing/deleted task 404, got %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type driverValue []interface{}

func TestCrossStoryTaskEditDelete(t *testing.T) {
	for _, action := range []string{"edit", "delete"} {
		t.Run(action, func(t *testing.T) {
			db, mock := setupMockDB(t)
			repo := NewRepo(db)
			svc := &Service{repo: repo}

			mock.ExpectQuery(`SELECT id, deleted FROM zt_story WHERE id = \? FOR UPDATE`).
				WithArgs(8).
				WillReturnRows(sqlmock.NewRows([]string{"id", "deleted"}).AddRow(8, "0"))
			mock.ExpectQuery(`SELECT id, story, deleted FROM zt_task WHERE id = \? FOR UPDATE`).
				WithArgs(42).
				WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}).AddRow(42, 7, "0"))

			err := svc.applySchedulingTasks(context.Background(), repo, "actor", 8, 100, []SaveSchedulingTask{{
				Action: action, ID: 42,
			}})
			var mutationErr *TaskMutationError
			if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationForbidden {
				t.Fatalf("expected cross-story rejection, got %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMixedBatchBothOrders(t *testing.T) {
	for _, tasks := range [][]SaveSchedulingTask{
		{{Action: "edit", ID: 41}, {Action: "delete", ID: 42}},
		{{Action: "delete", ID: 42}, {Action: "edit", ID: 41}},
	} {
		db, mock := setupMockDB(t)
		repo := NewRepo(db)
		svc := &Service{repo: repo}
		mock.ExpectQuery(`SELECT id, deleted FROM zt_story WHERE id = \? FOR UPDATE`).
			WithArgs(8).
			WillReturnRows(sqlmock.NewRows([]string{"id", "deleted"}).AddRow(8, "0"))
		for _, task := range tasks {
			mock.ExpectQuery(`SELECT id, story, deleted FROM zt_task WHERE id = \? FOR UPDATE`).
				WithArgs(task.ID).
				WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}).AddRow(task.ID, map[uint]uint{41: 8, 42: 7}[task.ID], "0"))
			if task.ID == 42 {
				break
			}
		}
		err := svc.applySchedulingTasks(context.Background(), repo, "actor", 8, 100, tasks)
		var mutationErr *TaskMutationError
		if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationForbidden {
			t.Fatalf("expected mixed batch rejection, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAllowedEditClose(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE .*zt_task.*SET .*name.*WHERE id = \? AND story = \? AND deleted = '0'`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.UpdateTaskForStory(context.Background(), 41, 8, map[string]interface{}{"name": "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	db, mock = setupMockDB(t)
	repo = NewRepo(db)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE .*zt_task.*SET .*assignedTo.*WHERE id = \? AND story = \? AND deleted = '0'`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CloseTaskForStory(context.Background(), 41, 8, "actor"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNoopUpdate(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	if err := repo.UpdateTaskForStory(context.Background(), 41, 8, nil); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOwnershipRace(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE .*zt_task.*SET .*name.*WHERE id = \? AND story = \? AND deleted = '0'`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT id, story, deleted FROM zt_task WHERE id = \? FOR UPDATE`).
		WithArgs(41).
		WillReturnRows(sqlmock.NewRows([]string{"id", "story", "deleted"}).AddRow(41, 7, "0"))
	err := repo.UpdateTaskForStory(context.Background(), 41, 8, map[string]interface{}{"name": "renamed"})
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationConflict {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplySingleSchedulingTask_NewValidatesParentOnly(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	svc := &Service{repo: repo}

	mock.ExpectQuery(`SELECT id, deleted FROM zt_story WHERE id = \? FOR UPDATE`).
		WithArgs(8).
		WillReturnRows(sqlmock.NewRows([]string{"id", "deleted"}).AddRow(8, "0"))

	err := svc.applySchedulingTasks(context.Background(), repo, "actor", 8, 100, []SaveSchedulingTask{{
		Action: "new", ExecutionID: 0,
	}})
	if err == nil || !strings.Contains(err.Error(), "执行 ID 无效") {
		t.Fatalf("expected normal new-task validation path, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStoryForTaskMutation_RejectsDeletedParent(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db)
	mock.ExpectQuery(`SELECT id, deleted FROM zt_story WHERE id = \? FOR UPDATE`).
		WithArgs(8).
		WillReturnRows(sqlmock.NewRows([]string{"id", "deleted"}).AddRow(8, "1"))
	var mutationErr *TaskMutationError
	if err := repo.ValidateStoryForTaskMutation(context.Background(), 8); !errors.As(err, &mutationErr) || mutationErr.Code != taskMutationNotFound {
		t.Fatalf("expected deleted parent 404, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
