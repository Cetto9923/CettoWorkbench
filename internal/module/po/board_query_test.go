package po

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func TestBoardTaskFocusBeforePageAndFinishedOwner(t *testing.T) {
	db, mock := setupMockDB(t)
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) AS total,.*zt_task.finishedBy IN`).
		WillReturnRows(sqlmock.NewRows([]string{"total", "overdue", "blocked"}).AddRow(75, 60, 0))
	mock.ExpectQuery(`(?s)SELECT zt_task.id,.*zt_task.finishedBy.*zt_task.deadline < \?.*ORDER BY zt_task.id DESC LIMIT \? OFFSET \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo", "deadline"}).AddRow(1, "doing", "alice", time.Now().AddDate(0, 0, -3)))
	cols, summary, err := NewRepo(db, db).FindBoardTaskList(context.Background(), BoardTaskReq{TeamgroupID: 1, members: []string{"alice"}, Page: 2, PageSize: 50, Focus: "overdue"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 75 || summary.FilteredTotal != 60 || len(cols[1].Items) != 1 {
		t.Fatalf("bad page: %+v %+v", cols, summary)
	}
	item := boardTaskItem(boardTaskRow{Status: "done", AssignedTo: "bob", FinishedBy: "alice"}, nil, map[string]string{"alice": "Alice"}, false, false)
	if item.OwnerAccount != "alice" || item.Owner != "Alice" {
		t.Fatalf("completed owner: %+v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardEmptyMembersNeverExpandTaskScope(t *testing.T) {
	db, mock := setupMockDB(t)
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) AS total,.*1 = 0`).WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(0))
	mock.ExpectQuery(`(?s)SELECT zt_task.id.*1 = 0`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, summary, err := NewRepo(db, db).FindBoardTaskList(context.Background(), BoardTaskReq{TeamgroupID: 1, Page: 1, PageSize: 50}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 0 {
		t.Fatalf("empty group leaked rows: %+v", summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardDemandAccountAuthorization(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(map[bool]string{false: "nonmember denied", true: "member allowed"}[allowed], func(t *testing.T) {
			db, mock := setupMockDB(t)
			mock.ExpectQuery(`SELECT DISTINCT tg.id, tg.name`).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "group"))
			members := sqlmock.NewRows([]string{"account"}).AddRow("actor")
			if allowed {
				members.AddRow("target")
			}
			mock.ExpectQuery(`SELECT DISTINCT account`).WithArgs(1, "teamgroup").WillReturnRows(members)
			want := errors.New("root query reached")
			if allowed {
				mock.ExpectQuery(`SELECT count\(\*\)`).WillReturnError(want)
			}
			_, err := NewService(NewRepo(db, db), nil, nil, zap.NewNop()).BoardDemand(context.Background(), &model.User{Account: "actor"}, BoardDemandReq{POAccount: "target"})
			if allowed {
				if !errors.Is(err, want) {
					t.Fatalf("authorized read did not reach repo: %v", err)
				}
			} else {
				e, ok := errorx.IsBizError(err)
				if !ok || e.Code != errorx.ErrCodeForbidden {
					t.Fatalf("not forbidden: %v", err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBoardDemandAccountForbiddenHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := setupMockDB(t)
	mock.ExpectQuery(`SELECT DISTINCT tg.id, tg.name`).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{Account: "actor"})
	c.Request = httptest.NewRequest("GET", "/board/demand/items?poAccount=stranger", nil)
	NewBoardHandler(NewService(NewRepo(db, db), nil, nil, zap.NewNop()), zap.NewNop()).BoardDemandItems(c)
	if w.Code != 403 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBoardRootsCountAndPageIncludeSplitParents(t *testing.T) {
	db, mock := setupMockDB(t)
	// Both queries retain the same scope/filter predicates. LIMIT is applied after the UNION.
	rootPattern := `(?s)parent IN \(\?,\?\).*teamGroup = \?.*name LIKE \?.*UNION ALL.*title LIKE \?`
	mock.ExpectQuery(`SELECT count\(\*\).*` + rootPattern).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(201))
	mock.ExpectQuery(`SELECT id, kind.*` + rootPattern + `.*ORDER BY kind ASC, id DESC LIMIT \? OFFSET \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind"}).AddRow(10, "demand"))
	mock.ExpectQuery(`SELECT id, parent, name, status, pri, assignedTo, deadline FROM .zt_demand.`).
		WithArgs(10, "0").WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "name"}).AddRow(10, -1, "split parent"))
	rows, _, total, err := NewRepo(db, db).findBoardRoots(context.Background(), BoardDemandReq{POAccount: "actor", TeamgroupID: 1, Keyword: "target", Page: 2, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if total != 201 || len(rows) != 1 || rows[0].Parent != -1 {
		t.Fatalf("root lost: %+v total=%d", rows, total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
