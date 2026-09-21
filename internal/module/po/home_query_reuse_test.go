package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"workbench/internal/model"
)

func TestHomeFocusReadsReviewIDsOncePerRequest(t *testing.T) {
	db, mock := setupMockDB(t)
	mock.MatchExpectationsInOrder(false)
	// 仅一次基础查询；多出第二次会使 Service 返回 sqlmock 错误。
	mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview`").WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}))
	mock.ExpectQuery(`SELECT count\(\*\)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT id, kind, stage_index, action_rank`).WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "stage_index", "action_rank"}))
	mock.ExpectQuery(`SELECT stage_index, COUNT\(\*\) AS count`).WillReturnRows(sqlmock.NewRows([]string{"stage_index", "count"}))
	svc := NewService(NewRepo(db, db), nil, nil, zap.NewNop())
	resp, err := svc.Demands(context.Background(), &model.User{Account: "alice"}, DemandsReq{Status: "all", Focus: "my_action", ObjectType: "demand", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 0 || len(resp.StageSummary) == 0 {
		t.Fatalf("bad response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
